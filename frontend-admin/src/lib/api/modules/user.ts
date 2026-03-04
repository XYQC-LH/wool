// lib/api/modules/user.ts
// 用户管理 API

import { api } from '../client'
import { ApiResponse, PaginatedResponse, User, UserStats } from '../types'

export const userApi = {
  list: (params: { page?: number; page_size?: number; keyword?: string; status?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<User>>>('/api/admin/users', { params }),

  get: (id: string) =>
    api.get<unknown, ApiResponse<User>>(`/api/admin/users/${id}`),

  stats: (id: string, params?: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<UserStats>>(`/api/admin/users/${id}/stats`, { params }),

  update: (id: string, data: Partial<User>) =>
    api.put<unknown, ApiResponse>(`/api/admin/users/${id}`, data),

  updateBalance: (id: string, amount: number, reason: string) =>
    api.put<unknown, ApiResponse>(`/api/admin/users/${id}/balance`, { amount, reason }),

  updateStatus: (id: string, status: string) =>
    api.put<unknown, ApiResponse>(`/api/admin/users/${id}/status`, { status }),
}
