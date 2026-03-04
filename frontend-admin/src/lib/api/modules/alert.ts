// lib/api/modules/alert.ts
// 告警管理 API

import { api } from '../client'
import { ApiResponse, PaginatedResponse, Alert, AlertStats } from '../types'

export const alertApi = {
  list: (params?: { page?: number; page_size?: number; type?: string; severity?: string; status?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Alert>>>('/api/admin/alerts', { params }),

  get: (id: string) =>
    api.get<unknown, ApiResponse<Alert>>(`/api/admin/alerts/${id}`),

  resolve: (id: string) =>
    api.put<unknown, ApiResponse>(`/api/admin/alerts/${id}/resolve`, {}),

  stats: () =>
    api.get<unknown, ApiResponse<AlertStats>>('/api/admin/alerts/stats'),

  active: () =>
    api.get<unknown, ApiResponse<Alert[]>>('/api/admin/alerts/active'),
}
