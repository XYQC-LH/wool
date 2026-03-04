// 订单相关的 API hooks
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { orderApi } from '@/lib/api'
import { queryKeys, invalidateQueries } from '../client'
import type { Order, PaginatedResponse, ConsumptionDetail } from '@/lib/api'

interface OrderListParams {
  page?: number
  page_size?: number
  status?: string
}

interface ConsumptionParams {
  page?: number
  page_size?: number
  start_date?: string
  end_date?: string
}

// 获取订单列表
export function useOrders(params?: OrderListParams) {
  return useQuery({
    queryKey: queryKeys.orders.list(params),
    queryFn: async () => {
      const res = await orderApi.list(params || {})
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    staleTime: 1 * 60 * 1000, // 1分钟
  })
}

// 获取订单详情
export function useOrder(id: string) {
  return useQuery({
    queryKey: queryKeys.orders.detail(id),
    queryFn: async () => {
      const res = await orderApi.get(id)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    enabled: !!id, // 只有 id 存在时才查询
    staleTime: 2 * 60 * 1000, // 2分钟
  })
}

// 创建订单
export function useCreateOrder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { amount: number; payment_method: string; currency?: string }) => {
      const res = await orderApi.create(data)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    onSuccess: () => {
      invalidateQueries(queryKeys.orders.list())
      invalidateQueries(queryKeys.user.billing)
    },
  })
}

// 取消订单
export function useCancelOrder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const res = await orderApi.cancel(id)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    onSuccess: () => {
      invalidateQueries(queryKeys.orders.list())
    },
  })
}

// 获取消费明细
export function useConsumptionDetails(params?: ConsumptionParams) {
  return useQuery({
    queryKey: ['consumption', 'list', params],
    queryFn: async () => {
      const res = await orderApi.getConsumptionDetails(params || {})
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    staleTime: 1 * 60 * 1000, // 1分钟
  })
}
