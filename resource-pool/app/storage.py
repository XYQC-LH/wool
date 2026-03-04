from __future__ import annotations

import base64
import hashlib
import hmac
import os
import time
import uuid
from dataclasses import dataclass
from functools import lru_cache
from typing import Optional, Protocol, Tuple
from urllib.parse import urlparse

from app.config import get_settings


class Storage(Protocol):
    def save_file(self, content: bytes, file_path: str, content_type: str, base_dir: str) -> str:  # noqa: ARG002
        raise NotImplementedError

    def save_file_with_key(
        self, content: bytes, file_path: str, content_type: str, base_dir: str  # noqa: ARG002
    ) -> Tuple[str, str]:
        raise NotImplementedError


def _infer_kind(content_type: str) -> str:
    ct = (content_type or "").lower()
    if ct.startswith("image/"):
        return "img"
    if ct.startswith("video/"):
        return "video"
    return "file"


def _infer_ext(content_type: str, fallback_filename: str) -> str:
    ct = (content_type or "").lower().strip()
    if ct == "image/jpeg":
        return ".jpg"
    if ct == "image/png":
        return ".png"
    if ct == "image/webp":
        return ".webp"
    if ct == "video/mp4":
        return ".mp4"
    if ct == "video/webm":
        return ".webm"
    _, ext = os.path.splitext((fallback_filename or "").strip())
    if ext and 1 <= len(ext) <= 16 and ext[1:].isalnum():
        return ext.lower()
    return ""


def _infer_user_id(file_path: str) -> str:
    # 期望 file_path 形如 "{user_id}/xxx.ext"
    file_path = (file_path or "").replace("\\", "/").lstrip("/")
    if not file_path:
        return "unknown"
    head = file_path.split("/", 1)[0].strip()
    return head or "unknown"


def _build_object_key(user_id: str, kind: str, ext: str) -> str:
    date_path = time.strftime("%Y/%m/%d", time.localtime())
    name = f"{uuid.uuid4().hex}{ext}"
    return f"users/{user_id}/outputs/{kind}/{date_path}/{name}"


def _compute_sig(object_key: str, exp: int, secret: bytes) -> str:
    mac = hmac.new(secret, digestmod=hashlib.sha256)
    mac.update(object_key.encode("utf-8"))
    mac.update(b":")
    mac.update(str(exp).encode("utf-8"))
    return base64.urlsafe_b64encode(mac.digest()).decode("ascii").rstrip("=")


def _parse_public_base_url(raw: str) -> Optional[Tuple[str, bool]]:
    raw = (raw or "").strip()
    if not raw:
        return None
    if not raw.startswith(("http://", "https://")):
        raw = "https://" + raw
    u = urlparse(raw)
    if not u.scheme or not u.netloc:
        return None
    return f"{u.scheme}://{u.netloc}", True


@dataclass(frozen=True)
class LocalStorage:
    base_dir: str
    sign_secret: bytes
    expire_seconds: int

    def save_file(self, content: bytes, file_path: str, content_type: str, base_dir: str) -> str:  # noqa: ARG002
        url, _object_key = self.save_file_with_key(content, file_path, content_type, base_dir=base_dir)
        return url

    def save_file_with_key(
        self, content: bytes, file_path: str, content_type: str, base_dir: str  # noqa: ARG002
    ) -> Tuple[str, str]:
        if content is None:
            raise ValueError("content 不能为空")

        user_id = _infer_user_id(file_path)
        kind = _infer_kind(content_type)
        ext = _infer_ext(content_type, file_path)
        object_key = _build_object_key(user_id, kind, ext)

        full_path = os.path.join(self.base_dir, object_key.replace("/", os.sep))
        os.makedirs(os.path.dirname(full_path), exist_ok=True)
        with open(full_path, "wb") as f:
            f.write(content)

        exp = int(time.time()) + max(1, int(self.expire_seconds))
        sig = _compute_sig(object_key, exp, self.sign_secret)
        return f"/objects/{object_key}?exp={exp}&sig={sig}", object_key


@dataclass(frozen=True)
class OSSStorage:
    endpoint: str
    bucket: str
    access_key_id: str
    access_key_secret: str
    expire_seconds: int
    public_base_url: str = ""

    def save_file(self, content: bytes, file_path: str, content_type: str, base_dir: str) -> str:  # noqa: ARG002
        url, _object_key = self.save_file_with_key(content, file_path, content_type, base_dir=base_dir)
        return url

    def save_file_with_key(
        self, content: bytes, file_path: str, content_type: str, base_dir: str  # noqa: ARG002
    ) -> Tuple[str, str]:
        import oss2  # lazy import

        user_id = _infer_user_id(file_path)
        kind = _infer_kind(content_type)
        ext = _infer_ext(content_type, file_path)
        object_key = _build_object_key(user_id, kind, ext)

        endpoint = (self.endpoint or "").strip()
        if endpoint and not endpoint.startswith(("http://", "https://")):
            endpoint = "https://" + endpoint

        is_cname = False
        parsed = _parse_public_base_url(self.public_base_url)
        if parsed:
            endpoint, is_cname = parsed

        auth = oss2.Auth(self.access_key_id, self.access_key_secret)
        bucket = oss2.Bucket(auth, endpoint, self.bucket, is_cname=is_cname)
        bucket.put_object(object_key, content, headers={"Content-Type": content_type or "application/octet-stream"})

        return bucket.sign_url("GET", object_key, max(1, int(self.expire_seconds))), object_key


@lru_cache(maxsize=1)
def get_storage() -> Storage:
    s = get_settings()

    driver = (s.storage_driver or "").lower().strip()
    if not driver:
        if s.oss_endpoint and s.oss_bucket and s.oss_access_key_id and s.oss_access_key_secret:
            driver = "oss"
        else:
            driver = "local"

    if driver == "oss":
        if not (s.oss_endpoint and s.oss_bucket and s.oss_access_key_id and s.oss_access_key_secret):
            raise ValueError("OSS 配置不完整，无法使用 STORAGE_DRIVER=oss")
        return OSSStorage(
            endpoint=s.oss_endpoint,
            bucket=s.oss_bucket,
            access_key_id=s.oss_access_key_id,
            access_key_secret=s.oss_access_key_secret,
            expire_seconds=s.oss_sign_expire_seconds,
            public_base_url=s.oss_public_base_url,
        )

    os.makedirs(s.storage_local_dir, exist_ok=True)
    return LocalStorage(
        base_dir=s.storage_local_dir,
        sign_secret=s.storage_sign_secret.encode("utf-8"),
        expire_seconds=s.oss_sign_expire_seconds,
    )
