// lib/api/index.ts
// API 层统一导出

// 导出客户端配置
export { 
  api, 
  API_BASE_URL 
} from './client'

// 导出 Gateway 客户端配置
export { 
  gatewayApi, 
  getGatewayApiKey, 
  setGatewayApiKey,
  getGatewayPublicBaseUrl,
  setGatewayPublicBaseUrl,
  getSwaggerUrl,
} from './gateway-client'

// 导出类型定义
export type {
  ApiResponse,
  PaginatedResponse,
  User,
  Token,
  TokenUsageStats,
  Log,
  UsageStats,
  DailyStat,
  ModelStat,
  Order,
  BillingOverview,
  ConsumptionDetail,
  Model,
  Announcement,
  CreateOrderResponse,
  ImageGenerationRequest,
  ImageGenerationResponse,
  ImageData,
  VideoGenerationRequest,
  VideoGenerationResponse,
  VideoData,
  ChatCompletionMessage,
  ChatCompletionRequest,
  ChatCompletionChoice,
  ChatCompletionUsage,
  ChatCompletionResponse,
  GenerationTaskResponse,
} from './types'

// 导出 API 模块
export { userApi } from './modules/user'
export { tokenApi } from './modules/token'
export { logApi } from './modules/log'
export { orderApi } from './modules/order'
export { modelApi } from './modules/model'
export { imageApi } from './modules/image'
export { videoApi } from './modules/video'
export { chatApi } from './modules/chat'
export { announcementApi } from './modules/announcement'
export { publicApi } from './modules/public'

// 默认导出 api 实例（保持向后兼容）
export { api as default } from './client'
