// lib/api/modules/token.ts
// Token 相关 API

import { api } from '../client'
import { ApiResponse, PaginatedResponse, Token } from '../types'

export const tokenApi = {
  // 获取 Token 列表
  list: () =>
    api.get<unknown, ApiResponse<PaginatedResponse<Token>>>('/api/user/tokens'),

  // 创建 Token
  create: (data: { 
    name: string
    quota?: number
    expires_at?: string | Date
    allowed_ips?: string[]
    allowed_models?: string[]
    rate_limit?: number 
  }) =>
    api.post<unknown, ApiResponse<Token>>('/api/user/tokens', data),

  // 删除 Token
  delete: (id: string) =>
    api.delete<unknown, ApiResponse>(`/api/user/tokens/${id}`),

  // 更新 Token 状态
  updateStatus: (id: string, status: string) =>
    api.put<unknown, ApiResponse>(`/api/user/tokens/${id}/status`, { status }),

  // 更新 Token 高级设置
  update: (id: string, data: {
    quota?: number
    expires_at?: string | Date
    allowed_ips?: string[]
    allowed_models?: string[]
    rate_limit?: number
  }) =>
    api.put<unknown, ApiResponse<Token>>(`/api/user/tokens/${id}`, data),
}
