// 模型和公告相关的 API hooks
import { useQuery } from '@tanstack/react-query'
import { modelApi, announcementApi } from '@/lib/api'
import { queryKeys } from '../client'
import type { Model, Announcement } from '@/lib/api'

// 获取模型列表
export function useModels() {
  return useQuery({
    queryKey: queryKeys.models.list,
    queryFn: async () => {
      const res = await modelApi.list()
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    staleTime: 10 * 60 * 1000, // 10分钟（模型列表变化较少）
  })
}

// 获取公告列表
export function useAnnouncements() {
  return useQuery({
    queryKey: queryKeys.announcements.list,
    queryFn: async () => {
      const res = await announcementApi.list()
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    staleTime: 5 * 60 * 1000, // 5分钟
  })
}
