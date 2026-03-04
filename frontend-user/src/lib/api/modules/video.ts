// lib/api/modules/video.ts
// 视频生成相关 API

import { gatewayApi } from '../gateway-client'
import { 
  VideoGenerationRequest, 
  VideoGenerationResponse, 
  GenerationTaskResponse 
} from '../types'

export const videoApi = {
  // 生成视频
  generate: (data: VideoGenerationRequest) =>
    gatewayApi.post<unknown, VideoGenerationResponse>('/v1/videos/generations', data),

  // 获取任务状态
  getTaskStatus: (taskId: string) =>
    gatewayApi.get<unknown, GenerationTaskResponse>(`/v1/generations/${taskId}`),

  // 获取任务列表
  listTasks: (params: { type?: string; page?: number; page_size?: number }) =>
    gatewayApi.get<unknown, { 
      data: GenerationTaskResponse[]
      total: number
      page: number
      page_size: number 
    }>('/v1/generations', { params }),
}
