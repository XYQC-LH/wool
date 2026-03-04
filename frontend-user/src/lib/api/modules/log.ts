// lib/api/modules/log.ts
// 日志相关 API

import { api } from '../client'
import { ApiResponse, PaginatedResponse, Log, UsageStats } from '../types'

export const logApi = {
  // 获取日志列表
  list: (params: { 
    page?: number
    page_size?: number
    model?: string
    status?: string
    start_date?: string
    end_date?: string 
  }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Log>>>('/api/user/logs', { params }),

  // 获取使用统计
  stats: (params: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<UsageStats>>('/api/user/logs/stats', { params }),
}
