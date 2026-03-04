// Token 相关的 API hooks
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { tokenApi } from '@/lib/api'
import { queryKeys, invalidateQueries } from '../client'
import type { Token, PaginatedResponse } from '@/lib/api'

// 获取 Token 列表
export function useTokens() {
  return useQuery({
    queryKey: queryKeys.tokens.list,
    queryFn: async () => {
      const res = await tokenApi.list()
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    staleTime: 1 * 60 * 1000, // 1分钟
  })
}

// 创建 Token
export function useCreateToken() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { 
      name: string
      quota?: number
      expires_at?: string | Date
      allowed_ips?: string[]
      allowed_models?: string[]
      rate_limit?: number 
    }) => {
      const res = await tokenApi.create(data)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    onSuccess: () => {
      invalidateQueries(queryKeys.tokens.list)
    },
  })
}

// 删除 Token
export function useDeleteToken() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const res = await tokenApi.delete(id)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    onSuccess: () => {
      invalidateQueries(queryKeys.tokens.list)
    },
  })
}

// 更新 Token 状态
export function useUpdateTokenStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ id, status }: { id: string; status: string }) => {
      const res = await tokenApi.updateStatus(id, status)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    onSuccess: () => {
      invalidateQueries(queryKeys.tokens.list)
    },
  })
}

// 更新 Token 配置
export function useUpdateToken() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ 
      id, 
      data 
    }: { 
      id: string
      data: {
        quota?: number
        expires_at?: string | Date
        allowed_ips?: string[]
        allowed_models?: string[]
        rate_limit?: number
      }
    }) => {
      const res = await tokenApi.update(id, data)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    onSuccess: () => {
      invalidateQueries(queryKeys.tokens.list)
    },
  })
}
