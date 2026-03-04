# Wool 网关统一 API 调用规范（OpenAI 兼容）

> 目标：对外提供**现代主流**、**低接入成本**的统一调用方式。优先遵循 OpenAI API 的路径与结构（OpenAI-Compatible），并在必要处提供少量可扩展字段以承载不同上游能力差异。

## 1. 基础约定

### 1.1 Base URL 与版本

- 统一前缀：`/v1`
- 示例：`https://<your-domain>/v1`

### 1.2 认证

所有 `/v1/*` 接口统一使用 Bearer Token：

```
Authorization: Bearer <YOUR_API_KEY>
Content-Type: application/json
```

### 1.3 模型发现（强烈建议不要硬编码）

- `GET /v1/models`：列出当前 API Key 可用模型
- `GET /v1/models/{model}`：获取指定模型信息

### 1.4 错误响应（OpenAI 风格）

非 2xx 时返回：

```json
{
  "error": {
    "message": "具体错误信息",
    "type": "invalid_request_error | authentication_error | permission_error | not_found_error | rate_limit_error | server_error"
  }
}
```

## 2. 图片生成（Banana 等）

> 模型专项文档：`api文档/Wool 网关 Banana 调用文档.md`

### 2.1 创建图片生成任务

`POST /v1/images/generations`

#### 请求字段（兼容 + 扩展）

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `model` | string | 是 | 模型 ID（从 `/v1/models` 获取）。Banana 模型见 2.3 |
| `prompt` | string | 是 | 提示词 |
| `resolution` | string | 否 | 分辨率等级：`1K` / `2K` / `4K`（默认 `1K`） |
| `aspect_ratio` | string | 否 | 画幅比例（默认 `1:1`；Banana 支持 `auto`） |
| `urls` | string[] | 否 | 参考图（URL 或 data URL）列表 |
| `image` | string | 否 | 单张参考图（URL 或 data URL），等价于 `urls[0]` |
| `seed` | number | 否 | 随机种子（不同模型支持度不同） |
| `watermark` | boolean | 否 | 是否水印（不同模型支持度不同） |
| `user` | string | 否 | 调用方自定义用户标识（用于审计/追踪） |

> 说明：当前网关图片生成仅输出 1 张图（`n` / `response_format` 仅保留兼容，不作为对外能力承诺）。

#### 响应

```json
{
  "created": 1730000000,
  "data": [{ "url": "https://.../objects/..." }],
  "task_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

### 2.2 查询任务状态 / 获取结果

> 图片/视频生成会在网关侧落库，可用于追踪与二次获取（注意产物 URL 可能为签名 URL，会过期）。

- `GET /v1/generations/{id}`：查询单个任务
- `GET /v1/generations?type=image&page=1&page_size=20`：分页列出任务

### 2.3 Banana（对外模型层约定）

对外仅暴露 3 个模型（模型层）：

- `nano-banana-fast`
- `nano-banana`
- `nano-banana-pro`

> `nano-banana-pro-*`（带后缀）属于**源头层/上游变体**，通过网关内部 `model_providers.upstream_model_name` 做多源头调度，不建议对外暴露。

### 2.4 Banana 示例（curl）

```bash
curl -X POST "https://<your-domain>/v1/images/generations" \
  -H "Authorization: Bearer <YOUR_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nano-banana-pro",
    "prompt": "一只可爱的猫在草地上玩耍，写实风格",
    "resolution": "2K",
    "aspect_ratio": "16:9",
    "seed": 42,
    "watermark": false
  }'
```

## 3. 视频生成（Sora2 等）

> 模型专项文档：`api文档/Wool 网关 Sora2 调用文档.md`

### 3.1 创建视频生成任务

`POST /v1/videos/generations`

常用字段：

- `model`（必填）
- `prompt`（必填）
- `aspect_ratio`（默认 `9:16`）
- `duration`（默认 `10`）
- `size`（默认 `small`）
- `image_url`（可选参考图）

### 3.2 查询任务

同 2.2：

- `GET /v1/generations/{id}`
- `GET /v1/generations?type=video&page=1&page_size=20`

## 4. 文本能力（Chat/Completions/Embeddings）

网关按 OpenAI 路由提供：

- `POST /v1/chat/completions`（支持 `stream=true` SSE）
- `POST /v1/completions`
- `POST /v1/embeddings`

建议客户端优先使用 OpenAI-Compatible SDK（设置 `base_url/baseURL` 指向你的网关域名，并使用网关 API Key）。

## 5. 面向后续模型接入的约束（对内）

为保持对外 API 稳定、可持续：

- **对外以 `model` 为唯一选择器**，不让调用方感知“厂商/源头/账号池”等内部细节
- 上游差异通过内部配置消化：`channels` / `model_providers.upstream_model_name` / resource-pool adapters
- 逆向号池（resource-pool）建议在源头层（`channels.config`）配置：`resource_pool_url`（资源池服务地址，可独立部署）与 `resource_pool_provider`（banana/seedream/sora2/jimeng）；未配置时回退到环境变量 `RESOURCE_POOL_URL`，provider 则尝试从 `upstream_model_name` / `model` / `channel.name` 推断
- 新模型优先复用已有字段；确需新增参数时，优先在同一 endpoint 下扩展**少量通用字段**，避免“每家厂商一套请求结构”
