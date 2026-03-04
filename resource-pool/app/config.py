import os
from dataclasses import dataclass
from functools import lru_cache


@dataclass(frozen=True)
class Settings:
    # adapters
    sora2_api_host: str
    sora2_api_key: str

    banana_api_host: str
    banana_api_key: str

    seedream_base_url: str
    seedream_api_key: str

    # jimeng pool
    jimeng_pool_db_path: str
    jimeng_credit_cache_seconds: int
    jimeng_account_cooldown_seconds: int
    jimeng_base_config_json: str

    # storage (align with Go backend env)
    storage_driver: str
    storage_local_dir: str
    storage_sign_secret: str
    jwt_secret: str
    oss_endpoint: str
    oss_bucket: str
    oss_access_key_id: str
    oss_access_key_secret: str
    oss_public_base_url: str
    oss_sign_expire_seconds: int

    @property
    def storage_root(self) -> str:
        # 兼容适配器的入参：用于判定本地文件/相对路径等
        return self.storage_local_dir


def _getenv(key: str, default: str = "") -> str:
    return (os.getenv(key) or default).strip()


def _getenv_int(key: str, default: int) -> int:
    raw = (os.getenv(key) or "").strip()
    if not raw:
        return default
    try:
        return int(raw)
    except Exception:
        return default


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    jwt_secret = _getenv("JWT_SECRET", "change-me")
    storage_sign_secret = _getenv("STORAGE_SIGN_SECRET", "")
    if not storage_sign_secret:
        storage_sign_secret = jwt_secret

    return Settings(
        sora2_api_host=_getenv("SORA2_API_HOST", ""),
        sora2_api_key=_getenv("SORA2_API_KEY", ""),
        banana_api_host=_getenv("BANANA_API_HOST", ""),
        banana_api_key=_getenv("BANANA_API_KEY", ""),
        seedream_base_url=_getenv("SEEDREAM_BASE_URL", ""),
        seedream_api_key=_getenv("SEEDREAM_API_KEY", ""),
        jimeng_pool_db_path=_getenv("JIMENG_POOL_DB_PATH", "/app/data/jimeng_pool.db"),
        jimeng_credit_cache_seconds=_getenv_int("JIMENG_CREDIT_CACHE_SECONDS", 60),
        jimeng_account_cooldown_seconds=_getenv_int("JIMENG_ACCOUNT_COOLDOWN_SECONDS", 60),
        jimeng_base_config_json=_getenv("JIMENG_BASE_CONFIG_JSON", ""),
        storage_driver=_getenv("STORAGE_DRIVER", ""),
        storage_local_dir=_getenv("STORAGE_LOCAL_DIR", "/app/data/objects"),
        storage_sign_secret=storage_sign_secret,
        jwt_secret=jwt_secret,
        oss_endpoint=_getenv("OSS_ENDPOINT", ""),
        oss_bucket=_getenv("OSS_BUCKET", ""),
        oss_access_key_id=_getenv("OSS_ACCESS_KEY_ID", ""),
        oss_access_key_secret=_getenv("OSS_ACCESS_KEY_SECRET", ""),
        oss_public_base_url=_getenv("OSS_PUBLIC_BASE_URL", "").rstrip("/"),
        oss_sign_expire_seconds=_getenv_int("OSS_SIGN_EXPIRE_SECONDS", 900),
    )
