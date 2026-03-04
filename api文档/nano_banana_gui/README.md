# Nano Banana 本地调用 GUI（多图输入）

基于 `Grsai Nano Banana API调用文档.md` 的本地测试小工具：支持**多图输入**（本地图片会自动转成 data URL）、支持 **Stream 流式**与**轮询**两种方式调用，并在界面里查看进度与结果。

## 1. 环境要求

- Windows / macOS / Linux
- Python 3.10+（你本机是 Python 3.14.x 也没问题）

## 2. 安装与运行（PowerShell）

```powershell
cd "c:/code/my/Wool/api文档/nano_banana_gui"
python -m venv ".venv"
.\.venv\Scripts\Activate.ps1
pip install -r "requirements.txt"
python "app.py"
```

## 3. 使用说明

- **Host**：默认 `https://grsai.dakka.com.cn`，也可切换 `https://grsaiapi.com` 或自行输入。
- **API Key**：填 `Authorization: Bearer <API_KEY>` 中的 `<API_KEY>`（界面不会写入磁盘）。
  - 也可以提前设置环境变量 `GRSAI_API_KEY`，GUI 会自动带出。
  - 或者在 `nano_banana_gui/config.local.json` 里配置（不会提交到仓库，已加入 `.gitignore`）。
- **多图输入**：
  - “添加本地图片”支持多选；会自动转成 `data:<mime>;base64,...` 形式塞进 `urls`。
  - 可在界面里切换“本地图片编码”：`data URL（推荐）` 或 `纯 base64`（用于兼容不同上游的入参要求）。
  - “添加URL”可直接追加远程图片链接。
- **调用模式**：
  - **Stream 流式**：直接读取每行 JSON 事件，更新进度与结果。
  - **轮询**：提交任务时自动设置 `webHook = "-1"` 拿到 `id`，再调用 `/v1/draw/result` 轮询。

## 4. 常见问题

- 生成图片 URL 通常有有效期（文档写的是 2 小时），请及时下载保存。
- 如果你填了公司代理/网关地址但路径不同，可在界面里改 `Draw Path`、`Result Path`。

## 5. 本地配置（可选）

复制 `nano_banana_gui/config.example.json` 为 `nano_banana_gui/config.local.json`，填入你的 `api_key`：

```json
{
  "host": "https://grsai.dakka.com.cn",
  "api_key": "<YOUR_API_KEY>"
}
```
