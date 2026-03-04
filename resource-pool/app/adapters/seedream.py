"""Seedream 图片生成适配器。"""

import base64
import io
import logging
import math
import uuid
from typing import Any, Dict, List, Optional, Tuple, Union

import requests
from PIL import Image

from app.adapters.base import BaseAdapter
from app.adapters.media_utils import maybe_convert_image_input
from app.config import get_settings
from app.core.generation import GenerationResult, GenerationTaskContext
from app.storage import get_storage

logger = logging.getLogger(__name__)

# 分辨率级别对应的最长边像素值
RESOLUTION_MAX_EDGE = {
    "1K": 1024,
    "2K": 2048,
    "4K": 4096,
}

# 各模型支持的分辨率级别
MODEL_SUPPORTED_RESOLUTIONS = {
    "seedream-4": ["1K", "2K", "4K"],
    "seedream-4.5": ["2K", "4K"],
}

# 标准画幅比例列表
STANDARD_ASPECT_RATIOS = [
    ("1:1", 1, 1),
    ("16:9", 16, 9),
    ("9:16", 9, 16),
    ("4:3", 4, 3),
    ("3:4", 3, 4),
    ("3:2", 3, 2),
    ("2:3", 2, 3),
    ("21:9", 21, 9),
]


class SeedreamAdapter(BaseAdapter):
    """Seedream (Ark) image generation adapter."""

    supported_models = {"seedream-4", "seedream-4.5"}

    def __init__(self, settings: Union[dict, None] = None):
        base_settings = get_settings()
        default_base_url = "https://ark.cn-beijing.volces.com/api/v3/images/generations"
        self.api_key = getattr(base_settings, "seedream_api_key", None)
        self.base_url = (getattr(base_settings, "seedream_base_url", default_base_url) or default_base_url).rstrip("/")

        if isinstance(settings, dict):
            self.base_url = settings.get("base_url") or self.base_url
            self.api_key = settings.get("api_key") or self.api_key

        # storage - 使用统一存储模块
        self.storage_root = base_settings.storage_root
        self.storage = get_storage()

    def _resolve_endpoint(self) -> str:
        # 允许 base_url 直接为完整的 generations 路径
        if self.base_url.endswith("/images/generations"):
            return self.base_url
        return f"{self.base_url}/images/generations"

    @staticmethod
    def _map_aspect_ratio_to_size(ratio: str) -> str:
        mapping = {
            "1:1": "1024x1024",
            "16:9": "1280x720",
            "9:16": "720x1280",
            "4:3": "1024x768",
            "3:4": "768x1024",
        }
        return mapping.get(ratio, "1024x1024")

    @staticmethod
    def _get_image_dimensions(image_input: Union[str, List[str], None]) -> Optional[Tuple[int, int]]:
        """
        从图片输入获取尺寸（宽, 高）
        支持 base64 字符串、URL 或列表形式
        """
        if not image_input:
            return None

        # 获取第一张图片
        if isinstance(image_input, list):
            if not image_input:
                return None
            image_input = image_input[0]

        if not isinstance(image_input, str):
            return None

        try:
            if image_input.startswith("data:image"):
                # Base64 格式: data:image/png;base64,xxxxx
                header, data = image_input.split(",", 1)
                image_bytes = base64.b64decode(data)
                img = Image.open(io.BytesIO(image_bytes))
                return img.size  # (width, height)
            elif image_input.startswith(("http://", "https://")):
                # URL 格式：需要下载图片获取尺寸
                resp = requests.get(image_input, stream=True, timeout=30)
                resp.raise_for_status()
                img = Image.open(io.BytesIO(resp.content))
                return img.size
        except Exception:
            return None

        return None

    @staticmethod
    def _detect_aspect_ratio(
        image_input: Optional[Union[str, List[str]]]
    ) -> Tuple[str, float, float]:
        """
        检测图片的画幅比例

        Args:
            image_input: 图片输入（base64/URL/列表）

        Returns:
            (aspect_ratio_label, width_ratio, height_ratio)
            如 ("16:9", 16, 9) 或 ("custom", 1920, 1080)
        """
        # 获取图片尺寸
        dimensions = SeedreamAdapter._get_image_dimensions(image_input)
        if not dimensions:
            return ("1:1", 1.0, 1.0)  # 默认

        width, height = dimensions

        # 计算图片的实际比例
        actual_ratio = width / height

        # 找到最接近的标准比例
        best_match = None
        min_diff = float('inf')

        for label, w, h in STANDARD_ASPECT_RATIOS:
            standard_ratio = w / h
            diff = abs(actual_ratio - standard_ratio)
            if diff < min_diff:
                min_diff = diff
                best_match = (label, float(w), float(h))

        # 如果差异太大（超过 5%），使用原始比例
        if min_diff > 0.05:
            # 简化比例
            gcd = math.gcd(width, height)
            return ("custom", float(width // gcd), float(height // gcd))

        return best_match

    @staticmethod
    def _calculate_approximate_size(
        resolution: str,
        width_ratio: float,
        height_ratio: float
    ) -> str:
        """
        根据分辨率级别和画幅比例计算趋近的像素尺寸

        Args:
            resolution: 分辨率级别 (1K/2K/4K)
            width_ratio: 宽度比例
            height_ratio: 高度比例

        Returns:
            尺寸字符串，如 "2048x1152"
        """
        # 分辨率对应的最大边长
        max_edge = RESOLUTION_MAX_EDGE.get(resolution, 2048)

        # 计算基于比例的尺寸
        if width_ratio >= height_ratio:
            # 横向或正方形
            width = max_edge
            height = int(max_edge * height_ratio / width_ratio)
        else:
            # 纵向
            height = max_edge
            width = int(max_edge * width_ratio / height_ratio)

        # 确保尺寸是偶数（某些编码器要求）
        width = width - (width % 2)
        height = height - (height % 2)

        # 确保不超过最大限制
        width = min(width, 4096)
        height = min(height, 4096)

        return f"{width}x{height}"

    @staticmethod
    def _map_resolution_and_ratio_to_size(resolution: str, aspect_ratio: str) -> str:
        """将分辨率级别和画幅比例映射到具体像素值"""
        # 如果直接传 1K/2K/4K，让 API 自己处理
        if resolution in ("1K", "2K", "4K") and not aspect_ratio:
            return resolution

        # 根据分辨率和比例计算具体像素值
        size_mapping = {
            # 1K 分辨率 (约 1M 像素)
            "1K": {
                "1:1": "1024x1024",
                "16:9": "1280x720",
                "9:16": "720x1280",
                "4:3": "1152x864",
                "3:4": "864x1152",
                "3:2": "1248x832",
                "2:3": "832x1248",
                "21:9": "1512x648",
            },
            # 2K 分辨率 (约 4M 像素)
            "2K": {
                "1:1": "2048x2048",
                "16:9": "2560x1440",
                "9:16": "1440x2560",
                "4:3": "2304x1728",
                "3:4": "1728x2304",
                "3:2": "2496x1664",
                "2:3": "1664x2496",
                "21:9": "3024x1296",
            },
            # 4K 分辨率 (约 16M 像素)
            "4K": {
                "1:1": "4096x4096",
                "16:9": "3840x2160",
                "9:16": "2160x3840",
                "4:3": "3648x2736",
                "3:4": "2736x3648",
                "3:2": "3968x2645",
                "2:3": "2645x3968",
                "21:9": "4032x1728",
            },
        }

        if resolution in size_mapping and aspect_ratio in size_mapping[resolution]:
            return size_mapping[resolution][aspect_ratio]

        # 默认返回 2K 1:1
        return "2048x2048"

    def _build_payload(self, ctx: GenerationTaskContext) -> Dict[str, Any]:
        merged = self._merge_params(ctx)
        prompt = merged.get("prompt", "") or ""

        # 获取参数
        resolution = merged.get("resolution") or "2K"
        aspect_ratio = merged.get("aspect_ratio") or "1:1"
        image_input = merged.get("image") or merged.get("images") or merged.get("urls")

        # 处理 auto 画幅比例
        if aspect_ratio == "auto":
            if image_input:
                # 有参考图，检测画幅比例
                ratio_label, width_ratio, height_ratio = self._detect_aspect_ratio(image_input)
            else:
                # 无参考图，使用默认 1:1
                ratio_label, width_ratio, height_ratio = "1:1", 1.0, 1.0
        else:
            # 用户指定的画幅比例
            ratio_label = aspect_ratio
            ratio_parts = aspect_ratio.split(":")
            width_ratio = float(ratio_parts[0])
            height_ratio = float(ratio_parts[1])

        # 计算最终尺寸
        # 优先使用预定义的映射表（标准比例），否则使用动态计算
        if ratio_label != "custom" and ratio_label in [r[0] for r in STANDARD_ASPECT_RATIOS]:
            size = self._map_resolution_and_ratio_to_size(resolution, ratio_label)
        else:
            # 非标准比例，使用动态计算
            size = self._calculate_approximate_size(resolution, width_ratio, height_ratio)

        # 映射内部模型名称到 Ark API 模型 ID
        model_mapping = {
            "seedream-4": "doubao-seedream-4-0-250828",
            "seedream-4.5": "doubao-seedream-4-5-251128",
        }
        api_model = model_mapping.get(ctx.model_name, ctx.model_name)

        payload: Dict[str, Any] = {
            "model": api_model,
            "prompt": prompt,
            "response_format": merged.get("response_format", "url"),
            "watermark": merged.get("watermark", False),
            "seed": merged.get("seed", -1),
            "stream": False,  # 当前不启用流式
            "size": size,
        }

        # 参考图/图片
        image_input = merged.get("image") or merged.get("images") or merged.get("urls")
        if image_input:
            payload["image"] = maybe_convert_image_input(image_input, storage_root=self.storage_root)

        # 连续生成
        if merged.get("sequential_image_generation") is not None:
            payload["sequential_image_generation"] = merged.get("sequential_image_generation")
            max_images = merged.get("max_images")
            if max_images:
                payload["sequential_image_generation_options"] = {"max_images": int(max_images)}

        # 优化模式
        if merged.get("optimize_mode"):
            payload["optimize_prompt_options"] = {"mode": merged.get("optimize_mode")}

        return payload

    def _download_and_store(self, url: str, ctx: GenerationTaskContext) -> Optional[Tuple[str, str]]:
        """下载图片并存储（支持本地/OSS）"""
        try:
            logger.info("Downloading image from: %s", url)
            resp = requests.get(url, stream=True, timeout=60)
            resp.raise_for_status()

            file_id = str(uuid.uuid4())
            ext = "png"
            lower = url.lower()
            if lower.endswith(".jpg") or lower.endswith(".jpeg"):
                ext = "jpg"
            elif lower.endswith(".webp"):
                ext = "webp"

            prefix = self._resolve_output_filename_prefix(ctx, "seedream")
            filename = f"{prefix}_{ctx.user_id}_{file_id}.{ext}"

            # 使用统一存储模块保存文件（支持本地/OSS）
            content_type = f"image/{ext}"
            if ext == "jpg":
                content_type = "image/jpeg"

            file_path = f"{ctx.user_id}/{filename}"
            public_url, object_key = self.storage.save_file_with_key(
                resp.content,
                file_path,
                content_type,
                base_dir="user/images"
            )

            logger.info("Image stored successfully: %s", public_url)
            return public_url, object_key
        except Exception as e:
            logger.error("Failed to download and store image: %s", str(e))
            return None

    def _save_base64_image(self, b64_data: str, ctx: GenerationTaskContext) -> Optional[Tuple[str, str]]:
        """保存 base64 图片（支持本地/OSS）"""
        try:
            file_id = str(uuid.uuid4())
            prefix = self._resolve_output_filename_prefix(ctx, "seedream")
            filename = f"{prefix}_{ctx.user_id}_{file_id}.png"

            # 使用统一存储模块保存文件（支持本地/OSS）
            file_path = f"{ctx.user_id}/{filename}"
            public_url, object_key = self.storage.save_file_with_key(
                base64.b64decode(b64_data),
                file_path,
                "image/png",
                base_dir="user/images"
            )

            logger.info("Base64 image stored successfully: %s", public_url)
            return public_url, object_key
        except Exception as e:
            logger.error("Failed to save base64 image: %s", str(e))
            return None

    def generate(self, ctx: GenerationTaskContext, progress_callback=None) -> GenerationResult:
        if not self.api_key:
            return GenerationResult(status="failed", error="Seedream API key not configured")
        if ctx.model_name not in self.supported_models:
            return GenerationResult(status="failed", error=f"Unsupported Seedream model: {ctx.model_name}")

        payload = self._build_payload(ctx)
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self.api_key}",
        }

        try:
            resp = requests.post(self._resolve_endpoint(), json=payload, headers=headers, timeout=120)
            if not resp.ok:
                return GenerationResult(status="failed", error=f"HTTP {resp.status_code}: {resp.text}")

            data = resp.json() or {}
            items = data.get("data") or []
            if not items:
                return GenerationResult(status="failed", error="No data returned from Seedream")

            public_url: Optional[str] = None
            object_key: Optional[str] = None
            # 取第一项的 url 或 b64_json
            first = items[0]
            url = first.get("url") if isinstance(first, dict) else getattr(first, "url", None)
            b64 = first.get("b64_json") if isinstance(first, dict) else getattr(first, "b64_json", None)

            if url:
                stored = self._download_and_store(url, ctx)
                if stored:
                    public_url, object_key = stored
            elif b64:
                stored = self._save_base64_image(b64, ctx)
                if stored:
                    public_url, object_key = stored

            return GenerationResult(
                status="completed" if public_url else "failed",
                progress=1.0,
                result_url=public_url,
                result_object_key=object_key,
                result_data=data,
                error=None if public_url else "Failed to store image",
            )
        except Exception as exc:  # noqa: BLE001
            return GenerationResult(status="failed", error=str(exc))
