// QueryClient 配置
import { QueryClient } from '@tanstack/react-query'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // 数据缓存时间：5分钟
      staleTime: 5 * 60 * 1000,
      // 数据保持时间：10分钟
      gcTime: 10 * 60 * 1000,
      // 失败时重试3次
      retry: 3,
      // 重试延迟：指数退避
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
      // 窗口重新聚焦时刷新数据
      refetchOnWindowFocus: true,
      // 网络重连时刷新数据
      refetchOnReconnect: true,
    },
    mutations: {
      // 失败时重试1次
      retry: 1,
    },
  },
})

// 常用查询键（query keys）
export const queryKeys = {
  // 用户相关
  user: {
    profile: ['user', 'profile'] as const,
    billing: ['user', 'billing'] as const,
  },
  // Token 相关
  tokens: {
    list: ['tokens', 'list'] as const,
    detail: (id: string) => ['tokens', 'detail', id] as const,
  },
  // 日志相关
  logs: {
    list: (params?: object) => ['logs', 'list', params] as const,
    stats: ['logs', 'stats'] as const,
  },
  // 订单相关
  orders: {
    list: (params?: object) => ['orders', 'list', params] as const,
    detail: (id: string) => ['orders', 'detail', id] as const,
  },
  // 模型相关
  models: {
    list: ['models', 'list'] as const,
  },
  // 公告相关
  announcements: {
    list: ['announcements', 'list'] as const,
  },
  // 生成任务相关
  generations: {
    list: (params?: object) => ['generations', 'list', params] as const,
    detail: (id: string) => ['generations', 'detail', id] as const,
  },
} as const

// 预加载数据工具函数
export function prefetchQuery<T>(
  queryKey: readonly unknown[],
  queryFn: () => Promise<T>
) {
  return queryClient.prefetchQuery({
    queryKey,
    queryFn,
    staleTime: 5 * 60 * 1000,
  })
}

// 使缓存失效工具函数
export function invalidateQueries(queryKey: readonly unknown[]) {
  return queryClient.invalidateQueries({ queryKey })
}
