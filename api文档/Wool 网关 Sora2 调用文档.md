# Wool 网关 Sora2（视频生成）调用文档

本文面向 **Wool 网关的 API 使用者**，说明如何通过网关调用 Sora2 视频生成能力。

通用约定（鉴权、错误结构、模型发现、任务查询）请先阅读：`api文档/Wool 网关统一API调用规范.md`。

## 1. 支持模型（对外）

Sora2 的 `model` 取值以 `GET /v1/models` 返回为准（由网关侧配置决定）。

> 建议网关侧将对外模型命名为 `sora2`（或 `sora-2`），并通过内部 `model_providers` 绑定到 Sora2 上游。

## 2. 创建视频生成

### 2.1 接口

`POST /v1/videos/generations`

### 2.2 请求参数

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `model` | string | 是 | 以 `/v1/models` 为准（建议 `sora2`） |
| `prompt` | string | 是 | 提示词 |
| `aspect_ratio` | string | 否 | 画幅比例：`9:16` / `16:9` / `1:1`（默认 `9:16`） |
| `duration` | number | 否 | 时长（秒）：`10` / `15`（默认 `10`） |
| `size` | string | 否 | 清晰度：`small` / `large`（默认 `small`） |
| `image_url` | string | 否 | 参考图：URL 或 data URL |

> 说明：当前网关对外不提供 webhook 回调；采用同步返回结果 + 任务查询的方式。

### 2.3 请求示例（curl）

```bash
curl -X POST "https://<your-domain>/v1/videos/generations" \
  -H "Authorization: Bearer <YOUR_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sora2",
    "prompt": "一只宇航员在月球上奔跑，电影级光影",
    "aspect_ratio": "16:9",
    "duration": 10,
    "size": "small"
  }'
```

### 2.4 响应示例

```json
{
  "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "status": "completed",
  "progress": 1,
  "created_at": 1730000000,
  "data": {
    "url": "https://<your-domain>/objects/...",
    "duration": 10
  }
}
```

> 注意：`url` 可能是签名 URL，会过期；请按需下载并自行长期存储。

## 3. 超时与重试建议

视频生成可能耗时较长（分钟级）。建议：

- 客户端将请求超时设置到 20 分钟以上；
- 如遇 `server_error` 或网络超时，可使用相同输入重试；
- 通过 `GET /v1/generations/{id}` 追踪任务（若你已保存 `id`）。

## 4. 任务查询（可选）

- `GET /v1/generations/{id}`：查询单个任务
- `GET /v1/generations?type=video&page=1&page_size=20`：分页列出任务

## 5. 常见错误与排查

- `authentication_error`：检查 `Authorization` 是否正确。
- `permission_error`：该 API Key 未被授权访问当前 `model`。
- `invalid_request_error`：缺少 `model/prompt` 或参数格式错误。
- `server_error`：上游/资源池异常或生成失败（可尝试降低 `duration/size`，或稍后重试）。

