"""Banana 图片生成适配器。"""

import time
import uuid
from typing import Any, Dict, Optional, Tuple, Union

import requests

from app.adapters.base import BaseAdapter
from app.adapters.media_utils import maybe_convert_image_input
from app.config import get_settings
from app.core.generation import GenerationResult, GenerationTaskContext
from app.core.logger import get_logger
from app.storage import get_storage

logger = get_logger(__name__)


class BananaAdapter(BaseAdapter):
    """Nano Banana adapter for image generation."""

    endpoint = "/v1/draw/nano-banana"
    result_path = "/v1/draw/result"
    supported_models = {
        "nano-banana-fast",
        "nano-banana",
        "nano-banana-pro",
        "nano-banana-pro-vt",
        "nano-banana-pro-cl",
        "nano-banana-pro-vip",
        "nano-banana-pro-4k-vip",
    }

    # 内部参数名 → API参数名 的映射
    PARAM_MAPPING = {
        "aspect_ratio": "aspectRatio",
        "image_size": "imageSize",
        "web_hook": "webHook",
        "shut_progress": "shutProgress",
    }

    def __init__(self, settings: Union[dict, None] = None):
        base_settings = get_settings()
        self.api_host = base_settings.banana_api_host
        self.api_key = base_settings.banana_api_key

        if isinstance(settings, dict):
            self.api_host = settings.get("base_url") or settings.get("api_host") or self.api_host
            self.api_key = settings.get("api_key") or settings.get("banana_api_key") or self.api_key

        self.base_url = self._normalize_base_url(self.api_host)
        # 强制生产模式 - 删除测试模式逻辑
        self.test_mode = False

        # 图片存储配置 - 使用统一存储模块
        self.storage_root = base_settings.storage_root
        self.storage = get_storage()

    @staticmethod
    def _normalize_base_url(host: str) -> str:
        if not host:
            return ""
        if host.startswith("http://") or host.startswith("https://"):
            return host.rstrip("/")
        return f"https://{host}".rstrip("/")

    def generate(self, ctx: GenerationTaskContext, progress_callback=None) -> GenerationResult:
        # 强制生产模式 - 直接调用真实API
        if not self.base_url:
            return GenerationResult(status="failed", error="Banana API host not configured")
        if not self.api_key:
            return GenerationResult(status="failed", error="Banana API key not configured")
        if ctx.model_name not in self.supported_models:
            return GenerationResult(status="failed", error=f"Unsupported Banana model: {ctx.model_name}")

        payload = self._build_payload(ctx)
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self.api_key}",
        }

        try:
            resp = requests.post(
                f"{self.base_url}{self.endpoint}",
                json=payload,
                headers=headers,
                timeout=15,
            )
        except Exception as exc:  # noqa: BLE001
            return GenerationResult(status="failed", error=f"Banana request error: {exc}")

        if resp.status_code != 200:
            return GenerationResult(status="failed", error=f"Banana request failed: {resp.status_code} {resp.text}")

        data = resp.json()
        task_id = data.get("data", {}).get("id") if data.get("code") == 0 else None
        if not task_id:
            return GenerationResult(
                status="failed",
                error=data.get("msg") or "Banana did not return task id",
                result_data=data,
            )

        return self._poll_result(task_id, headers=headers, ctx=ctx, progress_callback=progress_callback)

    def _build_payload(self, ctx: GenerationTaskContext) -> Dict[str, Any]:
        params = self._merge_params(ctx)

        # 兼容：网关侧可能只传 resolution（1K/2K/4K），而该上游使用 imageSize 字段
        if "image_size" not in params:
            resolution = params.get("resolution")
            if resolution:
                params["image_size"] = resolution

        # 统一使用映射表转换参数名
        payload: Dict[str, Any] = {
            "model": ctx.model_name,
            "prompt": params.pop("prompt", ""),
        }

        # 应用参数映射：snake_case → camelCase
        for snake_key, camel_key in self.PARAM_MAPPING.items():
            if snake_key in params:
                payload[camel_key] = params.pop(snake_key)

        # 处理 urls 字段
        if "urls" in params:
            payload["urls"] = self._convert_local_urls(params.pop("urls"))

        # 设置默认值
        payload.setdefault("aspectRatio", "auto")
        payload.setdefault("imageSize", "1K")
        payload.setdefault("webHook", "-1")
        payload.setdefault("shutProgress", False)

        return payload

    def _convert_local_urls(self, urls_value: Any) -> Any:
        return maybe_convert_image_input(urls_value, storage_root=self.storage_root)

    # 删除测试��式方法 - 强制使用生产API
    # def _generate_test_result(self, ctx: GenerationTaskContext) -> GenerationResult:
    #     import uuid
    #     time.sleep(1)
    #     mock_task_id = str(uuid.uuid4())[:8]
    #     mock_image_url = f"https://example.com/banana-test-{mock_task_id}.png"
    #     mock_metadata = {
    #         "task_id": mock_task_id,
    #         "model": ctx.model_name,
    #         "prompt": ctx.user_inputs.get("prompt", "test prompt"),
    #         "output_type": "image",
    #         "width": 512,
    #         "height": 512,
    #         "steps": 20,
    #         "seed": int(time.time()) % 10000,
    #         "created_at": time.strftime("%Y-%m-%d %H:%M:%S"),
    #         "test_mode": True,
    #     }
    #     return GenerationResult(status="completed", result_url=mock_image_url, result_data=mock_metadata)

    def _poll_result(
        self,
        task_id: str,
        *,
        headers: Dict[str, str],
        poll_interval: float = 5.0,
        timeout: float = 600.0,
        ctx: Optional[GenerationTaskContext] = None,
        progress_callback=None,
    ) -> GenerationResult:
        start = time.time()
        while time.time() - start < timeout:
            try:
                resp = requests.post(
                    f"{self.base_url}{self.result_path}",
                    json={"id": task_id},
                    headers=headers,
                    timeout=10,
                )
            except Exception as exc:  # noqa: BLE001
                return GenerationResult(status="failed", error=f"Query error: {exc}")

            if resp.status_code != 200:
                return GenerationResult(status="failed", error=f"Query failed: {resp.status_code} {resp.text}")

            body = resp.json()
            data: Optional[Dict[str, Any]] = body.get("data")
            if body.get("code") != 0 or not data:
                time.sleep(poll_interval)
                continue

            status = data.get("status", "running")
            progress_raw = data.get("progress", 0)
            progress = float(progress_raw) / 100 if progress_raw is not None else 0.0
            result_url = None
            results = data.get("results") or []
            if results:
                result_url = results[0].get("url")

            if status == "succeeded":
                if progress_callback:
                    try:
                        progress_callback(1.0, data=data)
                    except Exception as exc:  # noqa: BLE001
                        logger.warning("Progress callback failed: %s", exc)
                # 如果有图片URL，下载并存储到本地
                if result_url and ctx:
                    stored = self._download_and_store_image(result_url, ctx)
                    if stored:
                        local_image_url, object_key = stored
                        return GenerationResult(
                            status="completed",
                            progress=1.0,
                            message="Banana generation completed",
                            result_url=local_image_url,
                            result_object_key=object_key,
                            result_data=data,
                        )
                    else:
                        return GenerationResult(
                            status="failed",
                            error="Image generation completed but failed to download image"
                        )
                else:
                    # 测试模式或没有上下文时，直接返回远程URL
                    return GenerationResult(
                        status="completed",
                        progress=1.0,
                        message="Banana generation completed",
                        result_url=result_url,
                        result_data=data,
                    )
            if status == "failed":
                return GenerationResult(
                    status="failed",
                    progress=progress,
                    error=data.get("error") or data.get("failure_reason") or "task failed",
                    result_data=data,
                )

            # 运行中/队列中/未知状态，回调进度并继续轮询
            if progress_callback:
                try:
                    progress_callback(progress, data=data)
                except Exception as exc:  # noqa: BLE001
                    logger.warning("Progress callback failed: %s", exc)

            time.sleep(poll_interval)

        return GenerationResult(status="failed", error="Banana polling timeout")

    def _download_and_store_image(self, image_url: str, ctx: GenerationTaskContext) -> Optional[Tuple[str, str]]:
        """下载图片并存储（支持本地/OSS）"""
        try:
            logger.info("Downloading image from: %s", image_url)

            # 发送下载请求
            response = requests.get(image_url, stream=True, timeout=30)
            response.raise_for_status()

            # 生成唯一的文件名
            file_id = str(uuid.uuid4())
            # 根据URL确定文件扩展名
            file_ext = "png"  # 默认为png
            if image_url.lower().endswith('.jpg') or image_url.lower().endswith('.jpeg'):
                file_ext = "jpg"
            elif image_url.lower().endswith('.webp'):
                file_ext = "webp"

            prefix = self._resolve_output_filename_prefix(ctx, "banana")
            if prefix == "tools":
                filename = f"{prefix}_{ctx.user_id}_{file_id}.{file_ext}"
            else:
                filename = f"{prefix}_{ctx.model_name}_{ctx.user_id}_{file_id}.{file_ext}"

            # 使用统一存储模块保存文件（支持本地/OSS）
            content_type = f"image/{file_ext}"
            if file_ext == "jpg":
                content_type = "image/jpeg"
            
            file_path = f"{ctx.user_id}/{filename}"
            public_url, object_key = self.storage.save_file_with_key(
                response.content,
                file_path,
                content_type,
                base_dir="user/images",
            )

            logger.info("Image stored successfully: %s", public_url)
            return public_url, object_key

        except Exception as e:
            logger.error("Failed to download and store image: %s", str(e))
            return None
