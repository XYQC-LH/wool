// lib/api/modules/order.ts
// 订单管理 API

import { api } from '../client'
import { ApiResponse, PaginatedResponse, Order, OrderStats } from '../types'

export const orderApi = {
  list: (params: { 
    page?: number
    page_size?: number
    status?: string
    user_id?: string
    payment_method?: string
    start_date?: string
    end_date?: string 
  }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Order>>>('/api/admin/orders', { params }),

  get: (id: string) =>
    api.get<unknown, ApiResponse<Order>>(`/api/admin/orders/${id}`),

  updateStatus: (id: string, status: string) =>
    api.put<unknown, ApiResponse>(`/api/admin/orders/${id}/status`, { status }),

  stats: (params?: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<OrderStats>>('/api/admin/orders/stats', { params }),
}
