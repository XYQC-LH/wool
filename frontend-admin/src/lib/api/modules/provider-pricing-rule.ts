// lib/api/modules/provider-pricing-rule.ts
// 多模态计费规则 API

import { api } from '../client'
import { 
  ApiResponse, 
  PaginatedResponse, 
  ProviderPricingRule, 
  CreateProviderPricingRuleRequest, 
  UpdateProviderPricingRuleRequest 
} from '../types'

export const providerPricingRuleApi = {
  list: (params?: { provider_id?: number; operation?: string; page?: number; page_size?: number }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ProviderPricingRule>>>('/api/admin/provider-pricing-rules', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<ProviderPricingRule>>(`/api/admin/provider-pricing-rules/${id}`),

  create: (data: CreateProviderPricingRuleRequest) =>
    api.post<unknown, ApiResponse<ProviderPricingRule>>('/api/admin/provider-pricing-rules', data),

  update: (id: number, data: UpdateProviderPricingRuleRequest) =>
    api.put<unknown, ApiResponse<ProviderPricingRule>>(`/api/admin/provider-pricing-rules/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/provider-pricing-rules/${id}`),
}
