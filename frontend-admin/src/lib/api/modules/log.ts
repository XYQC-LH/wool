// lib/api/modules/log.ts
// 日志管理 API

import { api } from '../client'
import { ApiResponse, PaginatedResponse, Log, LogStats } from '../types'

export const logApi = {
  list: (params: {
    page?: number
    page_size?: number
    user_id?: string
    model?: string
    channel_id?: number
    status?: string
    start_date?: string
    end_date?: string
  }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Log>>>('/api/admin/logs', { params }),

  stats: (params?: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<LogStats>>('/api/admin/logs/stats', { params }),
}
