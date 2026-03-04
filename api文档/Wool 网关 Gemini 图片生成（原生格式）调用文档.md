# Wool 网关 Gemini 图片生成（原生格式）调用文档

本文面向 **Wool 网关的 API 使用者**，说明如何用 Gemini 原生 `generateContent` 接口生成图片，并控制**宽高比**与**清晰度**。

> 适用场景：需要使用 Gemini 官方格式（`generateContent`）直接调用图片生成能力。

---

## 1. 基础信息

- **Base URL**：`https://api.kuai.host`
- **Endpoint**：`POST /v1beta/models/gemini-3-pro-image-preview:generateContent`
- **Content-Type**：`application/json`

---

## 2. 认证方式（二选一）

> Wool 网关默认使用 Bearer Token；若你的渠道只支持 Gemini 原生 `key` 参数，可改用 query key。

### 2.1 Bearer Token（推荐）

```
Authorization: Bearer <YOUR_API_KEY>
Content-Type: application/json
```

### 2.2 Query Key（Gemini 原生）

```
POST https://api.kuai.host/v1beta/models/gemini-3-pro-image-preview:generateContent?key=<YOUR_API_KEY>
Content-Type: application/json
```

> 注意：**二选一即可**，不要同时使用。

---

## 3. 请求结构

### 3.1 请求体字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `contents` | array | 是 | 输入内容列表（通常只需 1 条 `role: user`） |
| `contents[].role` | string | 否 | 角色（建议 `user`） |
| `contents[].parts` | array | 是 | 内容片段 |
| `parts[].text` | string | 否 | 文本提示词 |
| `parts[].inline_data` | object | 否 | 参考图（Base64） |
| `inline_data.mime_type` | string | 是 | 例如 `image/jpeg`、`image/png` |
| `inline_data.data` | string | 是 | 图片 Base64（**不带** data URL 前缀） |
| `generationConfig` | object | 是 | 生成配置 |
| `generationConfig.responseModalities` | array | 是 | 响应模态，生成图片需包含 `IMAGE` |
| `generationConfig.imageConfig` | object | 是 | 图片配置 |
| `imageConfig.aspectRatio` | string | 是 | **宽高比**，如 `1:1`、`16:9`、`9:16` |
| `imageConfig.imageSize` | string | 否 | **清晰度/分辨率**（如 `1K`/`2K`/`4K`，以服务端支持为准） |

### 3.2 宽高比与清晰度说明

- **宽高比**：由 `imageConfig.aspectRatio` 控制，常见取值：`1:1`、`16:9`、`9:16`、`4:3`、`3:4`、`3:2`、`2:3`、`21:9` 等（以服务端实际支持为准）。
- **清晰度**：可通过 `imageConfig.imageSize` 控制（如 `1K`/`2K`/`4K`）。若不传则使用默认值。

---

## 4. 请求示例

### 4.1 文本生成图片（仅提示词）

```bash
curl -X POST "https://api.kuai.host/v1beta/models/gemini-3-pro-image-preview:generateContent" \
  -H "Authorization: Bearer <YOUR_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "一只可爱的猫在草地上玩耍，写实风格" }
        ]
      }
    ],
    "generationConfig": {
      "responseModalities": ["TEXT", "IMAGE"],
      "imageConfig": {
        "aspectRatio": "16:9",
        "imageSize": "1K"
      }
    }
  }'
```

### 4.2 带参考图（图片编辑/风格参考）

> `inline_data.data` 需要先把图片转为 Base64，且**不要**包含 `data:image/...;base64,` 前缀。

```bash
curl -X POST "https://api.kuai.host/v1beta/models/gemini-3-pro-image-preview:generateContent" \
  -H "Authorization: Bearer <YOUR_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "基于参考图生成同风格的插画" },
          {
            "inline_data": {
              "mime_type": "image/png",
              "data": "<BASE64_IMAGE>"
            }
          }
        ]
      }
    ],
    "generationConfig": {
      "responseModalities": ["IMAGE"],
      "imageConfig": {
        "aspectRatio": "1:1",
        "imageSize": "2K"
      }
    }
  }'
```

---

## 5. 备注与常见问题

- **必须包含 `IMAGE`**：`responseModalities` 中需包含 `IMAGE`，否则可能只返回文本。
- **Base64 格式**：必须是纯 Base64 字符串，不是 data URL。
- **参数兼容性**：若 `imageSize` 不被服务端识别，请移除该字段后重试。

