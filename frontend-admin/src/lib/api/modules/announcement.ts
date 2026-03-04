// lib/api/modules/announcement.ts
// 公告管理 API

import { api } from '../client'
import { 
  ApiResponse, 
  PaginatedResponse, 
  Announcement, 
  CreateAnnouncementRequest, 
  UpdateAnnouncementRequest 
} from '../types'

export const announcementApi = {
  list: (params?: { page?: number; page_size?: number; status?: string; type?: string; keyword?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Announcement>>>('/api/admin/announcements', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<Announcement>>(`/api/admin/announcements/${id}`),

  create: (data: CreateAnnouncementRequest) =>
    api.post<unknown, ApiResponse<Announcement>>('/api/admin/announcements', data),

  update: (id: number, data: UpdateAnnouncementRequest) =>
    api.put<unknown, ApiResponse<Announcement>>(`/api/admin/announcements/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/announcements/${id}`),

  publish: (id: number) =>
    api.post<unknown, ApiResponse>(`/api/admin/announcements/${id}/publish`),

  archive: (id: number) =>
    api.post<unknown, ApiResponse>(`/api/admin/announcements/${id}/archive`),
}
