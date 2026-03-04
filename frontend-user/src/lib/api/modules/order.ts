// lib/api/modules/order.ts
// 订单相关 API

import { api } from '../client'
import { 
  ApiResponse, 
  PaginatedResponse, 
  Order, 
  CreateOrderResponse, 
  ConsumptionDetail 
} from '../types'

export const orderApi = {
  // 获取订单列表
  list: (params: { page?: number; page_size?: number; status?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Order>>>('/api/user/orders', { params }),

  // 创建订单
  create: (data: { amount: number; payment_method: string; currency?: string }) =>
    api.post<unknown, ApiResponse<CreateOrderResponse>>('/api/user/orders', data),

  // 获取订单详情
  get: (id: string) =>
    api.get<unknown, ApiResponse<Order>>(`/api/user/orders/${id}`),

  // 根据订单号获取订单详情
  getByOrderNo: (orderNo: string) =>
    api.get<unknown, ApiResponse<Order>>(`/api/user/orders/by-no/${encodeURIComponent(orderNo)}`),

  // 取消订单
  cancel: (id: string) =>
    api.post<unknown, ApiResponse<void>>(`/api/user/orders/${id}/cancel`),

  // 订单支付（模拟/手动确认）
  payByOrderNo: (orderNo: string) =>
    api.post<unknown, ApiResponse<{ message?: string }>>(`/api/user/orders/by-no/${encodeURIComponent(orderNo)}/pay`),

  // 获取消费明细
  getConsumptionDetails: (params: { 
    page?: number
    page_size?: number
    start_date?: string
    end_date?: string 
  }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ConsumptionDetail>>>('/api/user/billing/consumption', { params }),
}
