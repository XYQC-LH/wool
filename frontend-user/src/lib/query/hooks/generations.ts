// 生成任务相关的 API hooks
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { imageApi, videoApi } from '@/lib/api'
import { queryKeys, invalidateQueries } from '../client'
import type { 
  ImageGenerationRequest, 
  VideoGenerationRequest, 
  ImageGenerationResponse,
  VideoGenerationResponse,
  GenerationTaskResponse 
} from '@/lib/api'

interface TaskListParams {
  type?: string
  page?: number
  page_size?: number
}

// 生成图片
export function useGenerateImage() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: ImageGenerationRequest) => {
      const res = await imageApi.generate(data)
      return res
    },
    onSuccess: () => {
      // 生成成功后刷新任务列表
      invalidateQueries(queryKeys.generations.list())
    },
  })
}

// 生成视频
export function useGenerateVideo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: VideoGenerationRequest) => {
      const res = await videoApi.generate(data)
      return res
    },
    onSuccess: () => {
      invalidateQueries(queryKeys.generations.list())
    },
  })
}

// 获取任务列表
export function useGenerationTasks(params?: TaskListParams) {
  return useQuery({
    queryKey: queryKeys.generations.list(params),
    queryFn: async () => {
      const res = await imageApi.listTasks(params || {})
      return res
    },
    staleTime: 10 * 1000, // 10秒（任务状态变化快）
    refetchInterval: 5000, // 每5秒自动刷新
  })
}

// 获取任务详情
export function useGenerationTask(id: string) {
  return useQuery({
    queryKey: queryKeys.generations.detail(id),
    queryFn: async () => {
      const res = await imageApi.getTaskStatus(id)
      return res
    },
    enabled: !!id,
    staleTime: 5 * 1000, // 5秒
    refetchInterval: (query) => {
      // 任务未完成时继续轮询
      const data = query.state.data as GenerationTaskResponse | undefined
      if (data && data.status === 'processing') {
        return 3000 // 每3秒刷新
      }
      return false // 完成后停止轮询
    },
  })
}
