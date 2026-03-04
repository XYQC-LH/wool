# Grsai Nano Banana API调用文档

## 目录
- [API概述](#api概述)
- [节点信息](#节点信息)
- [Gemini官方接口格式支持](#gemini官方接口格式支持)
- [Nano Banana绘画接口](#nano-banana绘画接口)
- [获取结果接口](#获取结果接口)
- [使用示例](#使用示例)

---

## API概述

Nano Banana是一个强大的AI绘画API，支持多种模型和响应方式，可以通过流式响应或WebHook回调获取绘画结果。

---

## 节点信息

### Host地址

| 类型 | 地址 |
|------|------|
| 海外Host | `https://grsaiapi.com` |
| 国内直连Host | `https://grsai.dakka.com.cn` |

### 使用方式

完整URL格式：`Host + 接口路径`

**示例**：
```
https://grsai.dakka.com.cn/v1/draw/nano-banana
```

---

## Gemini官方接口格式支持

该API支持Gemini官方的接口格式，使用时只需：

1. 将基础地址替换为Grsai的地址
2. 将模型名称 `gemini-2.5-flash-image` 改为 `nano-banana-fast`

**Gemini兼容接口示例**：
```
POST https://grsai.dakka.com.cn/v1beta/models/nano-banana-fast:streamGenerateContent
```

---

## Nano Banana绘画接口

### 基本信息

- **接口路径**: `/v1/draw/nano-banana`
- **请求方式**: `POST`
- **响应方式**: Stream流式响应 或 WebHook回调

### 请求头 (Headers)

```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer apikey"
}
```

**说明**：
- `Authorization`: 使用您的API密钥进行认证

### 请求参数 (JSON)

```json
{
  "model": "nano-banana-fast",
  "prompt": "提示词",
  "aspectRatio": "auto",
  "imageSize": "1K",
  "urls": [
    "https://example.com/example.png"
  ],
  "webHook": "https://example.com/callback",
  "shutProgress": false
}
```

### 参数说明

| 参数名 | 必填 | 类型 | 说明 |
|--------|------|------|------|
| **model** | 是 | string | 支持的模型：<br/>- nano-banana-fast<br/>- nano-banana<br/>- nano-banana-pro<br/>- nano-banana-pro-vt<br/>- nano-banana-pro-cl<br/>- nano-banana-pro-vip<br/>- nano-banana-pro-4k-vip |
| **prompt** | 是 | string | 提示词，描述您想要生成的图像<br/>示例: "一只可爱的猫咪在草地上玩耍" |
| **aspectRatio** | 否 | string | 输出图像比例，支持：<br/>- auto（默认）<br/>- 1:1<br/>- 16:9<br/>- 9:16<br/>- 4:3<br/>- 3:4<br/>- 3:2<br/>- 2:3<br/>- 5:4<br/>- 4:5<br/>- 21:9 |
| **imageSize** | 否 | string | 输出图像大小，支持：<br/>- 1K（默认）<br/>- 2K<br/>- 4K<br/><br/>**注意**：<br/>- nano-banana-pro 系列支持不同尺寸<br/>- nano-banana-pro-vip 只支持 1K, 2K<br/>- nano-banana-pro-4k-vip 只支持 4K<br/>- 分辨率越高，生成时间越长 |
| **urls** | 否 | array | 参考图URL或Base64<br/>示例: `["https://example.com/example.png"]` |
| **webHook** | 否 | string | 进度与结果的回调链接<br/>- 如果提供webHook，结果会通过POST请求回调到该地址<br/>- 请求头: `Content-Type: application/json`<br/>- 如果不使用回调而使用轮询result接口，填 `"-1"`，接口会立即返回一个id |
| **shutProgress** | 否 | boolean | 是否关闭进度回复，直接回复最终结果<br/>- false（默认）：返回进度<br/>- true：关闭进度，建议搭配webHook使用 |

### 响应方式

接口默认使用Stream流式响应进行回复。如果提供了`webHook`参数，则使用Post请求回调的方式进行回复。

#### 1. Stream流式响应

直接通过HTTP流式响应获取进度和结果。

#### 2. WebHook回调响应

当提供`webHook`参数时，结果会POST回调到指定地址。

**立即响应（返回id）**：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b"
  }
}
```

**参数说明**：

| 参数名 | 类型 | 说明 |
|--------|------|------|
| code | number | 状态码：0为成功 |
| msg | string | 状态信息 |
| data | object | 数据对象 |
| data.id | string | 程序任务id，对应回调数据 |

### 响应参数 (Stream和WebHook通用)

```json
{
  "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b",
  "results": [
    {
      "url": "https://example.com/generated-image.jpg",
      "content": "这是一只可爱的猫咪在草地上玩耍"
    }
  ],
  "progress": 100,
  "status": "succeeded",
  "failure_reason": "",
  "error": ""
}
```

### 响应参数说明

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | string | 任务ID（webHook回调可以用该id来对应数据） |
| results | array | 结果数组 |
| results[].url | string | 生成的图片URL（有效期为2小时） |
| results[].content | string | 回复内容描述 |
| progress | number | 任务进度，范围0~100 |
| status | string | 任务状态：<br/>- running：进行中<br/>- succeeded：成功<br/>- failed：失败 |
| failure_reason | string | 失败原因：<br/>- output_moderation：输出违规<br/>- input_moderation：输入违规<br/>- error：其他错误<br/><br/>**提示**：当触发"error"时，可尝试重新提交任务 |
| error | string | 失败详细信息 |

---

## 获取结果接口

如果使用轮询方式获取结果，可以使用该接口。

### 基本信息

- **接口路径**: `/v1/draw/result`
- **请求方式**: `POST`

### 请求参数

```json
{
  "id": "xxxxx"
}
```

**参数说明**：
- `id`: 任务ID（来自绘画接口的返回）

### 响应结果

```json
{
  "code": 0,
  "data": {
    "id": "xxxxx",
    "results": [
      {
        "url": "https://example.com/example.png",
        "content": "这是一只可爱的猫咪在草地上玩耍"
      }
    ],
    "progress": 100,
    "status": "succeeded",
    "failure_reason": "",
    "error": ""
  },
  "msg": "success"
}
```

### 响应参数说明

| 参数名 | 类型 | 说明 |
|--------|------|------|
| code | number | 状态码：0成功，-22任务不存在 |
| msg | string | 状态信息 |
| data | object | 绘画结果，格式与绘画接口的响应参数相同 |

---

## 使用示例

### 示例1：使用Stream流式响应

```bash
curl -X POST "https://grsai.dakka.com.cn/v1/draw/nano-banana" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "nano-banana-fast",
    "prompt": "一只可爱的猫咪在草地上玩耍",
    "aspectRatio": "auto",
    "imageSize": "1K"
  }'
```

### 示例2：使用WebHook回调

```bash
curl -X POST "https://grsai.dakka.com.cn/v1/draw/nano-banana" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "nano-banana-pro",
    "prompt": "夕阳下的城市天际线",
    "aspectRatio": "16:9",
    "imageSize": "2K",
    "webHook": "https://your-domain.com/callback",
    "shutProgress": true
  }'
```

### 示例3：使用参考图

```bash
curl -X POST "https://grsai.dakka.com.cn/v1/draw/nano-banana" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "nano-banana-fast",
    "prompt": "在同样风格下绘制另一张图片",
    "urls": ["https://example.com/reference-image.png"],
    "aspectRatio": "1:1"
  }'
```

### 示例4：使用轮询方式获取结果

**步骤1：提交任务并获取id**

```bash
curl -X POST "https://grsai.dakka.com.cn/v1/draw/nano-banana" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "nano-banana-fast",
    "prompt": "科幻风格的未来城市",
    "webHook": "-1"
  }'
```

**响应**：
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b"
  }
}
```

**步骤2：轮询获取结果**

```bash
curl -X POST "https://grsai.dakka.com.cn/v1/draw/result" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b"
  }'
```

### Python示例代码

```python
import requests
import json

# 配置
API_HOST = "https://grsai.dakka.com.cn"
API_KEY = "your_api_key_here"

# 示例1：流式响应
def stream_generation():
    url = f"{API_HOST}/v1/draw/nano-banana"
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {API_KEY}"
    }
    payload = {
        "model": "nano-banana-fast",
        "prompt": "一只可爱的猫咪在草地上玩耍",
        "aspectRatio": "auto"
    }
    
    response = requests.post(url, headers=headers, json=payload, stream=True)
    
    for line in response.iter_lines():
        if line:
            try:
                data = json.loads(line)
                print(f"进度: {data.get('progress')}%")
                if data.get('status') == 'succeeded':
                    print("生成成功！")
                    for result in data.get('results', []):
                        print(f"图片URL: {result['url']}")
                        print(f"描述: {result['content']}")
                elif data.get('status') == 'failed':
                    print(f"生成失败: {data.get('error')}")
            except json.JSONDecodeError:
                continue

# 示例2：轮询方式
def poll_generation():
    url = f"{API_HOST}/v1/draw/nano-banana"
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {API_KEY}"
    }
    
    # 提交任务
    payload = {
        "model": "nano-banana-pro",
        "prompt": "夕阳下的城市天际线",
        "imageSize": "2K",
        "webHook": "-1"
    }
    
    response = requests.post(url, headers=headers, json=payload)
    result = response.json()
    
    if result['code'] == 0:
        task_id = result['data']['id']
        print(f"任务ID: {task_id}")
        
        # 轮询结果
        result_url = f"{API_HOST}/v1/draw/result"
        while True:
            poll_response = requests.post(result_url, headers=headers, json={"id": task_id})
            poll_result = poll_response.json()
            
            if poll_result['code'] == 0:
                data = poll_result['data']
                print(f"进度: {data['progress']}%")
                
                if data['status'] == 'succeeded':
                    print("生成成功！")
                    for result in data['results']:
                        print(f"图片URL: {result['url']}")
                        print(f"描述: {result['content']}")
                    break
                elif data['status'] == 'failed':
                    print(f"生成失败: {data['error']}")
                    break
            
            time.sleep(2)  # 每2秒轮询一次

if __name__ == "__main__":
    # 选择使用哪种方式
    # stream_generation()
    poll_generation()
```

---

## 注意事项

1. **图片URL有效期**：生成的图片URL有效期为2小时，请及时保存
2. **分辨率与时间**：分辨率越高，生成时间越长，请根据需求合理选择
3. **模型选择**：
   - `nano-banana-fast`：速度最快，适合快速生成
   - `nano-banana`：标准版本
   - `nano-banana-pro`系列：更高品质，支持更高分辨率
4. **错误重试**：当触发`error`类型错误时，建议重新提交任务以确保系统稳定性
5. **国内使用**：建议使用国内直连Host以获得更好的访问速度
6. **WebHook服务**：如果使用WebHook，确保您的回调服务器可以接收来自API服务器的POST请求

---

## 技术支持

如有问题，请参考API文档或联系技术支持团队。
