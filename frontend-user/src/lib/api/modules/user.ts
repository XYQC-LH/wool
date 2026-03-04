// lib/api/modules/user.ts
// 用户相关 API

import { api } from '../client'
import { ApiResponse, User, BillingOverview } from '../types'

export const userApi = {
  // 注册
  register: (data: { username: string; email: string; password: string }) =>
    api.post<unknown, ApiResponse>('/api/user/register', data),

  // 登录
  login: (data: { username: string; password: string }) =>
    api.post<unknown, ApiResponse<{ token: string; user: User }>>('/api/user/login', data),

  // 获取用户信息
  getProfile: () =>
    api.get<unknown, ApiResponse<User>>('/api/user/profile'),

  // 更新用户信息
  updateProfile: (data: { username?: string; email?: string; avatar_url?: string }) =>
    api.put<unknown, ApiResponse>('/api/user/profile', data),

  // 修改密码
  changePassword: (data: { old_password: string; new_password: string }) =>
    api.put<unknown, ApiResponse>('/api/user/password', data),

  // 获取账单概览
  getBillingOverview: () =>
    api.get<unknown, ApiResponse<BillingOverview>>('/api/user/billing/overview'),

  // 更新通知设置
  updateNotifications: (data: { 
    email_notifications?: boolean
    usage_alerts?: boolean
    billing_alerts?: boolean 
  }) =>
    api.put<unknown, ApiResponse<User>>('/api/user/notifications', data),
}
