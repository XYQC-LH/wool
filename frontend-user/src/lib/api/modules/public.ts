// lib/api/modules/public.ts
// 公开 API（不需要认证）

import { api } from '../client'
import { ApiResponse, Model, Announcement } from '../types'

export const publicApi = {
  // 获取模型列表
  getModels: () =>
    api.get<unknown, ApiResponse<Model[]>>('/api/public/models'),
  
  // 获取公告列表
  getAnnouncements: () =>
    api.get<unknown, ApiResponse<Announcement[]>>('/api/public/announcements'),
}
