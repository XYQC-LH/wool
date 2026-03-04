// lib/api/modules/image.ts
// 图片生成相关 API

import { gatewayApi } from '../gateway-client'
import { 
  ImageGenerationRequest, 
  ImageGenerationResponse, 
  GenerationTaskResponse 
} from '../types'

export const imageApi = {
  // 生成图片
  generate: (data: ImageGenerationRequest) =>
    gatewayApi.post<unknown, ImageGenerationResponse>('/v1/images/generations', data),

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
