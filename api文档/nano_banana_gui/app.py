from __future__ import annotations

import base64
import io
import json
import mimetypes
import os
import queue
import threading
import time
import tkinter as tk
import tkinter.filedialog as fd
import tkinter.messagebox as mb
import tkinter.ttk as ttk
from dataclasses import dataclass
from typing import Any, Iterable

import requests
from PIL import Image, ImageTk


DEFAULT_HOSTS = [
    "https://grsai.dakka.com.cn",
    "https://grsaiapi.com",
]

DEFAULT_DRAW_PATH = "/v1/draw/nano-banana"
DEFAULT_RESULT_PATH = "/v1/draw/result"
DEFAULT_LOCAL_CONFIG_PATH = os.path.join(os.path.dirname(__file__), "config.local.json")

LOCAL_IMAGE_ENCODING_OPTIONS: list[tuple[str, str]] = [
    ("data_url", "data URL（推荐）"),
    ("base64", "纯 base64"),
]

MODEL_OPTIONS = [
    "nano-banana-fast",
    "nano-banana",
    "nano-banana-pro",
    "nano-banana-pro-vt",
    "nano-banana-pro-cl",
    "nano-banana-pro-vip",
    "nano-banana-pro-4k-vip",
]

ASPECT_RATIO_OPTIONS = [
    "auto",
    "1:1",
    "16:9",
    "9:16",
    "4:3",
    "3:4",
    "3:2",
    "2:3",
    "5:4",
    "4:5",
    "21:9",
]

IMAGE_SIZE_OPTIONS = [
    "1K",
    "2K",
    "4K",
]


def _join_url(base: str, path: str) -> str:
    base = base.strip().rstrip("/")
    path = path.strip()
    if not path.startswith("/"):
        path = "/" + path
    return base + path


def file_to_data_url(path: str) -> str:
    mime, _ = mimetypes.guess_type(path)
    if not mime:
        mime = "application/octet-stream"
    payload = file_to_base64(path)
    return f"data:{mime};base64,{payload}"


def file_to_base64(path: str) -> str:
    with open(path, "rb") as f:
        return base64.b64encode(f.read()).decode("ascii")


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


@dataclass(frozen=True)
class ImageInputItem:
    kind: str  # "file" | "url"
    value: str


@dataclass(frozen=True)
class RunRequest:
    host: str
    api_key: str
    draw_path: str
    result_path: str
    mode: str  # "stream" | "poll"
    payload_fields: dict[str, Any]
    image_inputs: tuple[ImageInputItem, ...]
    local_image_encoding: str  # "data_url" | "base64"


class NanoBananaClient:
    def __init__(self, host: str, api_key: str, *, draw_path: str, result_path: str) -> None:
        self._host = host
        self._api_key = api_key
        self._draw_path = draw_path
        self._result_path = result_path

    def _headers(self) -> dict[str, str]:
        return {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self._api_key}",
        }

    def draw_stream(self, payload: dict[str, Any], *, cancel: threading.Event) -> Iterable[dict[str, Any]]:
        url = _join_url(self._host, self._draw_path)
        with requests.post(
            url,
            headers=self._headers(),
            json=payload,
            stream=True,
            timeout=(10, 300),
        ) as resp:
            resp.raise_for_status()
            for raw_line in resp.iter_lines(decode_unicode=True):
                if cancel.is_set():
                    return
                if not raw_line:
                    continue
                line = raw_line.strip()
                if not line:
                    continue
                if line.startswith("data:"):
                    line = line.removeprefix("data:").strip()
                if line == "[DONE]":
                    continue
                try:
                    yield json.loads(line)
                except json.JSONDecodeError:
                    continue

    def create_task(self, payload: dict[str, Any]) -> str:
        url = _join_url(self._host, self._draw_path)
        resp = requests.post(url, headers=self._headers(), json=payload, timeout=(10, 60))
        resp.raise_for_status()
        data = resp.json()
        if data.get("code") != 0:
            raise RuntimeError(f"create_task failed: {data}")
        task_id = (data.get("data") or {}).get("id")
        if not task_id:
            raise RuntimeError(f"create_task missing id: {data}")
        return str(task_id)

    def get_result(self, task_id: str) -> dict[str, Any]:
        url = _join_url(self._host, self._result_path)
        resp = requests.post(url, headers=self._headers(), json={"id": task_id}, timeout=(10, 60))
        resp.raise_for_status()
        data = resp.json()
        if data.get("code") != 0:
            raise RuntimeError(f"get_result failed: {data}")
        payload = data.get("data")
        if not isinstance(payload, dict):
            raise RuntimeError(f"get_result missing data: {data}")
        return payload


class App(ttk.Frame):
    def __init__(self, master: tk.Tk) -> None:
        super().__init__(master)
        self.master.title("Nano Banana API 本地测试 GUI")
        self.master.geometry("1100x760")

        self._queue: queue.Queue[dict[str, Any]] = queue.Queue()
        self._worker: threading.Thread | None = None
        self._cancel = threading.Event()

        self._image_inputs: list[ImageInputItem] = []
        self._preview_photo: ImageTk.PhotoImage | None = None
        self._download_cache: dict[str, bytes] = {}
        self._local_config = load_local_config(DEFAULT_LOCAL_CONFIG_PATH)

        self._build_ui()
        self._update_payload_preview()
        self.after(100, self._drain_queue)

    def _build_ui(self) -> None:
        self.pack(fill="both", expand=True)

        self.columnconfigure(0, weight=1)
        self.columnconfigure(1, weight=1)
        self.rowconfigure(2, weight=1)

        req_frame = ttk.LabelFrame(self, text="请求参数")
        req_frame.grid(row=0, column=0, columnspan=2, sticky="nsew", padx=10, pady=(10, 6))
        for i in range(8):
            req_frame.columnconfigure(i, weight=1)

        ttk.Label(req_frame, text="Host").grid(row=0, column=0, sticky="w", padx=8, pady=6)
        default_host = str(self._local_config.get("host") or DEFAULT_HOSTS[0])
        self.host_var = tk.StringVar(value=default_host)
        self.host_combo = ttk.Combobox(req_frame, textvariable=self.host_var, values=DEFAULT_HOSTS, width=40)
        self.host_combo.grid(row=0, column=1, columnspan=3, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="API Key").grid(row=0, column=4, sticky="w", padx=8, pady=6)
        default_api_key = (
            str(self._local_config.get("api_key") or "").strip() or os.environ.get("GRSAI_API_KEY", "")
        )
        self.api_key_var = tk.StringVar(value=default_api_key)
        self.api_key_entry = ttk.Entry(req_frame, textvariable=self.api_key_var, show="•")
        self.api_key_entry.grid(row=0, column=5, columnspan=3, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="Draw Path").grid(row=1, column=0, sticky="w", padx=8, pady=6)
        self.draw_path_var = tk.StringVar(value=DEFAULT_DRAW_PATH)
        ttk.Entry(req_frame, textvariable=self.draw_path_var).grid(
            row=1, column=1, columnspan=3, sticky="ew", padx=8, pady=6
        )

        ttk.Label(req_frame, text="Result Path").grid(row=1, column=4, sticky="w", padx=8, pady=6)
        self.result_path_var = tk.StringVar(value=DEFAULT_RESULT_PATH)
        ttk.Entry(req_frame, textvariable=self.result_path_var).grid(
            row=1, column=5, columnspan=3, sticky="ew", padx=8, pady=6
        )

        ttk.Label(req_frame, text="Model").grid(row=2, column=0, sticky="w", padx=8, pady=6)
        self.model_var = tk.StringVar(value=MODEL_OPTIONS[0])
        model_combo = ttk.Combobox(req_frame, textvariable=self.model_var, values=MODEL_OPTIONS, width=30)
        model_combo.grid(row=2, column=1, columnspan=3, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="Aspect Ratio").grid(row=2, column=4, sticky="w", padx=8, pady=6)
        self.aspect_ratio_var = tk.StringVar(value="auto")
        aspect_combo = ttk.Combobox(
            req_frame, textvariable=self.aspect_ratio_var, values=ASPECT_RATIO_OPTIONS, width=12
        )
        aspect_combo.grid(row=2, column=5, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="Image Size").grid(row=2, column=6, sticky="w", padx=8, pady=6)
        self.image_size_var = tk.StringVar(value="1K")
        size_combo = ttk.Combobox(req_frame, textvariable=self.image_size_var, values=IMAGE_SIZE_OPTIONS, width=10)
        size_combo.grid(row=2, column=7, sticky="ew", padx=8, pady=6)

        ttk.Label(req_frame, text="调用模式").grid(row=3, column=0, sticky="w", padx=8, pady=6)
        self.mode_var = tk.StringVar(value="stream")
        ttk.Radiobutton(req_frame, text="Stream 流式", value="stream", variable=self.mode_var).grid(
            row=3, column=1, sticky="w", padx=8, pady=6
        )
        ttk.Radiobutton(req_frame, text="轮询", value="poll", variable=self.mode_var).grid(
            row=3, column=2, sticky="w", padx=8, pady=6
        )

        self.shut_progress_var = tk.BooleanVar(value=False)
        ttk.Checkbutton(req_frame, text="shutProgress", variable=self.shut_progress_var).grid(
            row=3, column=4, sticky="w", padx=8, pady=6
        )

        ttk.Label(req_frame, text="Prompt").grid(row=4, column=0, sticky="nw", padx=8, pady=6)
        self.prompt_text = tk.Text(req_frame, height=6, wrap="word")
        self.prompt_text.grid(row=4, column=1, columnspan=7, sticky="nsew", padx=8, pady=6)

        imgs_frame = ttk.LabelFrame(req_frame, text="多图输入（urls）")
        imgs_frame.grid(row=5, column=0, columnspan=8, sticky="nsew", padx=8, pady=8)
        imgs_frame.columnconfigure(0, weight=1)
        imgs_frame.rowconfigure(1, weight=1)

        btn_row = ttk.Frame(imgs_frame)
        btn_row.grid(row=0, column=0, sticky="ew", padx=8, pady=6)
        btn_row.columnconfigure(1, weight=1)

        ttk.Button(btn_row, text="添加本地图片", command=self._add_files).grid(row=0, column=0, padx=(0, 8))
        self.url_add_var = tk.StringVar()
        ttk.Entry(btn_row, textvariable=self.url_add_var).grid(row=0, column=1, sticky="ew", padx=(0, 8))
        ttk.Button(btn_row, text="添加URL", command=self._add_url).grid(row=0, column=2, padx=(0, 8))
        ttk.Button(btn_row, text="移除选中", command=self._remove_selected_images).grid(row=0, column=3, padx=(0, 8))
        ttk.Button(btn_row, text="清空", command=self._clear_images).grid(row=0, column=4)

        ttk.Label(btn_row, text="本地图片编码").grid(row=0, column=5, padx=(12, 8))
        self.local_image_encoding_var = tk.StringVar(value=LOCAL_IMAGE_ENCODING_OPTIONS[0][1])
        encoding_combo = ttk.Combobox(
            btn_row,
            textvariable=self.local_image_encoding_var,
            values=[label for _, label in LOCAL_IMAGE_ENCODING_OPTIONS],
            width=14,
            state="readonly",
        )
        encoding_combo.grid(row=0, column=6, padx=(0, 0))

        self.images_list = tk.Listbox(imgs_frame, height=5, selectmode=tk.EXTENDED)
        self.images_list.grid(row=1, column=0, sticky="nsew", padx=8, pady=(0, 8))

        action_frame = ttk.Frame(self)
        action_frame.grid(row=1, column=0, columnspan=2, sticky="ew", padx=10, pady=6)
        action_frame.columnconfigure(0, weight=1)

        self.run_btn = ttk.Button(action_frame, text="开始生成", command=self._run)
        self.run_btn.grid(row=0, column=0, sticky="w")
        self.cancel_btn = ttk.Button(action_frame, text="取消", command=self._cancel_run, state="disabled")
        self.cancel_btn.grid(row=0, column=1, sticky="w", padx=(8, 0))
        ttk.Button(action_frame, text="清空输出", command=self._clear_output).grid(row=0, column=2, sticky="w", padx=(8, 0))
        ttk.Button(action_frame, text="刷新请求JSON", command=self._update_payload_preview).grid(
            row=0, column=3, sticky="w", padx=(8, 0)
        )

        out_left = ttk.LabelFrame(self, text="输出日志 / 进度")
        out_left.grid(row=2, column=0, sticky="nsew", padx=10, pady=(6, 10))
        out_left.rowconfigure(1, weight=1)
        out_left.columnconfigure(0, weight=1)

        self.progress_var = tk.IntVar(value=0)
        self.progress_bar = ttk.Progressbar(out_left, maximum=100, variable=self.progress_var)
        self.progress_bar.grid(row=0, column=0, sticky="ew", padx=8, pady=6)

        self.log_text = tk.Text(out_left, wrap="word")
        self.log_text.grid(row=1, column=0, sticky="nsew", padx=8, pady=(0, 8))

        out_right = ttk.LabelFrame(self, text="结果 / 预览")
        out_right.grid(row=2, column=1, sticky="nsew", padx=10, pady=(6, 10))
        out_right.columnconfigure(0, weight=0)
        out_right.columnconfigure(1, weight=1)
        out_right.rowconfigure(2, weight=1)

        self.task_id_var = tk.StringVar(value="")
        ttk.Label(out_right, text="Task ID").grid(row=0, column=0, sticky="w", padx=8, pady=6)
        ttk.Entry(out_right, textvariable=self.task_id_var, state="readonly").grid(
            row=0, column=1, sticky="ew", padx=(0, 8), pady=6
        )

        self.results_list = tk.Listbox(out_right, height=6)
        self.results_list.grid(row=1, column=0, columnspan=2, sticky="ew", padx=8, pady=(0, 6))
        self.results_list.bind("<<ListboxSelect>>", self._on_result_selected)

        preview_frame = ttk.Frame(out_right)
        preview_frame.grid(row=2, column=0, columnspan=2, sticky="nsew", padx=8, pady=(0, 8))
        preview_frame.columnconfigure(0, weight=1)
        preview_frame.rowconfigure(0, weight=1)

        self.preview_label = ttk.Label(preview_frame, text="（生成结果预览）", anchor="center")
        self.preview_label.grid(row=0, column=0, sticky="nsew")

        btns = ttk.Frame(out_right)
        btns.grid(row=3, column=0, columnspan=2, sticky="ew", padx=8, pady=(0, 8))
        ttk.Button(btns, text="打开链接", command=self._open_selected_url).grid(row=0, column=0, padx=(0, 8))
        ttk.Button(btns, text="保存图片", command=self._save_selected_image).grid(row=0, column=1, padx=(0, 8))

        payload_frame = ttk.LabelFrame(self, text="将要发送的请求 JSON（预览）")
        payload_frame.grid(row=3, column=0, columnspan=2, sticky="nsew", padx=10, pady=(0, 10))
        payload_frame.columnconfigure(0, weight=1)
        payload_frame.rowconfigure(0, weight=1)
        self.payload_text = tk.Text(payload_frame, height=8, wrap="none")
        self.payload_text.grid(row=0, column=0, sticky="nsew", padx=8, pady=8)

        # 自动刷新 payload 预览（轻量：仅绑定常用字段）
        for var in [self.host_var, self.model_var, self.aspect_ratio_var, self.image_size_var, self.mode_var]:
            var.trace_add("write", lambda *_: self._update_payload_preview())
        self.shut_progress_var.trace_add("write", lambda *_: self._update_payload_preview())
        self.local_image_encoding_var.trace_add("write", lambda *_: self._update_payload_preview())
        self.draw_path_var.trace_add("write", lambda *_: self._update_payload_preview())
        self.result_path_var.trace_add("write", lambda *_: self._update_payload_preview())

    def _append_log(self, msg: str) -> None:
        self.log_text.insert("end", msg + "\n")
        self.log_text.see("end")

    def _clear_output(self) -> None:
        self.log_text.delete("1.0", "end")
        self.progress_var.set(0)
        self.task_id_var.set("")
        self.results_list.delete(0, "end")
        self.preview_label.configure(text="（生成结果预览）", image="")
        self._preview_photo = None
        self._download_cache.clear()

    def _add_files(self) -> None:
        paths = fd.askopenfilenames(
            title="选择参考图片（可多选）",
            filetypes=[
                ("Image Files", "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.bmp"),
                ("All Files", "*.*"),
            ],
        )
        if not paths:
            return
        for p in paths:
            self._image_inputs.append(ImageInputItem(kind="file", value=p))
        self._refresh_images_list()
        self._update_payload_preview()

    def _add_url(self) -> None:
        url = self.url_add_var.get().strip()
        if not url:
            return
        self._image_inputs.append(ImageInputItem(kind="url", value=url))
        self.url_add_var.set("")
        self._refresh_images_list()
        self._update_payload_preview()

    def _remove_selected_images(self) -> None:
        selected = list(self.images_list.curselection())
        if not selected:
            return
        for idx in reversed(selected):
            if 0 <= idx < len(self._image_inputs):
                self._image_inputs.pop(idx)
        self._refresh_images_list()
        self._update_payload_preview()

    def _clear_images(self) -> None:
        self._image_inputs.clear()
        self._refresh_images_list()
        self._update_payload_preview()

    def _refresh_images_list(self) -> None:
        self.images_list.delete(0, "end")
        for item in self._image_inputs:
            if item.kind == "file":
                self.images_list.insert("end", f"[FILE] {item.value}")
            else:
                self.images_list.insert("end", f"[URL ] {item.value}")

    def _get_local_image_encoding_key(self) -> str:
        selected = self.local_image_encoding_var.get()
        for key, label in LOCAL_IMAGE_ENCODING_OPTIONS:
            if label == selected:
                return key
        return LOCAL_IMAGE_ENCODING_OPTIONS[0][0]

    def _collect_payload_fields(self) -> dict[str, Any]:
        return {
            "model": self.model_var.get().strip(),
            "prompt": self.prompt_text.get("1.0", "end").strip(),
            "aspectRatio": self.aspect_ratio_var.get().strip() or "auto",
            "imageSize": self.image_size_var.get().strip() or "1K",
            "shutProgress": bool(self.shut_progress_var.get()),
        }

    def _build_payload_preview(self) -> dict[str, Any]:
        payload: dict[str, Any] = dict(self._collect_payload_fields())
        urls: list[str] = []
        encoding = self._get_local_image_encoding_key()
        for item in self._image_inputs:
            if item.kind == "url":
                urls.append(item.value)
            elif item.kind == "file":
                urls.append(f"<file:{encoding}:{item.value}>")
        if urls:
            payload["urls"] = urls
        if self.mode_var.get() == "poll":
            payload["webHook"] = "-1"
        return payload

    def _update_payload_preview(self) -> None:
        try:
            payload = self._build_payload_preview()
            view = json.dumps(payload, ensure_ascii=False, indent=2)
        except Exception as e:
            view = f"// payload 生成失败：{e}"
        self.payload_text.delete("1.0", "end")
        self.payload_text.insert("1.0", view)

    def _set_running(self, running: bool) -> None:
        self.run_btn.configure(state=("disabled" if running else "normal"))
        self.cancel_btn.configure(state=("normal" if running else "disabled"))

    def _collect_run_request(self) -> RunRequest | None:
        host = self.host_var.get().strip()
        if not host:
            mb.showerror("参数错误", "Host 不能为空")
            return None
        api_key = self.api_key_var.get().strip()
        if not api_key:
            mb.showerror("参数错误", "API Key 不能为空")
            return None
        payload_fields = self._collect_payload_fields()
        if not payload_fields.get("prompt"):
            mb.showerror("参数错误", "Prompt 不能为空")
            return None
        return RunRequest(
            host=host,
            api_key=api_key,
            draw_path=self.draw_path_var.get().strip() or DEFAULT_DRAW_PATH,
            result_path=self.result_path_var.get().strip() or DEFAULT_RESULT_PATH,
            mode=self.mode_var.get(),
            payload_fields=payload_fields,
            image_inputs=tuple(self._image_inputs),
            local_image_encoding=self._get_local_image_encoding_key(),
        )

    def _run(self) -> None:
        if self._worker and self._worker.is_alive():
            return

        req = self._collect_run_request()
        if not req:
            return

        draw_path = req.draw_path
        result_path = req.result_path
        mode = req.mode

        self._clear_output()
        self._set_running(True)
        self._cancel.clear()

        def worker() -> None:
            try:
                client = NanoBananaClient(req.host, req.api_key, draw_path=draw_path, result_path=result_path)
                payload = dict(req.payload_fields)
                urls: list[str] = []
                for item in req.image_inputs:
                    if item.kind == "url":
                        urls.append(item.value)
                        continue
                    if item.kind == "file":
                        if req.local_image_encoding == "base64":
                            urls.append(file_to_base64(item.value))
                        else:
                            urls.append(file_to_data_url(item.value))
                        continue
                if urls:
                    payload["urls"] = urls
                if mode == "poll":
                    payload["webHook"] = "-1"

                self._queue.put({"type": "log", "message": f"POST {draw_path} mode={mode}"})
                if mode == "stream":
                    for event in client.draw_stream(payload, cancel=self._cancel):
                        self._queue.put({"type": "event", "event": event})
                else:
                    task_id = client.create_task(payload)
                    self._queue.put({"type": "task_id", "task_id": task_id})
                    while not self._cancel.is_set():
                        data = client.get_result(task_id)
                        self._queue.put({"type": "poll", "data": data})
                        status = str(data.get("status", "")).lower()
                        if status and status != "running":
                            break
                        time.sleep(2)
            except Exception as e:
                self._queue.put({"type": "error", "message": str(e)})
            finally:
                self._queue.put({"type": "done"})

        self._worker = threading.Thread(target=worker, daemon=True)
        self._worker.start()

    def _cancel_run(self) -> None:
        self._cancel.set()
        self._append_log("已请求取消…（等待当前网络请求返回）")

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
        if t == "preview":
            self._show_preview(msg.get("content"))
            return
        if t == "log":
            self._append_log(str(msg.get("message", "")))
            return
        if t == "error":
            self._append_log("ERROR: " + str(msg.get("message", "")))
            return
        if t == "task_id":
            self.task_id_var.set(str(msg.get("task_id", "")))
            self._append_log(f"任务ID: {self.task_id_var.get()}")
            return
        if t == "event":
            event = msg.get("event")
            if isinstance(event, dict):
                self._ingest_event_like(event, raw_prefix="stream")
            return
        if t == "poll":
            data = msg.get("data")
            if isinstance(data, dict):
                self._ingest_event_like(data, raw_prefix="poll")
            return
        if t == "done":
            self._set_running(False)
            return

    def _ingest_event_like(self, data: dict[str, Any], *, raw_prefix: str) -> None:
        try:
            raw = json.dumps(data, ensure_ascii=False)
        except Exception:
            raw = str(data)
        self._append_log(f"[{raw_prefix}] {raw}")

        progress = data.get("progress")
        if isinstance(progress, int):
            self.progress_var.set(max(0, min(100, progress)))
        elif isinstance(progress, float):
            self.progress_var.set(max(0, min(100, int(progress))))

        task_id = data.get("id")
        if task_id and not self.task_id_var.get():
            self.task_id_var.set(str(task_id))

        status = data.get("status")
        if status:
            self.master.title(f"Nano Banana API 本地测试 GUI - {status}")

        results = data.get("results")
        if isinstance(results, list) and results:
            self._render_results(results)

    def _render_results(self, results: list[Any]) -> None:
        self.results_list.delete(0, "end")
        for i, item in enumerate(results):
            if not isinstance(item, dict):
                continue
            url = str(item.get("url", ""))
            content = str(item.get("content", ""))
            label = f"{i + 1}. {content[:30]} | {url}"
            self.results_list.insert("end", label)
        self.results_list.selection_clear(0, "end")
        if self.results_list.size() > 0:
            self.results_list.selection_set(0)
            self._on_result_selected()

    def _get_selected_result_url(self) -> str | None:
        idxs = self.results_list.curselection()
        if not idxs:
            return None
        idx = idxs[0]
        line = self.results_list.get(idx)
        if "|" not in line:
            return None
        return line.split("|", 1)[1].strip()

    def _open_selected_url(self) -> None:
        url = self._get_selected_result_url()
        if not url:
            return
        import webbrowser

        webbrowser.open(url)

    def _save_selected_image(self) -> None:
        url = self._get_selected_result_url()
        if not url:
            return
        default_ext = os.path.splitext(url.split("?", 1)[0])[1] or ".png"
        path = fd.asksaveasfilename(
            title="保存图片",
            defaultextension=default_ext,
            filetypes=[("Image", f"*{default_ext}"), ("All Files", "*.*")],
        )
        if not path:
            return
        try:
            content = self._download_cache.get(url)
            if content is None:
                resp = requests.get(url, timeout=(10, 60))
                resp.raise_for_status()
                content = resp.content
                self._download_cache[url] = content
            with open(path, "wb") as f:
                f.write(content)
            mb.showinfo("保存成功", f"已保存到：{path}")
        except Exception as e:
            mb.showerror("保存失败", str(e))

    def _on_result_selected(self, _event: Any | None = None) -> None:
        url = self._get_selected_result_url()
        if not url:
            return

        def worker() -> None:
            try:
                content = self._download_cache.get(url)
                if content is None:
                    resp = requests.get(url, timeout=(10, 60))
                    resp.raise_for_status()
                    content = resp.content
                    self._download_cache[url] = content
                self._queue.put({"type": "preview", "url": url, "content": content})
            except Exception as e:
                self._queue.put({"type": "log", "message": f"预览下载失败: {e}"})

        threading.Thread(target=worker, daemon=True).start()

    def _show_preview(self, content: Any) -> None:
        if not isinstance(content, (bytes, bytearray)):
            return
        try:
            img = Image.open(io.BytesIO(content))
        except Exception:
            return
        max_w = max(300, self.preview_label.winfo_width())
        max_h = max(300, self.preview_label.winfo_height())
        img.thumbnail((max_w, max_h))
        self._preview_photo = ImageTk.PhotoImage(img)
        self.preview_label.configure(image=self._preview_photo, text="")


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
