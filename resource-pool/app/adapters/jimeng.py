"""Jimeng（即梦）图片生成适配器（号池）。"""

from __future__ import annotations

import base64
import copy
import json
import os
import tempfile
import threading
import uuid
from dataclasses import dataclass
from typing import Any, Dict, List, Optional, Tuple, Union
from urllib.parse import urlsplit

import requests

from app.adapters.base import BaseAdapter
from app.config import get_settings
from app.core.generation import GenerationResult, GenerationTaskContext
from app.core.logger import get_logger
from app.providers.jimeng.errors import NoAvailableAccountError, PoolConfigError, UpstreamError
from app.providers.jimeng.pool import JimengPool
from app.providers.jimeng.store import AccountStore
from app.storage import get_storage

logger = get_logger(__name__)


DEFAULT_JIMENG_BASE_CONFIG: Dict[str, Any] = {
    "timeout": {
        "max_wait_time": 180,
        "check_interval": 10,
        "max_retries": 3,
    },
    "params": {
        "default_ratio": "1:1",
        "ratios": {
            "1:1": {"width": 1024, "height": 1024},
            "4:3": {"width": 1152, "height": 864},
            "3:4": {"width": 864, "height": 1152},
            "16:9": {"width": 1344, "height": 768},
            "9:16": {"width": 768, "height": 1344},
        },
        # ApiClient._get_model_key 会校验 model 是否存在于 models 中，否则回退 default_model
        "models": {
            "4.0": {"name": "Jimeng 4.0"},
            "4.1": {"name": "Jimeng 4.1"},
            "4.5": {"name": "Jimeng 4.5"},
        },
        "default_model": "4.0",
    },
}


def _deep_merge(base: Dict[str, Any], override: Dict[str, Any]) -> Dict[str, Any]:
    merged: Dict[str, Any] = copy.deepcopy(base)
    for key, value in (override or {}).items():
        if (
            key in merged
            and isinstance(merged.get(key), dict)
            and isinstance(value, dict)
        ):
            merged[key] = _deep_merge(merged[key], value)
        else:
            merged[key] = copy.deepcopy(value)
    return merged


def _normalize_model_key(model_name: str) -> str:
    raw = str(model_name or "").strip()
    if not raw:
        return ""
    lowered = raw.lower()
    if lowered.startswith("jimeng-"):
        return raw.split("-", 1)[1].strip()
    return raw


def _safe_join(base_dir: str, relative_path: str) -> str:
    base_dir = os.path.abspath(str(base_dir or ""))
    relative_path = str(relative_path or "").replace("\\", "/").lstrip("/")
    if not base_dir:
        raise ValueError("base_dir 不能为空")
    if not relative_path or ".." in relative_path:
        raise ValueError("relative_path 非法")
    full_path = os.path.abspath(os.path.join(base_dir, relative_path))
    if os.path.commonpath([base_dir, full_path]) != base_dir:
        raise ValueError("relative_path 越界")
    return full_path


def _coerce_str_list(value: Any) -> List[str]:
    if value is None:
        return []
    if isinstance(value, (list, tuple)):
        items = []
        for v in value:
            if v is None:
                continue
            items.append(str(v))
        return items
    return [str(value)]


@dataclass(frozen=True)
class JimengAdapterConfig:
    db_path: str
    credit_cache_seconds: int
    account_cooldown_seconds: int
    base_config: Dict[str, Any]

    def fingerprint(self) -> Tuple[str, int, int]:
        try:
            cfg_json = json.dumps(self.base_config, sort_keys=True, ensure_ascii=False, separators=(",", ":"))
        except Exception:
            cfg_json = repr(self.base_config)
        return cfg_json, int(self.credit_cache_seconds), int(self.account_cooldown_seconds)


class _JimengPoolManager:
    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._db_path: str = ""
        self._store: Optional[AccountStore] = None
        self._pool: Optional[JimengPool] = None
        self._enabled_sig: Tuple[int, int] = (0, 0)
        self._cfg_fp: Tuple[str, int, int] = ("", 0, 0)

    def get_pool(self, cfg: JimengAdapterConfig) -> JimengPool:
        with self._lock:
            if not self._store or self._db_path != cfg.db_path:
                self._db_path = cfg.db_path
                self._store = AccountStore(cfg.db_path)
                self._pool = None
                self._enabled_sig = (0, 0)
                self._cfg_fp = ("", 0, 0)

            assert self._store is not None

            enabled_sig = self._store.enabled_signature()
            cfg_fp = cfg.fingerprint()

            if not self._pool:
                self._pool = self._build_pool(cfg, enabled_sig)
                self._enabled_sig = enabled_sig
                self._cfg_fp = cfg_fp
                return self._pool

            # 账号/配置变化时，尽量在空闲时重载，避免丢失锁导致同一账号并发被占用
            if enabled_sig != self._enabled_sig or cfg_fp != self._cfg_fp:
                if self._pool.is_idle():
                    self._pool = self._build_pool(cfg, enabled_sig)
                    self._enabled_sig = enabled_sig
                    self._cfg_fp = cfg_fp
                else:
                    logger.info("[Jimeng] 检测到账号/配置变更，但号池忙碌，暂不重载")

            return self._pool

    def invalidate_if_idle(self) -> bool:
        """
        尝试失效当前缓存的号池（用于账号变更后的手动 reload）。

        - 若号池不存在：视为成功
        - 若号池空闲：清空缓存并返回 True
        - 若号池忙碌：不处理并返回 False
        """
        with self._lock:
            if not self._pool:
                return True

            if not self._pool.is_idle():
                return False

            self._pool = None
            self._enabled_sig = (0, 0)
            self._cfg_fp = ("", 0, 0)
            return True

    def _build_pool(self, cfg: JimengAdapterConfig, enabled_sig: Tuple[int, int]) -> JimengPool:
        assert self._store is not None

        accounts = []
        for record in self._store.list_accounts(include_disabled=False):
            accounts.append(
                {
                    "id": record.id,
                    "sessionid": record.sessionid,
                    "description": record.description,
                    "enabled": record.enabled,
                }
            )

        logger.info("[Jimeng] 初始化号池：db=%s enabled=%s", cfg.db_path, enabled_sig)
        return JimengPool(
            base_config=cfg.base_config,
            accounts=accounts,
            credit_cache_seconds=cfg.credit_cache_seconds,
            account_cooldown_seconds=cfg.account_cooldown_seconds,
        )


_POOL_MANAGER = _JimengPoolManager()


class JimengAdapter(BaseAdapter):
    """Jimeng adapter：文生图 + 参考图。"""

    def __init__(self, settings: Union[dict, None] = None):
        s = get_settings()

        db_path = getattr(s, "jimeng_pool_db_path", "").strip() or "/app/data/jimeng_pool.db"
        credit_cache_seconds = int(getattr(s, "jimeng_credit_cache_seconds", 60) or 60)
        account_cooldown_seconds = int(getattr(s, "jimeng_account_cooldown_seconds", 60) or 60)

        env_cfg_raw = (getattr(s, "jimeng_base_config_json", "") or "").strip()
        base_config = copy.deepcopy(DEFAULT_JIMENG_BASE_CONFIG)
        if env_cfg_raw:
            try:
                env_cfg = json.loads(env_cfg_raw)
                if isinstance(env_cfg, dict):
                    base_config = _deep_merge(base_config, env_cfg)
            except Exception as exc:  # noqa: BLE001
                logger.warning("[Jimeng] 解析 JIMENG_BASE_CONFIG_JSON 失败，将忽略：%s", exc)

        if isinstance(settings, dict):
            db_path = str(settings.get("db_path") or settings.get("jimeng_pool_db_path") or db_path)
            credit_cache_seconds = int(settings.get("credit_cache_seconds") or credit_cache_seconds)
            account_cooldown_seconds = int(settings.get("account_cooldown_seconds") or account_cooldown_seconds)
            override_cfg = settings.get("base_config")
            if isinstance(override_cfg, dict):
                base_config = _deep_merge(base_config, override_cfg)

        self.cfg = JimengAdapterConfig(
            db_path=db_path,
            credit_cache_seconds=max(0, credit_cache_seconds),
            account_cooldown_seconds=max(0, account_cooldown_seconds),
            base_config=base_config,
        )

        self.storage = get_storage()
        self.storage_root = s.storage_root

    def generate(self, ctx: GenerationTaskContext, progress_callback=None) -> GenerationResult:  # noqa: ARG002
        merged = self._merge_params(ctx)
        prompt = str(merged.get("prompt") or "").strip()
        if not prompt:
            return GenerationResult(status="failed", error="prompt 不能为空")

        model_key = _normalize_model_key(ctx.model_name)
        if not model_key:
            model_key = str(self.cfg.base_config.get("params", {}).get("default_model") or "4.0")

        ratio = str(merged.get("aspect_ratio") or merged.get("ratio") or "").strip()
        if not ratio or ratio.lower() == "auto":
            ratio = str(self.cfg.base_config.get("params", {}).get("default_ratio") or "1:1")

        seed = merged.get("seed", -1)
        try:
            seed = int(seed) if seed is not None else -1
        except Exception:
            seed = -1

        preferred_account_id = merged.get("preferred_account_id", merged.get("account_id"))
        if preferred_account_id is not None:
            try:
                preferred_account_id = int(preferred_account_id)
            except Exception:
                preferred_account_id = None

        reference_inputs: List[str] = []
        reference_inputs.extend(_coerce_str_list(merged.get("urls")))
        if merged.get("image"):
            reference_inputs.insert(0, str(merged.get("image")))
        reference_inputs = [v.strip() for v in reference_inputs if str(v).strip()]

        try:
            pool = _POOL_MANAGER.get_pool(self.cfg)
        except PoolConfigError as exc:
            return GenerationResult(status="failed", error=str(exc))
        except Exception as exc:  # noqa: BLE001
            return GenerationResult(status="failed", error=f"Jimeng 号池初始化失败: {exc}")

        try:
            with tempfile.TemporaryDirectory(prefix="wool-jimeng-") as tmp_dir:
                if reference_inputs:
                    reference_paths = self._materialize_reference_images(reference_inputs, tmp_dir)
                    resp = pool.generate_i2i_reference(
                        prompt=prompt,
                        model=model_key,
                        ratio=ratio,
                        reference_image_paths=reference_paths,
                        preferred_account_id=preferred_account_id,
                    )
                else:
                    resp = pool.generate_t2i(
                        prompt=prompt,
                        model=model_key,
                        ratio=ratio,
                        seed=seed,
                        preferred_account_id=preferred_account_id,
                    )
        except NoAvailableAccountError as exc:
            return GenerationResult(status="failed", error=str(exc))
        except PoolConfigError as exc:
            return GenerationResult(status="failed", error=str(exc))
        except UpstreamError as exc:
            return GenerationResult(status="failed", error=str(exc))
        except Exception as exc:  # noqa: BLE001
            return GenerationResult(status="failed", error=f"Jimeng 调用失败: {exc}")

        remote_urls = resp.get("urls") if isinstance(resp, dict) else None
        if not remote_urls:
            return GenerationResult(status="failed", error=f"Jimeng 未返回图片URL: {resp}")

        stored_urls: List[str] = []
        stored_object_keys: List[str] = []
        for url in _coerce_str_list(remote_urls):
            url = str(url or "").strip()
            if not url:
                continue
            stored = self._download_and_store(url, ctx)
            if stored:
                stored_url, object_key = stored
                stored_urls.append(stored_url)
                stored_object_keys.append(object_key)

        if not stored_urls:
            return GenerationResult(status="failed", error="Jimeng 图片生成完成，但下载/存储失败")

        result_data: Dict[str, Any] = {}
        if isinstance(resp, dict):
            result_data = dict(resp)
        result_data["remote_urls"] = _coerce_str_list(remote_urls)
        result_data["urls"] = list(stored_urls)
        result_data["object_keys"] = list(stored_object_keys)

        return GenerationResult(
            status="completed",
            progress=1.0,
            message="Jimeng generation completed",
            result_url=stored_urls[0],
            result_object_key=stored_object_keys[0] if stored_object_keys else None,
            result_data=result_data,
        )

    def _materialize_reference_images(self, inputs: List[str], tmp_dir: str) -> List[str]:
        paths: List[str] = []
        for idx, raw in enumerate(inputs):
            if len(paths) >= 6:
                break
            try:
                p = self._materialize_reference_image(raw, tmp_dir, idx)
            except Exception as exc:  # noqa: BLE001
                logger.warning("[Jimeng] 处理参考图失败：%s (%s)", raw, exc)
                continue
            if p:
                paths.append(p)
        if not paths:
            raise ValueError("参考图为空或不可用")
        return paths

    def _materialize_reference_image(self, raw: str, tmp_dir: str, idx: int) -> Optional[str]:
        raw = str(raw or "").strip()
        if not raw:
            return None

        if raw.startswith("data:"):
            return self._write_data_url_to_file(raw, tmp_dir, idx)

        if raw.startswith("/objects/"):
            return self._copy_local_object_url_to_file(raw, tmp_dir, idx)

        if raw.startswith(("http://", "https://")):
            return self._download_url_to_file(raw, tmp_dir, idx)

        if os.path.isfile(raw):
            return os.path.abspath(raw)

        # 尝试兼容 storage_root 下的相对路径
        if self.storage_root and not os.path.isabs(raw):
            candidate = os.path.join(self.storage_root, raw)
            if os.path.isfile(candidate):
                return os.path.abspath(candidate)

        raise ValueError("不支持的参考图输入")

    def _write_data_url_to_file(self, data_url: str, tmp_dir: str, idx: int) -> str:
        # data:{mime};base64,xxxx
        head, _, b64 = data_url.partition(",")
        if not b64:
            raise ValueError("data url 缺少 base64 内容")

        ext = ".png"
        if ";base64" in head:
            mime = head.split(";", 1)[0].removeprefix("data:").strip().lower()
            if mime == "image/jpeg":
                ext = ".jpg"
            elif mime == "image/webp":
                ext = ".webp"
            elif mime == "image/png":
                ext = ".png"

        content = base64.b64decode(b64, validate=False)
        file_path = os.path.join(tmp_dir, f"ref_{idx:02d}{ext}")
        with open(file_path, "wb") as f:
            f.write(content)
        return file_path

    def _copy_local_object_url_to_file(self, url: str, tmp_dir: str, idx: int) -> str:
        # /objects/{object_key}?exp=...&sig=...
        parsed = urlsplit(url)
        path = parsed.path or ""
        object_key = path.removeprefix("/objects/").lstrip("/")
        if not object_key:
            raise ValueError("object_key 为空")

        s = get_settings()
        src = _safe_join(s.storage_local_dir, object_key)
        if not os.path.isfile(src):
            raise ValueError("本地对象不存在")

        _, ext = os.path.splitext(src)
        if not ext:
            ext = ".bin"

        dst = os.path.join(tmp_dir, f"ref_{idx:02d}{ext}")
        with open(src, "rb") as rf, open(dst, "wb") as wf:
            wf.write(rf.read())
        return dst

    def _download_url_to_file(self, url: str, tmp_dir: str, idx: int) -> str:
        resp = requests.get(url, timeout=60)
        resp.raise_for_status()

        content_type = (resp.headers.get("Content-Type") or "").split(";", 1)[0].strip().lower()
        ext = ".png"
        if content_type == "image/jpeg":
            ext = ".jpg"
        elif content_type == "image/webp":
            ext = ".webp"
        elif content_type == "image/png":
            ext = ".png"

        file_path = os.path.join(tmp_dir, f"ref_{idx:02d}{ext}")
        with open(file_path, "wb") as f:
            f.write(resp.content)
        return file_path

    def _download_and_store(self, url: str, ctx: GenerationTaskContext) -> Optional[Tuple[str, str]]:
        try:
            resp = requests.get(url, timeout=120)
            resp.raise_for_status()
        except Exception as exc:  # noqa: BLE001
            logger.warning("[Jimeng] 下载图片失败: %s (%s)", url, exc)
            return None

        content_type = (resp.headers.get("Content-Type") or "").split(";", 1)[0].strip()
        if not content_type:
            content_type = "image/png"

        prefix = self._resolve_output_filename_prefix(ctx, "jimeng")
        filename = f"{prefix}_{ctx.model_name}_{ctx.user_id}_{uuid.uuid4().hex}"
        file_path = f"{ctx.user_id}/{filename}"

        try:
            public_url, object_key = self.storage.save_file_with_key(
                resp.content,
                file_path=file_path,
                content_type=content_type,
                base_dir="user/outputs",
            )
            return public_url, object_key
        except Exception as exc:  # noqa: BLE001
            logger.warning("[Jimeng] 存储图片失败: %s", exc)
            return None
