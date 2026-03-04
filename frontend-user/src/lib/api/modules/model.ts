// lib/api/modules/model.ts
// 模型相关 API

import { api } from '../client'
import { ApiResponse, Model } from '../types'

export const modelApi = {
  // 获取可用模型列表
  list: () =>
    api.get<unknown, ApiResponse<Model[]>>('/api/public/models'),
}
