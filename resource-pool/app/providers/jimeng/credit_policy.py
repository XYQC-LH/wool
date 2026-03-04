from __future__ import annotations

from typing import Optional


DEFAULT_IMAGES_PER_REQUEST = 4


_CREDITS_PER_IMAGE_BY_MODEL = {
    "4.5": 3,
    "4.1": 1,
    "4.0": 1,
}


def estimate_required_credit(model: str, *, images_per_request: int = DEFAULT_IMAGES_PER_REQUEST) -> Optional[int]:
    model_key = str(model or "").strip()
    per_image = _CREDITS_PER_IMAGE_BY_MODEL.get(model_key)
    if per_image is None:
        return None

    try:
        count = int(images_per_request)
    except Exception:
        return None
    if count <= 0:
        return None

    return per_image * count

