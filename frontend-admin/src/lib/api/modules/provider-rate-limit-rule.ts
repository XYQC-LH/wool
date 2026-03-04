// lib/api/modules/provider-rate-limit-rule.ts
// 多模态限流规则 API

import { api } from '../client'
import { 
  ApiResponse, 
  PaginatedResponse, 
  ProviderRateLimitRule, 
  CreateProviderRateLimitRuleRequest, 
  UpdateProviderRateLimitRuleRequest 
} from '../types'

export const providerRateLimitRuleApi = {
  list: (params?: { provider_id?: number; instance_id?: number; scope?: string; operation?: string; page?: number; page_size?: number }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ProviderRateLimitRule>>>('/api/admin/provider-rate-limit-rules', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<ProviderRateLimitRule>>(`/api/admin/provider-rate-limit-rules/${id}`),

  create: (data: CreateProviderRateLimitRuleRequest) =>
    api.post<unknown, ApiResponse<ProviderRateLimitRule>>('/api/admin/provider-rate-limit-rules', data),

  update: (id: number, data: UpdateProviderRateLimitRuleRequest) =>
    api.put<unknown, ApiResponse<ProviderRateLimitRule>>(`/api/admin/provider-rate-limit-rules/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/provider-rate-limit-rules/${id}`),
}
