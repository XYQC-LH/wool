// 用户相关的 API hooks
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { userApi } from '@/lib/api'
import { queryKeys, invalidateQueries } from '../client'
import type { User, BillingOverview } from '@/lib/api'

// 获取用户信息
export function useUserProfile() {
  return useQuery({
    queryKey: queryKeys.user.profile,
    queryFn: async () => {
      const res = await userApi.getProfile()
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    staleTime: 5 * 60 * 1000, // 5分钟
  })
}

// 获取账单概览
export function useBillingOverview() {
  return useQuery({
    queryKey: queryKeys.user.billing,
    queryFn: async () => {
      const res = await userApi.getBillingOverview()
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    staleTime: 2 * 60 * 1000, // 2分钟
  })
}

// 更新用户信息
export function useUpdateProfile() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { username?: string; email?: string; avatar_url?: string }) => {
      const res = await userApi.updateProfile(data)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    onSuccess: () => {
      // 更新成功后刷新用户信息
      invalidateQueries(queryKeys.user.profile)
    },
  })
}

// 修改密码
export function useChangePassword() {
  return useMutation({
    mutationFn: async (data: { old_password: string; new_password: string }) => {
      const res = await userApi.changePassword(data)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
  })
}

// 更新通知设置
export function useUpdateNotifications() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { email_notifications?: boolean; usage_alerts?: boolean; billing_alerts?: boolean }) => {
      const res = await userApi.updateNotifications(data)
      if (res.code !== 0) throw new Error(res.message)
      return res.data
    },
    onSuccess: () => {
      invalidateQueries(queryKeys.user.profile)
    },
  })
}
