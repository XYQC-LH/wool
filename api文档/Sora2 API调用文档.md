# Sora2 API 调用文档

## 目录
- [基础信息](#基础信息)
- [Sora2视频接口](#sora2视频接口)
- [上传角色接口](#上传角色接口)
- [从原视频创建角色接口](#从原视频创建角色接口)
- [获取结果接口](#获取结果接口)
- [补充说明](#补充说明)

---

## 基础信息

### Host地址
- **海外地址**: `https://grsaiapi.com`
- **国内直连地址**: `https://grsai.dakka.com.cn`

### 使用方式
完整的API请求地址格式为：`Host + 接口路径`

例如：
```
https://grsai.dakka.com.cn/v1/video/sora-video
```

### 请求头
所有接口都需要在请求头中携带以下信息：

```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer apikey"
}
```

---

## Sora2视频接口

### 接口信息
- **接口路径**: `/v1/video/sora-video`
- **请求方式**: `POST`
- **响应方式**: `stream` 或 `回调接口`

### 完整请求URL
```
https://grsai.dakka.com.cn/v1/video/sora-video
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| model | string | 是 | 支持的模型：`sora-2` |
| prompt | string | 是 | 视频生成的提示词 |
| url | string | 否 | 参考图URL或Base64 |
| aspectRatio | string | 否 | 输出视频比例，支持：`9:16`、`16:9`，默认 `9:16` |
| duration | number | 否 | 视频时长（秒）：`10`、`15`，默认 `10` |
| remixTargetId | string | 否 | 视频续作的目标id，格式：`s_xxxxxxxxx` |
| size | string | 否 | 视频清晰度：`small`、`large`，默认 `small` |
| webHook | string | 否 | 进度与结果的回调链接 |
| shutProgress | boolean | 否 | 关闭进度回复，直接回复最终结果，建议搭配webHook使用，默认 `false` |

### 请求示例

```bash
curl -X POST "https://grsai.dakka.com.cn/v1/video/sora-video" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "sora-2",
    "prompt": "A cute cat playing on the grass",
    "url": "https://example.com/example.png",
    "aspectRatio": "16:9",
    "duration": 10,
    "remixTargetId": "",
    "size": "small",
    "webHook": "https://example.com/callback",
    "shutProgress": false
  }'
```

### JavaScript示例

```javascript
const response = await fetch('https://grsai.dakka.com.cn/v1/video/sora-video', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-api-key'
  },
  body: JSON.stringify({
    model: 'sora-2',
    prompt: 'A cute cat playing on the grass',
    url: 'https://example.com/example.png',
    aspectRatio: '16:9',
    duration: 10,
    size: 'small'
  })
});

// 流式响应处理
const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;
  
  const chunk = decoder.decode(value);
  console.log(chunk);
}
```

### Python示例

```python
import requests

url = "https://grsai.dakka.com.cn/v1/video/sora-video"
headers = {
    "Content-Type": "application/json",
    "Authorization": "Bearer your-api-key"
}
payload = {
    "model": "sora-2",
    "prompt": "A cute cat playing on the grass",
    "url": "https://example.com/example.png",
    "aspectRatio": "16:9",
    "duration": 10,
    "size": "small",
    "webHook": "https://example.com/callback"
}

response = requests.post(url, headers=headers, json=payload, stream=True)

# 处理流式响应
for line in response.iter_lines():
    if line:
        print(line.decode('utf-8'))
```

### 响应参数

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | string | 任务ID，webHook回调可以用该id来对应数据 |
| results | array | 视频结果数组 |
| results[].url | string | 视频URL（有效期为2小时） |
| results[].removeWatermark | boolean | 是否已去水印 |
| results[].pid | string | 视频续作的目标id，格式：`s_xxxxxxxxx` |
| progress | number | 任务进度，范围：0~100 |
| status | string | 任务状态：`running`（进行中）、`succeeded`（成功）、`failed`（失败） |
| failure_reason | string | 失败原因：`output_moderation`（输出违规）、`input_moderation`（输入违规）、`error`（其他错误） |
| error | string | 失败详细信息 |

### 响应示例

```json
{
  "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b",
  "results": [
    {
      "url": "https://example.com/example.mp4",
      "removeWatermark": true,
      "pid": "s_xxxxxxxxxxxxxxx"
    }
  ],
  "progress": 100,
  "status": "succeeded",
  "failure_reason": "",
  "error": ""
}
```

---

## 上传角色接口

### 接口信息
- **接口路径**: `/v1/video/sora-upload-character`
- **请求方式**: `POST`
- **响应方式**: `stream` 或 `回调接口`

### 完整请求URL
```
https://grsai.dakka.com.cn/v1/video/sora-upload-character
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| url | string | 否 | 角色视频URL或Base64 |
| timestamps | string | 否 | 角色视频范围，格式：`开始秒数,结束秒数`，例如：`0,3`，最多3秒 |
| webHook | string | 否 | 进度与结果的回调链接 |
| shutProgress | boolean | 否 | 关闭进度回复，直接回复最终结果，建议搭配webHook使用，默认 `false` |

### 请求示例

```bash
curl -X POST "https://grsai.dakka.com.cn/v1/video/sora-upload-character" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "url": "https://example.com/example.mp4",
    "timestamps": "0,3",
    "webHook": "https://example.com/callback"
  }'
```

### JavaScript示例

```javascript
const response = await fetch('https://grsai.dakka.com.cn/v1/video/sora-upload-character', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-api-key'
  },
  body: JSON.stringify({
    url: 'https://example.com/example.mp4',
    timestamps: '0,3',
    webHook: 'https://example.com/callback'
  })
});
```

### 响应参数

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | string | 任务ID，webHook回调可以用该id来对应数据 |
| results | array | 角色结果数组 |
| results[].character_id | string | 角色ID，在提示词里@该character_id即可使用 |
| progress | number | 任务进度，范围：0~100 |
| status | string | 任务状态：`running`（进行中）、`succeeded`（成功）、`failed`（失败） |
| failure_reason | string | 失败原因 |
| error | string | 失败详细信息 |

### 响应示例

```json
{
  "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b",
  "results": [
    {
      "character_id": "character.name"
    }
  ],
  "progress": 100,
  "status": "succeeded",
  "failure_reason": "",
  "error": ""
}
```

---

## 从原视频创建角色接口

### 接口信息
- **接口路径**: `/v1/video/sora-create-character`
- **请求方式**: `POST`
- **响应方式**: `stream` 或 `回调接口`

### 完整请求URL
```
https://grsai.dakka.com.cn/v1/video/sora-create-character
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| pid | string | 是 | 原视频id，参考生成视频后返回结果的pid值：`s_xxxxxxxxx` |
| webHook | string | 否 | 进度与结果的回调链接 |
| shutProgress | boolean | 否 | 关闭进度回复，直接回复最终结果，建议搭配webHook使用，默认 `false` |

### 请求示例

```bash
curl -X POST "https://grsai.dakka.com.cn/v1/video/sora-create-character" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "pid": "s_xxxxxxxxxxxxxxx",
    "webHook": "https://example.com/callback"
  }'
```

### JavaScript示例

```javascript
const response = await fetch('https://grsai.dakka.com.cn/v1/video/sora-create-character', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-api-key'
  },
  body: JSON.stringify({
    pid: 's_xxxxxxxxxxxxxxx',
    webHook: 'https://example.com/callback'
  })
});
```

### 响应参数

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | string | 任务ID，webHook回调可以用该id来对应数据 |
| results | array | 角色结果数组 |
| results[].character_id | string | 角色ID，在提示词里@该character_id即可使用 |
| progress | number | 任务进度，范围：0~100 |
| status | string | 任务状态：`running`（进行中）、`succeeded`（成功）、`failed`（失败） |
| failure_reason | string | 失败原因 |
| error | string | 失败详细信息 |

### 响应示例

```json
{
  "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b",
  "results": [
    {
      "character_id": "character.name"
    }
  ],
  "progress": 100,
  "status": "succeeded",
  "failure_reason": "",
  "error": ""
}
```

---

## 获取结果接口

### 接口信息
- **接口路径**: `/v1/draw/result`
- **请求方式**: `POST`
- **响应方式**: `JSON`

### 完整请求URL
```
https://grsai.dakka.com.cn/v1/draw/result
```

### 接口说明
该接口用于单独获取任务结果，适用于使用webHook参数设置为"-1"的场景。

### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | string | 是 | 任务ID |

### 请求示例

```bash
curl -X POST "https://grsai.dakka.com.cn/v1/draw/result" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b"
  }'
```

### JavaScript示例

```javascript
const response = await fetch('https://grsai.dakka.com.cn/v1/draw/result', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-api-key'
  },
  body: JSON.stringify({
    id: 'f44bcf50-f2d0-4c26-a467-26f2014a771b'
  })
});

const result = await response.json();
console.log(result);
```

### 响应参数

| 参数名 | 类型 | 说明 |
|--------|------|------|
| code | number | 状态码：`0`成功，`-22`任务不存在 |
| msg | string | 状态信息 |
| data | object | 视频结果，参考上方的视频结果的数据格式 |
| data.id | string | 任务ID |
| data.results | array | 视频结果数组 |
| data.progress | number | 任务进度 |
| data.status | string | 任务状态 |
| data.failure_reason | string | 失败原因 |
| data.error | string | 失败详细信息 |

### 响应示例

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b",
    "results": [
      {
        "url": "https://example.com/example.mp4",
        "removeWatermark": true,
        "pid": "s_xxxxxxxxxxxxxxx"
      }
    ],
    "progress": 100,
    "status": "succeeded",
    "failure_reason": "",
    "error": ""
  }
}
```

---

## 补充说明

### 响应方式说明

#### 1. 流式响应（Stream）
- 默认响应方式
- 实时返回任务进度和结果
- 需要客户端持续读取流式数据
- 适用于需要实时反馈的场景

#### 2. 回调接口（WebHook）
- 在请求参数中设置`webHook`参数
- 服务器会通过POST请求将进度和结果发送到指定的回调地址
- 回调请求头：`Content-Type: application/json`
- 适用于异步处理和服务器端集成的场景

### webHook参数的特殊用法

#### 场景1：使用回调获取结果
```json
{
  "webHook": "https://example.com/callback"
}
```
服务器会将进度和结果通过POST请求发送到该地址。

#### 场景2：使用轮询获取结果
```json
{
  "webHook": "-1"
}
```
接口会立即返回一个任务ID，然后可以使用`/v1/draw/result`接口轮询获取结果。

**立即返回示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": "f44bcf50-f2d0-4c26-a467-26f2014a771b"
  }
}
```

### shutProgress参数说明

该参数用于控制是否返回进度信息：

- `false`（默认）：返回所有进度信息，包括进行中的状态
- `true`：不返回进度信息，直接返回最终结果，建议搭配`webHook`使用

### 错误处理

#### 失败原因（failure_reason）
- `output_moderation`：输出内容违规
- `input_moderation`：输入内容违规
- `error`：其他错误

**注意**：当生成失败时，会返还积分。

#### 响应状态码
- `0`：成功
- `-22`：任务不存在（用于获取结果接口）

### 注意事项

1. **API Key**：所有请求都需要在`Authorization`请求头中携带有效的API Key
2. **视频URL有效期**：生成的视频URL有效期为2小时，请及时下载或保存
3. **视频时长**：`duration`参数只支持10秒或15秒
4. **视频比例**：`aspectRatio`只支持`9:16`和`16:9`两种格式
5. **角色视频范围**：上传角色时，`timestamps`参数最多支持3秒的视频片段
6. **流式响应处理**：使用流式响应时，客户端需要正确处理分块传输的数据
7. **积分返还**：生成失败时会自动返还积分

### 完整示例流程

#### 流程1：使用流式响应生成视频
```javascript
const response = await fetch('https://grsai.dakka.com.cn/v1/video/sora-video', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-api-key'
  },
  body: JSON.stringify({
    model: 'sora-2',
    prompt: 'A beautiful sunset over the ocean',
    aspectRatio: '16:9',
    duration: 10
  })
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;
  
  const chunk = decoder.decode(value);
  const data = JSON.parse(chunk);
  console.log(`进度: ${data.progress}%, 状态: ${data.status}`);
  
  if (data.status === 'succeeded') {
    console.log('视频URL:', data.results[0].url);
    console.log('视频PID:', data.results[0].pid);
  }
}
```

#### 流程2：使用回调生成视频
```javascript
// 1. 发起请求
const response = await fetch('https://grsai.dakka.com.cn/v1/video/sora-video', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-api-key'
  },
  body: JSON.stringify({
    model: 'sora-2',
    prompt: 'A beautiful sunset over the ocean',
    webHook: 'https://your-server.com/callback'
  })
});

const result = await response.json();
console.log('任务ID:', result.data.id);

// 2. 在回调服务器中处理结果
// https://your-server.com/callback
app.post('/callback', (req, res) => {
  const data = req.body;
  console.log('任务ID:', data.id);
  console.log('进度:', data.progress);
  console.log('状态:', data.status);
  
  if (data.status === 'succeeded') {
    console.log('视频URL:', data.results[0].url);
    console.log('视频PID:', data.results[0].pid);
  }
  
  res.status(200).send('OK');
});
```

#### 流程3：使用轮询获取结果
```javascript
// 1. 发起请求，设置webHook为"-1"
const response = await fetch('https://grsai.dakka.com.cn/v1/video/sora-video', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-api-key'
  },
  body: JSON.stringify({
    model: 'sora-2',
    prompt: 'A beautiful sunset over the ocean',
    webHook: '-1'
  })
});

const { data } = await response.json();
const taskId = data.id;

// 2. 轮询获取结果
async function pollResult(taskId) {
  while (true) {
    const resultResponse = await fetch('https://grsai.dakka.com.cn/v1/draw/result', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer your-api-key'
      },
      body: JSON.stringify({
        id: taskId
      })
    });
    
    const result = await resultResponse.json();
    
    if (result.data.status === 'succeeded') {
      console.log('生成成功！');
      console.log('视频URL:', result.data.results[0].url);
      console.log('视频PID:', result.data.results[0].pid);
      break;
    } else if (result.data.status === 'failed') {
      console.log('生成失败:', result.data.error);
      break;
    } else {
      console.log('生成中，进度:', result.data.progress + '%');
      await new Promise(resolve => setTimeout(resolve, 3000)); // 等待3秒
    }
  }
}

pollResult(taskId);
```

---

## 联系支持

如有问题或需要帮助，请联系技术支持团队。
