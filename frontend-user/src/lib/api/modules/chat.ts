// lib/api/modules/chat.ts
// 聊天相关 API（OpenAI 兼容 /v1）

import { gatewayApi } from '../gateway-client'
import { ChatCompletionRequest, ChatCompletionResponse } from '../types'

export const chatApi = {
  // 聊天完成
  chatCompletions: (data: ChatCompletionRequest) =>
    gatewayApi.post<unknown, ChatCompletionResponse>('/v1/chat/completions', data),
}
