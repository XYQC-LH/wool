// lib/api/modules/provider-instance.ts
// Provider 实例 API

import { api } from '../client'
import { ApiResponse, PaginatedResponse, ProviderInstanceResponse } from '../types'

export const providerInstanceApi = {
  list: (providerId: number, params?: { page?: number; page_size?: number; status?: string; instance_type?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ProviderInstanceResponse>>>(`/api/admin/providers/${providerId}/instances`, { params }),
}
