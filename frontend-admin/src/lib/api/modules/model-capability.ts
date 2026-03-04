// lib/api/modules/model-capability.ts
// 模型能力（operation 开关）API

import { api } from '../client'
import { 
  ApiResponse, 
  PaginatedResponse, 
  ModelCapability, 
  CreateModelCapabilityRequest, 
  UpdateModelCapabilityRequest 
} from '../types'

export const modelCapabilityApi = {
  list: (params?: { model_id?: string; operation?: string; page?: number; page_size?: number }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ModelCapability>>>('/api/admin/model-capabilities', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<ModelCapability>>(`/api/admin/model-capabilities/${id}`),

  create: (data: CreateModelCapabilityRequest) =>
    api.post<unknown, ApiResponse<ModelCapability>>('/api/admin/model-capabilities', data),

  update: (id: number, data: UpdateModelCapabilityRequest) =>
    api.put<unknown, ApiResponse<ModelCapability>>(`/api/admin/model-capabilities/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/model-capabilities/${id}`),
}
