import argparse
import base64
import json
import mimetypes
import os
import sys
import time
from typing import Dict, List, Optional, Tuple

import requests


def _load_api_key(cli_key: Optional[str]) -> str:
    key = cli_key or os.environ.get("KUAI_API_KEY") or os.environ.get("GEMINI_API_KEY") or os.environ.get("API_KEY")
    if not key:
        raise SystemExit("缺少 API Key：请设置环境变量 KUAI_API_KEY 或使用 --api-key")
    return key


def _load_image_base64(path: str) -> Tuple[str, str]:
    mime_type, _ = mimetypes.guess_type(path)
    if not mime_type:
        mime_type = "application/octet-stream"
    with open(path, "rb") as f:
        data = base64.b64encode(f.read()).decode("ascii")
    return data, mime_type


def _parse_modalities(raw: str) -> List[str]:
    items = [s.strip().upper() for s in raw.split(",") if s.strip()]
    return items or ["IMAGE"]


def _build_payload(
    prompt: Optional[str],
    ref_image: Optional[Tuple[str, str]],
    aspect_ratio: str,
    image_size: Optional[str],
    modalities: List[str],
) -> Dict:
    parts = []
    if prompt:
        parts.append({"text": prompt})
    if ref_image:
        data, mime_type = ref_image
        parts.append({"inline_data": {"mime_type": mime_type, "data": data}})
    if not parts:
        raise SystemExit("至少需要提供 prompt 或参考图")

    image_config = {"aspectRatio": aspect_ratio}
    if image_size:
        image_config["imageSize"] = image_size

    return {
        "contents": [{"role": "user", "parts": parts}],
        "generationConfig": {
            "responseModalities": modalities,
            "imageConfig": image_config,
        },
    }


def _build_url(base: str, model: str, use_query_key: bool, api_key: str) -> str:
    url = f"{base.rstrip('/')}/v1beta/models/{model}:generateContent"
    if use_query_key:
        sep = "&" if "?" in url else "?"
        url = f"{url}{sep}key={api_key}"
    return url


def _should_retry(status_code: int) -> bool:
    return status_code in (408, 429) or 500 <= status_code <= 599


def _extract_images_and_text(resp_json: Dict) -> Tuple[List[str], List[Dict]]:
    texts: List[str] = []
    images: List[Dict] = []
    for cand in resp_json.get("candidates", []) or []:
        content = cand.get("content") or {}
        for part in content.get("parts", []) or []:
            if "text" in part and isinstance(part["text"], str):
                texts.append(part["text"])
            inline = part.get("inlineData") or part.get("inline_data")
            if inline and isinstance(inline, dict):
                data = inline.get("data")
                mime = inline.get("mimeType") or inline.get("mime_type")
                if data:
                    images.append({"data": data, "mime": mime})
    return texts, images


def _mime_to_ext(mime: Optional[str]) -> str:
    if not mime:
        return "bin"
    if mime == "image/jpeg":
        return "jpg"
    if mime == "image/png":
        return "png"
    if mime == "image/webp":
        return "webp"
    return "bin"


def _save_images(images: List[Dict], out_dir: str, prefix: str) -> List[str]:
    os.makedirs(out_dir, exist_ok=True)
    saved = []
    for i, img in enumerate(images, start=1):
        ext = _mime_to_ext(img.get("mime"))
        path = os.path.join(out_dir, f"{prefix}_{i}.{ext}")
        with open(path, "wb") as f:
            f.write(base64.b64decode(img["data"]))
        saved.append(path)
    return saved


def main() -> None:
    parser = argparse.ArgumentParser(description="Gemini 原生格式图片生成示例")
    parser.add_argument("--api-base", default="https://api.kuai.host", help="API Base URL")
    parser.add_argument("--model", default="gemini-3-pro-image-preview", help="模型名称")
    parser.add_argument("--api-key", default=None, help="API Key（优先使用 KUAI_API_KEY 环境变量）")
    parser.add_argument("--use-query-key", action="store_true", help="使用 ?key= 方式鉴权")
    parser.add_argument("--prompt", default=None, help="提示词")
    parser.add_argument("--aspect-ratio", default="1:1", help="宽高比，如 16:9")
    parser.add_argument("--image-size", default="1K", help="清晰度/分辨率，如 1K/2K/4K")
    parser.add_argument("--ref-image", default=None, help="参考图文件路径（可选）")
    parser.add_argument("--modalities", default="TEXT,IMAGE", help="响应模态，逗号分隔，如 IMAGE")
    parser.add_argument("--timeout", type=int, default=120, help="请求超时时间（秒）")
    parser.add_argument("--retries", type=int, default=3, help="失败重试次数")
    parser.add_argument("--retry-interval", type=int, default=2, help="重试间隔（秒）")
    parser.add_argument("--retry-forever", action="store_true", help="一直重试直到成功")
    parser.add_argument("--out-dir", default=".", help="输出目录")
    parser.add_argument("--out-prefix", default="gemini_image", help="输出文件前缀")
    parser.add_argument("--save-json", action="store_true", help="保存原始 JSON 响应")
    args = parser.parse_args()

    api_key = _load_api_key(args.api_key)
    url = _build_url(args.api_base, args.model, args.use_query_key, api_key)

    ref_image = _load_image_base64(args.ref_image) if args.ref_image else None
    payload = _build_payload(args.prompt, ref_image, args.aspect_ratio, args.image_size, _parse_modalities(args.modalities))

    headers = {"Content-Type": "application/json"}
    if not args.use_query_key:
        headers["Authorization"] = f"Bearer {api_key}"

    attempt = 0
    while True:
        attempt += 1
        try:
            resp = requests.post(url, headers=headers, json=payload, timeout=args.timeout)
        except requests.RequestException as exc:
            print(f"请求异常: {exc}")
            if args.retry_forever or attempt <= args.retries:
                time.sleep(args.retry_interval)
                continue
            raise SystemExit(1)

        if not resp.ok:
            print(f"请求失败: HTTP {resp.status_code}")
            print(resp.text)
            if args.retry_forever:
                time.sleep(args.retry_interval)
                continue
            if attempt <= args.retries and _should_retry(resp.status_code):
                time.sleep(args.retry_interval)
                continue
            raise SystemExit(1)

        resp_json = resp.json()
        if args.save_json:
            json_path = os.path.join(args.out_dir, f"{args.out_prefix}.json")
            os.makedirs(args.out_dir, exist_ok=True)
            with open(json_path, "w", encoding="utf-8") as f:
                json.dump(resp_json, f, ensure_ascii=False, indent=2)
            print(f"已保存响应: {json_path}")

        texts, images = _extract_images_and_text(resp_json)
        if texts:
            print("文本内容:")
            for t in texts:
                print(t)

        if images:
            saved = _save_images(images, args.out_dir, args.out_prefix)
            for p in saved:
                print(f"已保存图片: {p}")
            break

        print("未检测到图片数据，将继续重试。")
        if args.retry_forever:
            time.sleep(args.retry_interval)
            continue
        raise SystemExit(1)


if __name__ == "__main__":
    main()
