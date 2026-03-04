// lib/api/modules/system.ts
// 系统监控 API

import { api } from '../client'
import { ApiResponse, SystemStats } from '../types'

export const systemApi = {
  // 获取系统监控数据
  getMonitor: () =>
    api.get<unknown, ApiResponse<{
      cpu_percent: number
      memory_percent: number
      redis_connections: number
      db_connections: number
    }>>('/api/admin/dashboard/system'),

  // 获取异常告警
  getAlerts: () =>
    api.get<unknown, ApiResponse<Array<{
      id: string
      message: string
      level: 'info' | 'warning' | 'error' | 'critical'
      created_at: string
    }>>>('/api/admin/dashboard/alerts'),

  // 保存系统设置
  saveSettings: (section: string, data: Record<string, unknown>) =>
    api.put<unknown, ApiResponse>(`/api/admin/settings/${section}`, data),

  // 获取系统设置
  getSettings: (section: string) =>
    api.get<unknown, ApiResponse<Record<string, unknown>>>(`/api/admin/settings/${section}`),
}
