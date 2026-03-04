// lib/api/modules/model.ts
// 模型管理 API

import { api } from '../client'
import { 
  ApiResponse, 
  PaginatedResponse, 
  Model, 
  CreateModelRequest, 
  UpdateModelRequest 
} from '../types'

export const modelApi = {
  list: (params?: { page?: number; page_size?: number; enabled?: boolean; type?: string; keyword?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Model>>>('/api/admin/models', { params }),

  get: (id: string) =>
    api.get<unknown, ApiResponse<Model>>(`/api/admin/models/${id}`),

  create: (data: CreateModelRequest) =>
    api.post<unknown, ApiResponse<Model>>('/api/admin/models', data),

  update: (id: string, data: UpdateModelRequest) =>
    api.put<unknown, ApiResponse<Model>>(`/api/admin/models/${id}`, data),

  delete: (id: string) =>
    api.delete<unknown, ApiResponse>(`/api/admin/models/${id}`),

  updateStatus: (id: string, enabled: boolean) =>
    api.put<unknown, ApiResponse>(`/api/admin/models/${id}/status`, { enabled }),
}
