"""
Sora2 视频生成适配器 - 完整版本
基于验证过的API调用逻辑实现，包含视频下载功能
"""

import json
import time
import logging
import requests
import uuid
from typing import Dict, Any, Optional, Tuple
from urllib.parse import urlparse

from app.adapters.base import BaseAdapter, GenerationResult, GenerationTaskContext
from app.adapters.media_utils import maybe_convert_image_input
from app.config import get_settings
from app.storage import get_storage

logger = logging.getLogger(__name__)


class Sora2Adapter(BaseAdapter):
    """Sora2 视频生成适配器"""

    # 内部参数名 → API参数名 的映射
    PARAM_MAPPING = {
        "aspect_ratio": "aspectRatio",
        "duration_seconds": "duration",
        "web_hook": "webHook",
        "shut_progress": "shutProgress",
    }

    def __init__(self, settings: Dict[str, Any] = None):
        # 获取配置
        app_settings = get_settings()

        self.api_host = app_settings.sora2_api_host or "grsai.dakka.com.cn"
        self.api_path = "/v1/video/sora-video"
        self.result_path = "/v1/draw/result"
        self.api_key = app_settings.sora2_api_key

        # 如果传入settings参数，从中提取配置
        if settings and isinstance(settings, dict):
            self.api_key = settings.get('api_key', self.api_key)
            # 允许通过 base_url / api_host 覆盖，自动剥离协议
            override_host = settings.get('api_host') or settings.get('base_url')
            if override_host:
                self.api_host = self._normalize_host(override_host)

        # 必须通过环境变量/配置提供，不允许在代码里硬编码默认密钥

        # 视频存储配置 - 使用统一存储模块
        self.storage_root = app_settings.storage_root
        self.storage = get_storage()

        # 状态映射（对应数据库TaskStatus）
        self.status_map = {
            'pending': 'pending',
            'running': 'processing',
            'succeeded': 'completed',
            'failed': 'failed',
            'queue': 'processing',  # 队列状态也视为处理中
            'queued': 'processing'   # queued状态也视为处理中
        }

    def _validate_config(self) -> bool:
        """验证配置"""
        if not self.api_key:
            logger.error("Sora2 adapter: API key is required")
            return False
        return True

    def _merge_params(self, ctx: GenerationTaskContext) -> Dict[str, Any]:
        """合并参数并转换为API格式"""
        # 1. 按优先级合并参数
        merged = {}
        merged.update(ctx.default_params or {})
        merged.update(ctx.user_inputs or {})
        merged.update(ctx.admin_fixed or {})

        # 2. 提取并验证参数
        prompt = merged.get("prompt")
        if not prompt:
            raise ValueError("Prompt is required for Sora2 video generation")

        aspect_ratio = merged.get("aspect_ratio", "9:16")
        duration = merged.get("duration_seconds") or merged.get("duration", 10)
        size = merged.get("size", "small")
        image_url = merged.get("image") or merged.get("image_url")
        web_hook = merged.get("web_hook", "-1")
        shut_progress = merged.get("shut_progress", False)

        # 3. 参数校验
        try:
            duration = int(duration)
        except Exception:
            duration = 10
        if duration not in [10, 15]:
            duration = 10
        if aspect_ratio not in ["16:9", "9:16", "1:1"]:
            aspect_ratio = "9:16"
        if size not in ["small", "large"]:
            size = "small"

        # 4. 构建API payload（使用驼峰命名）
        payload = {
            "prompt": prompt,
            "model": "sora-2",
            "aspectRatio": aspect_ratio,
            "duration": duration,
            "size": size,
            "webHook": web_hook,
            "shutProgress": bool(shut_progress),
        }
        if image_url:
            payload["url"] = maybe_convert_image_input(image_url, storage_root=self.storage_root)

        return payload

    def generate(self, ctx: GenerationTaskContext, progress_callback=None) -> GenerationResult:
        """生成方法（基类要求的抽象方法实现）

        progress_callback: 可选回调，用于在轮询过程中持续上报进度到调用方（例如写库给前端展示）。
        """
        if not self._validate_config():
            return GenerationResult(
                status="failed",
                error="Sora2 adapter configuration is invalid"
            )

        request_data = self._merge_params(ctx)

        try:
            logger.info(f"Sora2 generation request: {json.dumps(request_data, ensure_ascii=False)}")

            # 提交生成任务
            task_id, submit_error = self._submit_generation_task(request_data)

            if not task_id:
                return GenerationResult(
                    status="failed",
                    error=submit_error or "Failed to submit Sora2 generation task"
                )

            # 轮询任务状态
            return self._poll_task_status(task_id, ctx, progress_callback)

        except Exception as e:
            logger.error(f"Sora2 generation error: {str(e)}")
            return GenerationResult(
                status="failed",
                error=f"Sora2 generation failed: {str(e)}"
            )

    def _submit_generation_task(self, request_data: Dict[str, Any]) -> tuple[Optional[str], Optional[str]]:
        """提交生成任务，返回 (task_id, error)。对 429/网络错误做退避重试。"""
        url = f"https://{self._normalize_host(self.api_host)}{self.api_path}"
        headers = {
            'Content-Type': 'application/json',
            'Authorization': f'Bearer {self.api_key}'
        }

        max_attempts = 5
        for attempt in range(1, max_attempts + 1):
            try:
                response = requests.post(
                    url,
                    json=request_data,
                    headers=headers,
                    timeout=30
                )
            except Exception as e:
                logger.error(f"Sora2 task submission error (attempt {attempt}/{max_attempts}): {str(e)}")
                if attempt == max_attempts:
                    return None, f"request error: {e}"
                time.sleep(attempt * 2)
                continue

            if response.status_code == 429:
                wait = attempt * 2
                logger.warning(f"Sora2 API 429 rate limited, retrying in {wait}s (attempt {attempt}/{max_attempts})")
                if attempt == max_attempts:
                    return None, f"api rate limited 429: {response.text}"
                time.sleep(wait)
                continue

            if response.status_code != 200:
                logger.error(f"Sora2 API request failed: {response.status_code} - {response.text}")
                return None, f"api request failed: {response.status_code} {response.text}"

            try:
                result = response.json()
            except Exception:
                logger.error(f"Sora2 API response not json: {response.text}")
                return None, f"api response not json: {response.text}"

            if result.get("code") == 0:
                task_id = result.get('data', {}).get('id')
                logger.info(f"Sora2 task submitted successfully, task_id: {task_id}")
                return task_id, None

            error_msg = result.get('msg', 'Unknown error')
            logger.error(f"Sora2 API returned error: {error_msg}")
            return None, f"api error: {error_msg}"

        return None, "api submission failed after retries"

    @staticmethod
    def _normalize_host(host_or_url: str) -> str:
        """剥离协议/路径，只返回主机名。"""
        if not host_or_url:
            return host_or_url
        parsed = urlparse(host_or_url)
        if parsed.scheme and parsed.hostname:
            return parsed.hostname
        # 如果传入的是 //host 或 host/path
        stripped = host_or_url.replace("https://", "").replace("http://", "").lstrip("/")
        return stripped.split("/")[0]

    def _poll_task_status(self, task_id: str, ctx: GenerationTaskContext, progress_callback=None) -> GenerationResult:
        """轮询任务状态，支持进度回调。"""
        max_wait_time = 15 * 60  # 15分钟最大等待时间（sora2视频生成需要更长时间）
        check_interval = 3  # 3秒检查一次，更频繁的进度更新
        start_time = time.time()

        logger.info(f"Starting to poll Sora2 task status, task_id: {task_id}")

        while (time.time() - start_time) < max_wait_time:
            try:
                status_result = self._query_task_status(task_id)

                if not status_result:
                    time.sleep(check_interval)
                    continue

                if status_result.get("code") == 0:
                    data = status_result.get('data', {})
                    status = data.get('status', 'unknown')
                    progress = data.get('progress', 0)

                    logger.info(f"Sora2 task status: {status}, progress: {progress}%")

                    # 映射状态
                    mapped_status = self.status_map.get(status, 'processing')

                    # 首先统一上报进度（不管什么状态都上报）
                    progress_percent = float(progress) / 100 if progress is not None else 0.0
                    if progress_callback:
                        try:
                            progress_callback(progress_percent, data=data)
                            logger.info(f"Sora2 progress callback: {progress_percent:.2f} (status: {status})")
                        except Exception as exc:  # noqa: BLE001
                            logger.warning(f"Progress callback failed: {exc}")

                    if status == 'succeeded':
                        # 处理成功结果
                        results = data.get('results', [])
                        if results:
                            video_url = results[0].get('url')
                            logger.info(f"Sora2 generation completed successfully, remote video_url: {video_url}")

                            # 尝试下载视频并存储到本地；如果失败则按失败处理（不返回供应商临时URL）
                            stored = self._download_and_store_video(video_url, ctx)
                            if stored:
                                local_video_url, object_key = stored
                                logger.info(f"Video downloaded and stored locally: {local_video_url}")
                                return GenerationResult(
                                    status=mapped_status,
                                    progress=1.0,
                                    result_url=local_video_url,
                                    result_object_key=object_key,
                                    result_data=data
                                )
                            else:
                                logger.error(
                                    "Sora2 video download/store failed; refusing to return provider URL. video_url=%s",
                                    video_url,
                                )
                                return GenerationResult(
                                    status="failed",
                                    progress=1.0,
                                    result_data=data,
                                    error="视频生成成功但下载/落盘失败，请稍后重试",
                                )
                        else:
                            return GenerationResult(
                                status="failed",
                                error="Generation completed but no video result found"
                            )

                    elif status == 'failed':
                        error_msg = data.get('error', data.get('failure_reason', 'Unknown error'))
                        logger.error(f"Sora2 task failed: {error_msg}")
                        return GenerationResult(
                            status=mapped_status,
                            error=error_msg
                        )

                    elif status in ['running', 'pending', 'queued', 'queue']:
                        # 继续轮询，进度已经在前面统一上报了
                        time.sleep(check_interval)
                        continue

                    else:
                        # 未知状态，进度已经在前面统一上报了
                        logger.info(f"Sora2 task unknown status: {status}, progress: {progress}%")
                        time.sleep(check_interval)
                        continue

                # 等待下一次检查
                time.sleep(check_interval)

            except Exception as e:
                logger.error(f"Sora2 status polling error: {str(e)}")
                # 对于网络错误等临时问题，继续轮询而不是立即失败
                if "connection" in str(e).lower() or "timeout" in str(e).lower():
                    logger.info(f"Network error detected, continuing polling: {str(e)}")
                    time.sleep(check_interval)
                    continue
                else:
                    # 对于其他类型的错误，继续轮询但记录更详细信息
                    logger.warning(f"Non-network polling error, continuing: {str(e)}")
                    time.sleep(check_interval)
                    continue

        # 轮询超时
        logger.warning(f"Sora2 task polling timeout, task_id: {task_id}")
        return GenerationResult(
            status="failed",
            error="Task polling timeout"
        )

    def _query_task_status(self, task_id: str, max_retries: int = 3) -> Optional[Dict[str, Any]]:
        """查询任务状态，支持重试机制"""
        for attempt in range(max_retries):
            try:
                headers = {
                    'Content-Type': 'application/json',
                    'Authorization': f'Bearer {self.api_key}'
                }

                # 统一规范 host，避免出现 https://https... 导致解析到 host='https'
                url = f"https://{self._normalize_host(self.api_host)}{self.result_path}"
                request_data = {"id": task_id}

                response = requests.post(
                    url,
                    json=request_data,
                    headers=headers,
                    timeout=30
                )

                if response.status_code == 200:
                    return response.json()
                elif response.status_code in [429, 500, 502, 503, 504]:  # 可重试的错误码
                    logger.warning(f"Sora2 status query retryable error: {response.status_code}, attempt {attempt + 1}/{max_retries}")
                    if attempt < max_retries - 1:
                        time.sleep(2 ** attempt)  # 指数退避
                        continue
                else:
                    logger.error(f"Sora2 status query failed: {response.status_code} - {response.text}")
                    return None

            except Exception as e:
                error_str = str(e).lower()
                if any(keyword in error_str for keyword in ["connection", "timeout", "network"]):
                    logger.warning(f"Sora2 status query network error, attempt {attempt + 1}/{max_retries}: {str(e)}")
                    if attempt < max_retries - 1:
                        time.sleep(2 ** attempt)  # 指数退避
                        continue
                else:
                    logger.error(f"Sora2 status query non-retryable error: {str(e)}")
                    return None

        return None

    def _download_and_store_video(self, video_url: str, ctx: GenerationTaskContext) -> Optional[Tuple[str, str]]:
        """下载视频并存储（支持本地/OSS）"""
        try:
            logger.info(f"Starting video download from: {video_url}")
            logger.info(f"User ID: {ctx.user_id}")

            # 发送下载请求，添加必要的请求头
            headers = {
                'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
                'Referer': 'https://grsai.dakka.com.cn/',
                'Accept': 'video/mp4,video/webm,video/*;q=0.9,application/json;q=0.8,*/*;q=0.5',
                'Accept-Language': 'en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7',
                'Accept-Encoding': 'gzip, deflate, br',
                'Cache-Control': 'no-cache',
                'Pragma': 'no-cache'
            }

            logger.info(f"Sending GET request to: {video_url}")
            response = requests.get(video_url, stream=True, timeout=60, headers=headers)
            logger.info(f"Download response status: {response.status_code}")
            response.raise_for_status()

            # 检查内容类型
            content_type = response.headers.get('content-type', 'video/mp4')
            content_length = response.headers.get('content-length', 'unknown')
            logger.info(f"Content type: {content_type}, size: {content_length}")

            # 生成唯一的文件名
            file_id = str(uuid.uuid4())
            prefix = self._resolve_output_filename_prefix(ctx, "sora2")
            filename = f"{prefix}_{ctx.user_id}_{file_id}.mp4"
            logger.info(f"Generated filename: {filename}")

            # 使用统一存储模块保存文件（支持本地/OSS）
            file_path = f"{ctx.user_id}/{filename}"
            public_url, object_key = self.storage.save_file_with_key(
                response.content,
                file_path,
                "video/mp4",
                base_dir="user/videos"
            )

            logger.info(f"Video stored successfully: {public_url}")
            return public_url, object_key

        except requests.exceptions.HTTPError as e:
            resp = getattr(e, 'response', None)
            if resp and resp.status_code == 404:
                logger.error(f"Video file not found (404): {video_url}")
                logger.error("This could mean:")
                logger.error("1. The URL has expired (Grsai URLs are temporary)")
                logger.error("2. The file was removed from Grsai servers")
                logger.error("3. Authentication headers are missing")
            else:
                logger.error(f"HTTP error downloading video: {e}")
                if resp:
                    logger.error(f"Status: {resp.status_code}")
                    logger.error(f"Headers: {dict(resp.headers)}")
            logger.exception("Full exception details:")
            return None
        except requests.exceptions.RequestException as e:
            logger.error(f"Network error downloading video: {e}")
            logger.exception("Full exception details:")
            return None
        except Exception as e:
            logger.error(f"Unexpected error downloading video: {str(e)}")
            logger.exception("Full exception details:")
            return None

    def get_supported_formats(self) -> Dict[str, Any]:
        """获取支持的格式信息"""
        return {
            "output_types": ["video"],
            "supported_ratios": ["9:16", "16:9"],
            "supported_durations": [10, 15],
            "supported_sizes": ["small", "large"],
            "supports_image_input": True,
            "default_ratio": "9:16",
            "default_duration": 10,
            "default_size": "small"
        }
