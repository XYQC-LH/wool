// lib/api/modules/announcement.ts
// 公告相关 API

import { api } from '../client'
import { ApiResponse, Announcement } from '../types'

export const announcementApi = {
  // 获取公告列表
  list: () =>
    api.get<unknown, ApiResponse<Announcement[]>>('/api/public/announcements'),
}
