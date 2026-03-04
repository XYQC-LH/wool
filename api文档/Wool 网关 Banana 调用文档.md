# Wool 网关 Banana（Nano Banana）调用文档

本文面向 **Wool 网关的 API 使用者**，说明如何通过网关调用 Banana 系列图片生成模型。

通用约定（鉴权、错误结构、模型发现、任务查询）请先阅读：`api文档/Wool 网关统一API调用规范.md`。

## 1. 支持模型（对外）

对外仅暴露以下 3 个模型（`model` 字段取值）：

- `nano-banana-fast`
- `nano-banana`
- `nano-banana-pro`

> 说明：`nano-banana-pro-*`（带后缀）在 Wool 内部作为 **源头层/上游变体** 进行配置与调度，不建议对外暴露为可选模型。

## 2. 创建图片生成

### 2.1 接口

`POST /v1/images/generations`

### 2.2 请求参数（Banana 推荐用法）

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `model` | string | 是 | `nano-banana-fast` / `nano-banana` / `nano-banana-pro` |
| `prompt` | string | 是 | 提示词 |
| `resolution` | string | 否 | 输出分辨率等级：`1K` / `2K` / `4K`（默认 `1K`） |
| `aspect_ratio` | string | 否 | 输出画幅比例（默认 `1:1`），支持 `auto` |
| `urls` | string[] | 否 | 参考图列表：URL 或 data URL |
| `image` | string | 否 | 单张参考图：URL 或 data URL（等价于 `urls[0]`） |

#### 兼容字段说明（不建议依赖）

- `n`：网关侧会强制为 `1`（固定输出 1 张图）。
- `response_format`：网关侧固定返回 `url`（暂不提供 `b64_json`）。
- `size`：为兼容保留，但 Wool 的推荐做法是使用 `resolution` + `aspect_ratio`，避免与 OpenAI 的 `size=1024x1024` 语义混淆。

### 2.3 画幅比例建议值

Banana 上游支持多种比例，网关建议优先使用以下常见值：

- `auto`
- `1:1`
- `16:9`
- `9:16`
- `4:3`
- `3:4`
- `3:2`
- `2:3`
- `21:9`

### 2.4 请求示例（curl）

```bash
curl -X POST "https://<your-domain>/v1/images/generations" \
  -H "Authorization: Bearer <YOUR_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nano-banana-pro",
    "prompt": "一只可爱的猫在草地上玩耍，写实风格",
    "resolution": "2K",
    "aspect_ratio": "16:9",
    "urls": ["https://example.com/reference.png"]
  }'
```

### 2.5 响应示例

```json
{
  "created": 1730000000,
  "data": [
    {
      "url": "https://<your-domain>/objects/..."
    }
  ],
  "task_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

> 注意：`url` 可能是签名 URL，会过期；请按需下载并自行长期存储。

## 3. 任务查询（可选）

Banana 调用在成功时会同步返回结果，同时网关会记录任务，支持查询：

- `GET /v1/generations/{id}`：查询单个任务
- `GET /v1/generations?type=image&page=1&page_size=20`：分页列出任务

## 4. 常见错误与排查

- `authentication_error`：检查 `Authorization: Bearer <YOUR_API_KEY>` 是否正确。
- `permission_error`：该 API Key 未被授权访问当前 `model`（请在管理端为该 key 放行模型）。
- `invalid_request_error`：缺少 `model/prompt` 或参数格式错误。
- `server_error`：上游/资源池异常或生成失败（可降低 `resolution`、更换 `aspect_ratio`，或稍后重试）。

