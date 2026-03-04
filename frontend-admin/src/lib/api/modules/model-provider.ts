// lib/api/modules/model-provider.ts
// 模型源头（ProviderGroup）API

import { api } from '../client'
import { ApiResponse, PaginatedResponse, ModelProviderResponse } from '../types'

export const modelProviderApi = {
  list: (params?: {
    page?: number
    page_size?: number
    operation?: string
    model_id?: string
    channel_id?: number
    status?: string
    circuit_state?: string
  }) => api.get<unknown, ApiResponse<PaginatedResponse<ModelProviderResponse>>>('/api/admin/providers', { params }),
}
