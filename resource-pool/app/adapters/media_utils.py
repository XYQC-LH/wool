from __future__ import annotations

import base64
import mimetypes
import os
from typing import Any, Iterable


def maybe_convert_image_input(value: Any, storage_root: str = "") -> Any:  # noqa: ANN401
    """
    兼容多种输入：
    - http(s) URL / data URL：原样返回
    - 本地文件路径：读取并转为 data URL
    - list/tuple/dict：递归处理
    """
    if value is None:
        return None

    if isinstance(value, (list, tuple)):
        return [maybe_convert_image_input(v, storage_root=storage_root) for v in value]

    if isinstance(value, dict):
        return {k: maybe_convert_image_input(v, storage_root=storage_root) for k, v in value.items()}

    if not isinstance(value, str):
        return value

    raw = value.strip()
    if not raw:
        return raw

    if raw.startswith(("http://", "https://", "data:")):
        return raw

    # 尝试按本地文件路径处理
    candidate_paths: Iterable[str]
    if storage_root and not os.path.isabs(raw):
        candidate_paths = (os.path.join(storage_root, raw), raw)
    else:
        candidate_paths = (raw,)

    for p in candidate_paths:
        if os.path.isfile(p):
            mime, _ = mimetypes.guess_type(p)
            if not mime:
                mime = "application/octet-stream"
            with open(p, "rb") as f:
                data = f.read()
            b64 = base64.b64encode(data).decode("ascii")
            return f"data:{mime};base64,{b64}"

    return raw

