# TokenHub API 使用文档

> 版本：V1.0  
> 最后更新：2025 年 5 月

本文档介绍 TokenHub 平台的 API 使用方法。

## 概述

TokenHub 提供完全兼容 OpenAI API 格式的接口，您可以直接使用现有的 OpenAI SDK 进行调用。

### 基础信息

- **API 端点**: `https://your-domain.com/v1`
- **认证方式**: Bearer Token (API Key)
- **请求格式**: JSON
- **响应格式**: JSON

## 快速开始

### 1. 获取 API Key

1. 登录 TokenHub 用户控制台
2. 进入 **API Key 管理** 页面
3. 点击 **创建新 Key**
4. 复制并保存您的 API Key（只显示一次）

### 2. 调用示例

#### cURL 示例

```bash
curl https://your-domain.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

#### Python 示例

```python
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://your-domain.com/v1"
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

#### JavaScript 示例

```javascript
import OpenAI from 'openai';

const openai = new OpenAI({
  apiKey: 'YOUR_API_KEY',
  baseURL: 'https://your-domain.com/v1',
});

async function main() {
  const completion = await openai.chat.completions.create({
    model: 'gpt-4o',
    messages: [
      { role: 'system', content: 'You are a helpful assistant.' },
      { role: 'user', content: 'Hello!' }
    ],
  });

  console.log(completion.choices[0].message.content);
}

main();
```

## API 参考

### 聊天补全

**端点**: `POST /v1/chat/completions`

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | 是 | 模型名称，如 `gpt-4o`, `claude-3-sonnet` |
| messages | array | 是 | 消息列表 |
| stream | boolean | 否 | 是否流式输出，默认 false |
| temperature | number | 否 | 温度值，0-2 之间，默认 1 |
| max_tokens | integer | 否 | 最大生成 token 数 |
| top_p | number | 否 | 核采样参数，默认 1 |
| frequency_penalty | number | 否 | 频率惩罚，默认 0 |
| presence_penalty | number | 否 | 存在惩罚，默认 0 |

**请求示例**:

```json
{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "解释一下量子计算"}
  ],
  "temperature": 0.7,
  "max_tokens": 1000
}
```

**响应示例**:

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "量子计算是一种..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 100,
    "total_tokens": 120
  }
}
```

### 流式输出

设置 `stream: true` 启用流式输出：

```bash
curl https://your-domain.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

**流式响应格式**:

```
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: [DONE]
```

### 获取模型列表

**端点**: `GET /v1/models`

**请求示例**:

```bash
curl https://your-domain.com/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**响应示例**:

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o",
      "object": "model",
      "owned_by": "openai"
    },
    {
      "id": "gpt-4-turbo",
      "object": "model",
      "owned_by": "openai"
    },
    {
      "id": "claude-3-sonnet",
      "object": "model",
      "owned_by": "anthropic"
    }
  ]
}
```

### 文本补全

**端点**: `POST /v1/completions`

**请求示例**:

```bash
curl https://your-domain.com/v1/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-3.5-turbo-instruct",
    "prompt": "写一首关于春天的诗",
    "max_tokens": 200
  }'
```

### 向量嵌入

**端点**: `POST /v1/embeddings`

**请求示例**:

```bash
curl https://your-domain.com/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "text-embedding-ada-002",
    "input": "Hello World"
  }'
```

### 图像生成

**端点**: `POST /v1/images/generations`

**请求示例**:

```bash
curl https://your-domain.com/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "dall-e-3",
    "prompt": "一只在太空的猫",
    "n": 1,
    "size": "1024x1024"
  }'
```

## 计费说明

### Token 计算

TokenHub 按照实际消耗的 Token 数量计费：

- **Prompt Token**: 输入内容的 Token 数
- **Completion Token**: 输出内容的 Token 数

**总费用 = Prompt Token × 输入单价 + Completion Token × 输出单价**

### 价格查询

在管理后台查看各模型的实时价格，或通过 API 获取：

```bash
curl https://your-domain.com/api/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### 余额查询

```bash
curl https://your-domain.com/api/user/balance \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**响应示例**:

```json
{
  "balance": 100.50,
  "currency": "CNY",
  "total_usage": 49.50
}
```

## 用量查询

### 查询当前用量

```bash
curl https://your-domain.com/api/user/usage \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### 查询历史用量

```bash
curl "https://your-domain.com/api/user/usage?start_date=2025-01-01&end_date=2025-01-31" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

## 错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 认证失败（API Key 无效） |
| 402 | 余额不足 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁（限流） |
| 500 | 服务器内部错误 |
| 503 | 服务暂时不可用 |

**错误响应格式**:

```json
{
  "error": {
    "code": "insufficient_quota",
    "message": "余额不足，请充值后重试",
    "type": "quota_error"
  }
}
```

## 限流策略

TokenHub 实施多级限流策略：

| 级别 | 限制 | 说明 |
|------|------|------|
| IP 限流 | 10 请求/秒 | 单 IP 地址 |
| Key 限流 | 60 请求/分钟 | 单 API Key |
| 用户限流 | 1000 请求/小时 | 单用户 |

超过限制将返回 429 错误。

## 最佳实践

### 1. 错误处理

```python
from openai import OpenAI, APIError

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="https://your-domain.com/v1"
)

try:
    response = client.chat.completions.create(
        model="gpt-4o",
        messages=[{"role": "user", "content": "Hello!"}]
    )
    print(response.choices[0].message.content)
except APIError as e:
    if e.status_code == 402:
        print("余额不足，请充值")
    elif e.status_code == 429:
        print("请求过于频繁，请稍后重试")
    else:
        print(f"API 错误：{e}")
```

### 2. 重试机制

```python
import time
from tenacity import retry, stop_after_attempt, wait_exponential

@retry(stop=stop_after_attempt(3), wait=wait_exponential(multiplier=1, min=4, max=10))
def chat_with_retry(messages):
    return client.chat.completions.create(
        model="gpt-4o",
        messages=messages
    )
```

### 3. 流式处理

```python
stream = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello!"}],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

## SDK 支持

TokenHub 兼容所有 OpenAI 官方 SDK：

- [Python SDK](https://github.com/openai/openai-python)
- [Node.js SDK](https://github.com/openai/openai-node)
- [Go SDK](https://github.com/sashabaranov/go-openai)
- [Java SDK](https://github.com/TheoKanning/openai-java)
- [其他语言](https://platform.openai.com/docs/libraries)

只需修改 `base_url` 即可使用。

## 技术支持

- 文档中心：https://docs.tokenhub.example.com
- GitHub Issues: https://github.com/your-org/tokenhub/issues
- 技术支持：support@tokenhub.example.com

---

**祝您使用愉快！**
