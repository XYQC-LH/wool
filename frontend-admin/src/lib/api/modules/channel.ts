// lib/api/modules/channel.ts
// 渠道管理 API

import { api } from '../client'
import { 
  ApiResponse, 
  PaginatedResponse, 
  Channel, 
  CreateChannelRequest, 
  UpdateChannelRequest, 
  ChannelTestResult 
} from '../types'

export const channelApi = {
  list: (params?: { page?: number; page_size?: number; status?: string; type?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Channel>>>('/api/admin/channels', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<Channel>>(`/api/admin/channels/${id}`),

  create: (data: CreateChannelRequest) =>
    api.post<unknown, ApiResponse<Channel>>('/api/admin/channels', data),

  update: (id: number, data: UpdateChannelRequest) =>
    api.put<unknown, ApiResponse<Channel>>(`/api/admin/channels/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/channels/${id}`),

  test: (id: number) =>
    api.post<unknown, ApiResponse<ChannelTestResult>>(`/api/admin/channels/${id}/test`),

  updateStatus: (id: number, status: string) =>
    api.put<unknown, ApiResponse<Channel>>(`/api/admin/channels/${id}/status`, { status }),
}
