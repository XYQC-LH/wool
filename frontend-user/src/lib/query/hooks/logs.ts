// 日志相关的 API hooks
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { logApi } from '@/lib/api'
import { queryKeys, invalidateQueries } from '../client'
import type { Log, UsageStats, PaginatedResponse } from '@/lib/api'

interface LogListParams {
  page?: number
  page_size?: number
  model?: string
  status?: string
  start_date?: string
  end_date?: string
}

// 获取日志列表
export function useLogs(params?: LogListParams) {
  return useQuery({
    queryKey: queryKeys.logs.list(params),
    queryFn: async () => {
      const res = await logApi.list(params || {})
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    staleTime: 30 * 1000, // 30秒
  })
}

// 获取使用统计
export function useUsageStats(params?: { start_date?: string; end_date?: string }) {
  return useQuery({
    queryKey: [...queryKeys.logs.stats, params],
    queryFn: async () => {
      const res = await logApi.stats(params || {})
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    staleTime: 2 * 60 * 1000, // 2分钟
  })
}
