from __future__ import annotations

import base64
import json
import mimetypes
import os
import queue
import threading
import tkinter as tk
import tkinter.filedialog as fd
import tkinter.messagebox as mb
import tkinter.ttk as ttk
from dataclasses import dataclass
from typing import Any, Optional

import requests


DEFAULT_HOST = "https://api.kuai.host"
DEFAULT_MODEL = "gemini-3-pro-image-preview"
DEFAULT_LOCAL_CONFIG_PATH = os.path.join(os.path.dirname(__file__), "config.local.json")

ASPECT_RATIO_OPTIONS = [
    "auto",
    "1:1",
    "2:3",
    "16:9",
    "9:16",
    "4:3",
    "4:5",
    "3:4",
    "3:2",
    "5:4",
    "21:9",
]

IMAGE_SIZE_OPTIONS = [
    "auto",
    "1K",
    "2K",
    "4K",
]

ASPECT_RATIO_ALLOWED = {
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
}


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


def _file_to_base64(path: str) -> tuple[str, str]:
    mime, _ = mimetypes.guess_type(path)
    if not mime:
        mime = "application/octet-stream"
    with open(path, "rb") as f:
        payload = base64.b64encode(f.read()).decode("ascii")
    return payload, mime


def _build_url(host: str, model: str, use_query_key: bool, api_key: str) -> str:
    base = host.strip().rstrip("/")
    model_name = model.strip()
    url = f"{base}/v1beta/models/{model_name}:generateContent"
    if use_query_key:
        sep = "&" if "?" in url else "?"
        url = f"{url}{sep}key={api_key}"
    return url


def _extract_images_and_text(resp_json: dict[str, Any]) -> tuple[list[str], list[dict[str, Optional[str]]]]:
    texts: list[str] = []
    images: list[dict[str, Optional[str]]] = []
    for cand in resp_json.get("candidates", []) or []:
        content = cand.get("content") or {}
        for part in content.get("parts", []) or []:
            if isinstance(part.get("text"), str):
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


@dataclass(frozen=True)
class RunRequest:
    host: str
    api_key: str
    model: str
    use_query_key: bool
    prompt: str
    aspect_ratio: Optional[str]
    image_size: Optional[str]
    ref_image_path: Optional[str]
    modalities: list[str]


class App(ttk.Frame):
    def __init__(self, master: tk.Tk) -> None:
        super().__init__(master)
        self.master.title("Gemini 图片生成测试 GUI")
        self.master.geometry("960x720")

        self._queue: queue.Queue[dict[str, Any]] = queue.Queue()
        self._worker: threading.Thread | None = None

        self._images: list[dict[str, Optional[str]]] = []
        self._local_config = load_local_config(DEFAULT_LOCAL_CONFIG_PATH)

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
        default_host = str(self._local_config.get("host") or DEFAULT_HOST)
        self.host_var = tk.StringVar(value=default_host)
        ttk.Entry(req_frame, textvariable=self.host_var).grid(row=0, column=1, columnspan=2, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="Model").grid(row=0, column=3, sticky="w", padx=8, pady=6)
        default_model = str(self._local_config.get("model") or DEFAULT_MODEL)
        self.model_var = tk.StringVar(value=default_model)
        ttk.Entry(req_frame, textvariable=self.model_var).grid(row=0, column=4, columnspan=2, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="API Key").grid(row=1, column=0, sticky="w", padx=8, pady=6)
        default_api_key = (
            str(self._local_config.get("api_key") or "").strip()
            or os.environ.get("KUAI_API_KEY", "")
            or os.environ.get("GEMINI_API_KEY", "")
        )
        self.api_key_var = tk.StringVar(value=default_api_key)
        self.api_key_entry = ttk.Entry(req_frame, textvariable=self.api_key_var, show="•")
        self.api_key_entry.grid(row=1, column=1, columnspan=3, sticky="ew", padx=8, pady=6)

        self.use_query_key_var = tk.BooleanVar(value=bool(self._local_config.get("use_query_key")))
        ttk.Checkbutton(req_frame, text="使用 ?key= 鉴权", variable=self.use_query_key_var).grid(
            row=1, column=4, sticky="w", padx=8, pady=6
        )
        ttk.Button(req_frame, text="保存配置", command=self._save_config).grid(
            row=1, column=5, sticky="e", padx=8, pady=6
        )

        ttk.Label(req_frame, text="Prompt").grid(row=2, column=0, sticky="nw", padx=8, pady=6)
        self.prompt_text = tk.Text(req_frame, height=6, wrap="word")
        self.prompt_text.grid(row=2, column=1, columnspan=5, sticky="nsew", padx=8, pady=6)

        ttk.Label(req_frame, text="Aspect Ratio").grid(row=3, column=0, sticky="w", padx=8, pady=6)
        self.aspect_ratio_var = tk.StringVar(value="16:9")
        ttk.Combobox(
            req_frame,
            textvariable=self.aspect_ratio_var,
            values=ASPECT_RATIO_OPTIONS,
            width=12,
            state="readonly",
        ).grid(row=3, column=1, sticky="w", padx=8, pady=6)

        ttk.Label(req_frame, text="Image Size").grid(row=3, column=2, sticky="w", padx=8, pady=6)
        self.image_size_var = tk.StringVar(value="1K")
        ttk.Combobox(
            req_frame,
            textvariable=self.image_size_var,
            values=IMAGE_SIZE_OPTIONS,
            width=10,
            state="readonly",
        ).grid(row=3, column=3, sticky="w", padx=8, pady=6)

        self.mod_text_var = tk.BooleanVar(value=True)
        self.mod_image_var = tk.BooleanVar(value=True)
        ttk.Label(req_frame, text="Modalities").grid(row=3, column=4, sticky="w", padx=8, pady=6)
        ttk.Checkbutton(req_frame, text="TEXT", variable=self.mod_text_var).grid(row=3, column=5, sticky="w", padx=4)
        ttk.Checkbutton(req_frame, text="IMAGE", variable=self.mod_image_var).grid(row=3, column=5, sticky="e", padx=4)

        ref_frame = ttk.Frame(req_frame)
        ref_frame.grid(row=4, column=0, columnspan=6, sticky="ew", padx=8, pady=(4, 8))
        ref_frame.columnconfigure(1, weight=1)
        ttk.Label(ref_frame, text="参考图").grid(row=0, column=0, sticky="w", padx=(0, 8))
        self.ref_image_var = tk.StringVar(value="")
        ttk.Entry(ref_frame, textvariable=self.ref_image_var).grid(row=0, column=1, sticky="ew")
        ttk.Button(ref_frame, text="选择文件", command=self._pick_ref_image).grid(row=0, column=2, padx=(8, 0))
        ttk.Button(ref_frame, text="清除", command=self._clear_ref_image).grid(row=0, column=3, padx=(8, 0))

        action_frame = ttk.Frame(self)
        action_frame.grid(row=1, column=0, sticky="ew", padx=10, pady=6)
        self.run_btn = ttk.Button(action_frame, text="发送请求", command=self._run)
        self.run_btn.grid(row=0, column=0, sticky="w")
        ttk.Button(action_frame, text="清空输出", command=self._clear_output).grid(row=0, column=1, sticky="w", padx=(8, 0))
        ttk.Button(action_frame, text="保存图片", command=self._save_images).grid(row=0, column=2, sticky="w", padx=(8, 0))

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

    def _pick_ref_image(self) -> None:
        path = fd.askopenfilename(
            title="选择参考图",
            filetypes=[("Image Files", "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.bmp"), ("All Files", "*.*")],
        )
        if path:
            self.ref_image_var.set(path)

    def _clear_ref_image(self) -> None:
        self.ref_image_var.set("")

    def _save_config(self) -> None:
        data = {
            "host": self.host_var.get().strip(),
            "model": self.model_var.get().strip(),
            "api_key": self.api_key_var.get().strip(),
            "use_query_key": bool(self.use_query_key_var.get()),
        }
        try:
            save_local_config(DEFAULT_LOCAL_CONFIG_PATH, data)
            mb.showinfo("保存成功", f"已写入 {DEFAULT_LOCAL_CONFIG_PATH}")
        except Exception as e:
            mb.showerror("保存失败", str(e))

    def _collect_run_request(self) -> Optional[RunRequest]:
        host = self.host_var.get().strip()
        if not host:
            mb.showerror("参数错误", "Host 不能为空")
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
        if not prompt and not self.ref_image_var.get().strip():
            mb.showerror("参数错误", "至少提供 Prompt 或参考图")
            return None
        aspect_ratio = self.aspect_ratio_var.get().strip()
        if aspect_ratio == "auto":
            aspect_ratio = None
        if aspect_ratio and aspect_ratio not in ASPECT_RATIO_ALLOWED:
            mb.showerror("参数错误", "Aspect Ratio 不合法，请从下拉列表选择")
            return None
        image_size = self.image_size_var.get().strip()
        if image_size == "auto":
            image_size = None
        modalities = []
        if self.mod_text_var.get():
            modalities.append("TEXT")
        if self.mod_image_var.get():
            modalities.append("IMAGE")
        if not modalities:
            mb.showerror("参数错误", "至少勾选一种 Modality")
            return None
        return RunRequest(
            host=host,
            api_key=api_key,
            model=model,
            use_query_key=bool(self.use_query_key_var.get()),
            prompt=prompt,
            aspect_ratio=aspect_ratio,
            image_size=image_size,
            ref_image_path=self.ref_image_var.get().strip() or None,
            modalities=modalities,
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
            try:
                parts: list[dict[str, Any]] = []
                if req.prompt:
                    parts.append({"text": req.prompt})
                if req.ref_image_path:
                    data, mime = _file_to_base64(req.ref_image_path)
                    parts.append({"inline_data": {"mime_type": mime, "data": data}})

                image_config: dict[str, Any] = {}
                if req.aspect_ratio:
                    image_config["aspectRatio"] = req.aspect_ratio
                if req.image_size:
                    image_config["imageSize"] = req.image_size

                generation_config: dict[str, Any] = {"responseModalities": req.modalities}
                if image_config:
                    generation_config["imageConfig"] = image_config

                payload: dict[str, Any] = {
                    "contents": [{"role": "user", "parts": parts}],
                    "generationConfig": generation_config,
                }
                url = _build_url(req.host, req.model, req.use_query_key, req.api_key)
                headers = {"Content-Type": "application/json"}
                if not req.use_query_key:
                    headers["Authorization"] = f"Bearer {req.api_key}"

                self._queue.put({"type": "log", "message": f"POST {url}"})
                resp = requests.post(url, headers=headers, json=payload, timeout=(10, 120))
                if not resp.ok:
                    self._queue.put({"type": "error", "message": f"HTTP {resp.status_code}: {resp.text}"})
                    return
                data = resp.json()
                self._queue.put({"type": "response", "data": data})
            except Exception as e:
                self._queue.put({"type": "error", "message": str(e)})
            finally:
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

        texts, images = _extract_images_and_text(data)
        for t in texts:
            self._append_log(f"[TEXT] {t}")

        if images:
            self._images = images
            self._append_log(f"检测到图片数量: {len(images)}（点击“保存图片”）")
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
            data = img.get("data")
            if not data:
                continue
            ext = _mime_to_ext(img.get("mime"))
            path = os.path.join(out_dir, f"gemini_image_{i}.{ext}")
            try:
                with open(path, "wb") as f:
                    f.write(base64.b64decode(data))
                saved += 1
            except Exception as e:
                self._append_log(f"保存失败: {path} {e}")
        mb.showinfo("保存完成", f"已保存 {saved} 张图片到 {out_dir}")


def main() -> None:
    mimetypes.init()
    root = tk.Tk()
    style = ttk.Style(root)
    try:
        style.theme_use("clam")
    except Exception:
        pass
    App(root)
    root.mainloop()


if __name__ == "__main__":
    main()
