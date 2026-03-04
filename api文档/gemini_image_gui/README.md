# 图片生成本地 GUI

用于测试：

- `gemini-3-pro-image-preview:generateContent`（官方格式）
- `nano-banana`（`/v1/images/edits` 兼容格式）

## 1. 环境要求

- Windows / macOS / Linux
- Python 3.10+

## 2. 安装与运行（PowerShell）

```powershell
cd "c:/code/my/Wool/api文档/gemini_image_gui"
python -m venv ".venv"
.\.venv\Scripts\Activate.ps1
pip install -r "requirements.txt"
python "app.py"
```

如需测试 Nano-banana：

```powershell
python "nano_banana_gui.py"
```

## 3. 使用说明（Gemini 官方格式）

- **Host**：默认 `https://api.kuai.host`
- **Model**：默认 `gemini-3-pro-image-preview`
- **API Key**：在界面填写；也可用环境变量 `KUAI_API_KEY` / `GEMINI_API_KEY`
- **鉴权方式**：默认 Bearer；勾选“使用 ?key=”可改成 query key
- **参考图**：选择本地图片，自动转为 base64 inline_data
- **自适应尺寸**：`Aspect Ratio` / `Image Size` 选择“auto”即可由服务端按默认规则处理

## 4. 使用说明（Nano-banana Edits 兼容）

- **Host**：默认 `https://ai.t8star.cn`
- **Endpoint**：默认 `/v1/images/edits`（也可改 `/v1/images/generations`）
- **Model**：默认 `nano-banana`
- **API Key**：在界面填写；也可用环境变量 `NANO_BANANA_API_KEY` / `KUAI_API_KEY`
- **Response Format**：`url` 或 `b64_json`
- **参考图**：支持多图；允许不选图，仅用 Prompt
- **Aspect Ratio / Image Size**：选择“auto(不传)”则不发送该字段

## 5. 本地配置（可选）

复制 `config.example.json` 为 `config.local.json`，写入你的 Key：

```json
{
  "host": "https://api.kuai.host",
  "model": "gemini-3-pro-image-preview",
  "api_key": "<YOUR_API_KEY>",
  "use_query_key": false,
  "nano_host": "https://ai.t8star.cn",
  "nano_endpoint": "/v1/images/edits",
  "nano_model": "nano-banana",
  "nano_api_key": "<YOUR_API_KEY>",
  "nano_response_format": "url"
}
```

`config.local.json` 已加入 `.gitignore`，不会提交到仓库。
