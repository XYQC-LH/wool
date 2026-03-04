// lib/api/modules/auth.ts
// 管理员认证 API

import { api } from '../client'
import { ApiResponse, Admin } from '../types'

export const authApi = {
  login: (data: { username: string; password: string }) =>
    api.post<unknown, ApiResponse<{ token: string; user: Admin }>>('/api/admin/login', data),
}
