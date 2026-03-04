// lib/api/modules/resource-account.ts
// 资源账户管理 API

import { api } from '../client'
import { 
  ApiResponse, 
  PaginatedResponse, 
  ResourceAccount, 
  ResourceAccountStats,
  CreateResourceAccountRequest,
  UpdateResourceAccountRequest 
} from '../types'

export const resourceAccountApi = {
  list: (params?: { page?: number; page_size?: number; status?: string; channel_id?: number; keyword?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ResourceAccount>>>('/api/admin/resource-accounts', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<ResourceAccount>>(`/api/admin/resource-accounts/${id}`),

  create: (data: CreateResourceAccountRequest) =>
    api.post<unknown, ApiResponse<ResourceAccount>>('/api/admin/resource-accounts', data),

  update: (id: number, data: UpdateResourceAccountRequest) =>
    api.put<unknown, ApiResponse<ResourceAccount>>(`/api/admin/resource-accounts/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/resource-accounts/${id}`),

  refresh: (id: number) =>
    api.post<unknown, ApiResponse>(`/api/admin/resource-accounts/${id}/refresh`),

  stats: () =>
    api.get<unknown, ApiResponse<ResourceAccountStats>>('/api/admin/resource-accounts/stats'),
}
