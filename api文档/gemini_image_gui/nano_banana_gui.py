from __future__ import annotations

import base64
import json
import mimetypes
import os
import queue
import threading
from dataclasses import dataclass
from typing import Any, Optional
from urllib.parse import urlsplit

import requests
import tkinter as tk
import tkinter.filedialog as fd
import tkinter.messagebox as mb
import tkinter.ttk as ttk


DEFAULT_NANO_HOST = "https://ai.t8star.cn"
DEFAULT_NANO_ENDPOINT = "/v1/images/edits"
DEFAULT_NANO_MODEL = "nano-banana"
DEFAULT_LOCAL_CONFIG_PATH = os.path.join(os.path.dirname(__file__), "config.local.json")

ASPECT_RATIO_OPTIONS = [
    "auto(不传)",
    "1:1",
    "2:3",
    "3:2",
    "3:4",
    "4:3",
    "4:5",
    "5:4",
    "9:16",
    "16:9",
    "21:9",
]

IMAGE_SIZE_OPTIONS = [
    "auto(不传)",
    "1K",
    "2K",
    "4K",
]

RESPONSE_FORMAT_OPTIONS = [
    "url",
    "b64_json",
]


def load_local_config(path: str) -> dict[str, Any]:
    try:
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
        if isinstance(data, dict):
            return data
        return {}
    except FileNotFoundError:
        return {}
    except Exception:
        return {}


def save_local_config(path: str, data: dict[str, Any]) -> None:
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)


def _normalize_option(value: str) -> Optional[str]:
    if value.strip() == "auto(不传)":
        return None
    return value.strip() or None


def _build_url(host: str, endpoint: str) -> str:
    base = host.strip().rstrip("/")
    path = endpoint.strip()
    if not path:
        path = DEFAULT_NANO_ENDPOINT
    if not path.startswith("/"):
        path = "/" + path
    return f"{base}{path}"


def _guess_ext_from_url(url: str) -> str:
    try:
        path = urlsplit(url).path
        ext = os.path.splitext(path)[1].lstrip(".")
        return ext or "png"
    except Exception:
        return "png"


def _guess_ext_from_content_type(content_type: Optional[str]) -> str:
    if not content_type:
        return "png"
    mime = content_type.split(";")[0].strip().lower()
    if mime == "image/jpeg":
        return "jpg"
    if mime == "image/png":
        return "png"
    if mime == "image/webp":
        return "webp"
    return "png"


@dataclass(frozen=True)
class NanoRunRequest:
    host: str
    endpoint: str
    api_key: str
    model: str
    prompt: str
    response_format: Optional[str]
    aspect_ratio: Optional[str]
    image_size: Optional[str]
    ref_image_paths: list[str]


class NanoBananaApp(ttk.Frame):
    def __init__(self, master: tk.Tk) -> None:
        super().__init__(master)
        self.master.title("Nano-banana(Edits兼容) 测试 GUI")
        self.master.geometry("980x760")

        self._queue: queue.Queue[dict[str, Any]] = queue.Queue()
        self._worker: threading.Thread | None = None
        self._images: list[dict[str, str]] = []
        self._local_config = load_local_config(DEFAULT_LOCAL_CONFIG_PATH)
        self._ref_image_paths: list[str] = []

        self._build_ui()
        self.after(100, self._drain_queue)

    def _build_ui(self) -> None:
        self.pack(fill="both", expand=True)
        self.columnconfigure(0, weight=1)
        self.rowconfigure(2, weight=1)

        req_frame = ttk.LabelFrame(self, text="请求参数")
        req_frame.grid(row=0, column=0, sticky="nsew", padx=10, pady=(10, 6))
        for i in range(6):
            req_frame.columnconfigure(i, weight=1)

        ttk.Label(req_frame, text="Host").grid(row=0, column=0, sticky="w", padx=8, pady=6)
        default_host = str(self._local_config.get("nano_host") or DEFAULT_NANO_HOST)
        self.host_var = tk.StringVar(value=default_host)
        ttk.Entry(req_frame, textvariable=self.host_var).grid(row=0, column=1, columnspan=2, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="Endpoint").grid(row=0, column=3, sticky="w", padx=8, pady=6)
        default_endpoint = str(self._local_config.get("nano_endpoint") or DEFAULT_NANO_ENDPOINT)
        self.endpoint_var = tk.StringVar(value=default_endpoint)
        ttk.Entry(req_frame, textvariable=self.endpoint_var).grid(row=0, column=4, columnspan=2, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="Model").grid(row=1, column=0, sticky="w", padx=8, pady=6)
        default_model = str(self._local_config.get("nano_model") or DEFAULT_NANO_MODEL)
        self.model_var = tk.StringVar(value=default_model)
        ttk.Entry(req_frame, textvariable=self.model_var).grid(row=1, column=1, columnspan=2, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="API Key").grid(row=1, column=3, sticky="w", padx=8, pady=6)
        default_api_key = (
            str(self._local_config.get("nano_api_key") or self._local_config.get("api_key") or "").strip()
            or os.environ.get("NANO_BANANA_API_KEY", "")
            or os.environ.get("KUAI_API_KEY", "")
        )
        self.api_key_var = tk.StringVar(value=default_api_key)
        ttk.Entry(req_frame, textvariable=self.api_key_var, show="•").grid(
            row=1, column=4, columnspan=2, sticky="ew", padx=8, pady=6
        )

        ttk.Label(req_frame, text="Prompt").grid(row=2, column=0, sticky="nw", padx=8, pady=6)
        self.prompt_text = tk.Text(req_frame, height=6, wrap="word")
        self.prompt_text.grid(row=2, column=1, columnspan=5, sticky="nsew", padx=8, pady=6)

        ttk.Label(req_frame, text="Response Format").grid(row=3, column=0, sticky="w", padx=8, pady=6)
        default_format = str(self._local_config.get("nano_response_format") or "url")
        self.response_format_var = tk.StringVar(value=default_format)
        ttk.Combobox(req_frame, textvariable=self.response_format_var, values=RESPONSE_FORMAT_OPTIONS, width=10).grid(
            row=3, column=1, sticky="w", padx=8, pady=6
        )

        ttk.Label(req_frame, text="Aspect Ratio").grid(row=3, column=2, sticky="w", padx=8, pady=6)
        self.aspect_ratio_var = tk.StringVar(value="auto(不传)")
        ttk.Combobox(req_frame, textvariable=self.aspect_ratio_var, values=ASPECT_RATIO_OPTIONS, width=12).grid(
            row=3, column=3, sticky="w", padx=8, pady=6
        )

        ttk.Label(req_frame, text="Image Size").grid(row=3, column=4, sticky="w", padx=8, pady=6)
        self.image_size_var = tk.StringVar(value="auto(不传)")
        ttk.Combobox(req_frame, textvariable=self.image_size_var, values=IMAGE_SIZE_OPTIONS, width=10).grid(
            row=3, column=5, sticky="w", padx=8, pady=6
        )

        ref_frame = ttk.Frame(req_frame)
        ref_frame.grid(row=4, column=0, columnspan=6, sticky="ew", padx=8, pady=(4, 8))
        ref_frame.columnconfigure(1, weight=1)
        ttk.Label(ref_frame, text="参考图").grid(row=0, column=0, sticky="w", padx=(0, 8))
        self.ref_image_var = tk.StringVar(value="")
        ttk.Entry(ref_frame, textvariable=self.ref_image_var).grid(row=0, column=1, sticky="ew")
        ttk.Button(ref_frame, text="选择文件", command=self._pick_ref_images).grid(row=0, column=2, padx=(8, 0))
        ttk.Button(ref_frame, text="清除", command=self._clear_ref_images).grid(row=0, column=3, padx=(8, 0))

        action_frame = ttk.Frame(self)
        action_frame.grid(row=1, column=0, sticky="ew", padx=10, pady=6)
        self.run_btn = ttk.Button(action_frame, text="发送请求", command=self._run)
        self.run_btn.grid(row=0, column=0, sticky="w")
        ttk.Button(action_frame, text="清空输出", command=self._clear_output).grid(row=0, column=1, sticky="w", padx=(8, 0))
        ttk.Button(action_frame, text="保存图片", command=self._save_images).grid(row=0, column=2, sticky="w", padx=(8, 0))
        ttk.Button(action_frame, text="保存配置", command=self._save_config).grid(row=0, column=3, sticky="w", padx=(8, 0))

        out_frame = ttk.LabelFrame(self, text="输出")
        out_frame.grid(row=2, column=0, sticky="nsew", padx=10, pady=(6, 10))
        out_frame.rowconfigure(1, weight=1)
        out_frame.columnconfigure(0, weight=1)

        self.status_var = tk.StringVar(value="就绪")
        ttk.Label(out_frame, textvariable=self.status_var).grid(row=0, column=0, sticky="w", padx=8, pady=6)

        self.log_text = tk.Text(out_frame, wrap="word")
        self.log_text.grid(row=1, column=0, sticky="nsew", padx=8, pady=(0, 8))

    def _append_log(self, msg: str) -> None:
        self.log_text.insert("end", msg + "\n")
        self.log_text.see("end")

    def _clear_output(self) -> None:
        self.log_text.delete("1.0", "end")
        self.status_var.set("就绪")
        self._images = []

    def _pick_ref_images(self) -> None:
        paths = fd.askopenfilenames(
            title="选择参考图",
            filetypes=[("Image Files", "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.bmp"), ("All Files", "*.*")],
        )
        if paths:
            self._ref_image_paths = list(paths)
            self.ref_image_var.set("; ".join(paths))

    def _clear_ref_images(self) -> None:
        self._ref_image_paths = []
        self.ref_image_var.set("")

    def _save_config(self) -> None:
        data = dict(self._local_config)
        data.update(
            {
                "nano_host": self.host_var.get().strip(),
                "nano_endpoint": self.endpoint_var.get().strip(),
                "nano_model": self.model_var.get().strip(),
                "nano_api_key": self.api_key_var.get().strip(),
                "nano_response_format": self.response_format_var.get().strip(),
            }
        )
        try:
            save_local_config(DEFAULT_LOCAL_CONFIG_PATH, data)
            mb.showinfo("保存成功", f"已写入 {DEFAULT_LOCAL_CONFIG_PATH}")
        except Exception as e:
            mb.showerror("保存失败", str(e))

    def _collect_run_request(self) -> Optional[NanoRunRequest]:
        host = self.host_var.get().strip()
        if not host:
            mb.showerror("参数错误", "Host 不能为空")
            return None
        endpoint = self.endpoint_var.get().strip()
        if not endpoint:
            mb.showerror("参数错误", "Endpoint 不能为空")
            return None
        api_key = self.api_key_var.get().strip()
        if not api_key:
            mb.showerror("参数错误", "API Key 不能为空")
            return None
        model = self.model_var.get().strip()
        if not model:
            mb.showerror("参数错误", "Model 不能为空")
            return None
        prompt = self.prompt_text.get("1.0", "end").strip()
        if not prompt:
            mb.showerror("参数错误", "Prompt 不能为空")
            return None
        if "/edits" in endpoint and not self._ref_image_paths:
            mb.showerror("参数错误", "该接口必须上传 image；无参考图请改用 /v1/images/generations")
            return None
        response_format = _normalize_option(self.response_format_var.get())
        aspect_ratio = _normalize_option(self.aspect_ratio_var.get())
        image_size = _normalize_option(self.image_size_var.get())
        return NanoRunRequest(
            host=host,
            endpoint=endpoint,
            api_key=api_key,
            model=model,
            prompt=prompt,
            response_format=response_format,
            aspect_ratio=aspect_ratio,
            image_size=image_size,
            ref_image_paths=list(self._ref_image_paths),
        )

    def _set_running(self, running: bool) -> None:
        self.run_btn.configure(state=("disabled" if running else "normal"))
        self.status_var.set("请求中..." if running else "就绪")

    def _run(self) -> None:
        if self._worker and self._worker.is_alive():
            return

        req = self._collect_run_request()
        if not req:
            return

        self._set_running(True)
        self._append_log("准备请求...")

        def worker() -> None:
            file_handles: list[Any] = []
            try:
                url = _build_url(req.host, req.endpoint)
                headers = {"Authorization": f"Bearer {req.api_key}"}
                multipart: list[tuple[str, tuple[Any, ...]]] = [
                    ("model", (None, req.model)),
                ]
                if req.prompt:
                    multipart.append(("prompt", (None, req.prompt)))
                if req.response_format:
                    multipart.append(("response_format", (None, req.response_format)))
                if req.aspect_ratio:
                    multipart.append(("aspect_ratio", (None, req.aspect_ratio)))
                if req.image_size:
                    multipart.append(("image_size", (None, req.image_size)))

                for path in req.ref_image_paths:
                    mime, _ = mimetypes.guess_type(path)
                    if not mime:
                        mime = "application/octet-stream"
                    f = open(path, "rb")
                    file_handles.append(f)
                    multipart.append(("image", (os.path.basename(path), f, mime)))

                self._queue.put({"type": "log", "message": f"POST {url}"})
                resp = requests.post(url, headers=headers, files=multipart, timeout=(10, 120))
                if not resp.ok:
                    self._queue.put({"type": "error", "message": f"HTTP {resp.status_code}: {resp.text}"})
                    return
                try:
                    payload = resp.json()
                except Exception:
                    self._queue.put({"type": "error", "message": "响应不是有效 JSON"})
                    return
                self._queue.put({"type": "response", "data": payload})
            except Exception as e:
                self._queue.put({"type": "error", "message": str(e)})
            finally:
                for f in file_handles:
                    try:
                        f.close()
                    except Exception:
                        pass
                self._queue.put({"type": "done"})

        self._worker = threading.Thread(target=worker, daemon=True)
        self._worker.start()

    def _drain_queue(self) -> None:
        try:
            while True:
                msg = self._queue.get_nowait()
                self._handle_msg(msg)
        except queue.Empty:
            pass
        self.after(100, self._drain_queue)

    def _handle_msg(self, msg: dict[str, Any]) -> None:
        t = msg.get("type")
        if t == "log":
            self._append_log(str(msg.get("message", "")))
            return
        if t == "error":
            self._append_log("ERROR: " + str(msg.get("message", "")))
            return
        if t == "response":
            data = msg.get("data")
            if isinstance(data, dict):
                self._render_response(data)
            return
        if t == "done":
            self._set_running(False)
            return

    def _render_response(self, data: dict[str, Any]) -> None:
        try:
            pretty = json.dumps(data, ensure_ascii=False, indent=2)
        except Exception:
            pretty = str(data)
        self._append_log(pretty)

        images: list[dict[str, str]] = []
        for item in data.get("data", []) or []:
            if isinstance(item, dict):
                if isinstance(item.get("b64_json"), str):
                    images.append({"type": "b64", "data": item["b64_json"]})
                if isinstance(item.get("url"), str):
                    images.append({"type": "url", "data": item["url"]})

        if images:
            self._images = images
            self._append_log(f"检测到图片数量: {len(images)}（点击“保存图片”）")
            for img in images:
                if img["type"] == "url":
                    self._append_log(f"[URL] {img['data']}")
        else:
            self._append_log("未检测到图片数据")

    def _save_images(self) -> None:
        if not self._images:
            mb.showinfo("没有图片", "当前没有可保存的图片数据")
            return
        out_dir = fd.askdirectory(title="选择保存目录")
        if not out_dir:
            return
        saved = 0
        for i, img in enumerate(self._images, start=1):
            if img["type"] == "b64":
                data = img.get("data", "")
                if not data:
                    continue
                path = os.path.join(out_dir, f"nano_banana_image_{i}.png")
                try:
                    with open(path, "wb") as f:
                        f.write(base64.b64decode(data))
                    saved += 1
                except Exception as e:
                    self._append_log(f"保存失败: {path} {e}")
            elif img["type"] == "url":
                url = img.get("data", "")
                if not url:
                    continue
                try:
                    resp = requests.get(url, timeout=(10, 120))
                    if not resp.ok:
                        self._append_log(f"下载失败: {url} HTTP {resp.status_code}")
                        continue
                    ext = _guess_ext_from_content_type(resp.headers.get("Content-Type")) or _guess_ext_from_url(url)
                    path = os.path.join(out_dir, f"nano_banana_image_{i}.{ext}")
                    with open(path, "wb") as f:
                        f.write(resp.content)
                    saved += 1
                except Exception as e:
                    self._append_log(f"下载失败: {url} {e}")
        mb.showinfo("保存完成", f"已保存 {saved} 张图片到 {out_dir}")


def main() -> None:
    mimetypes.init()
    root = tk.Tk()
    style = ttk.Style(root)
    try:
        style.theme_use("clam")
    except Exception:
        pass
    NanoBananaApp(root)
    root.mainloop()


if __name__ == "__main__":
    main()
