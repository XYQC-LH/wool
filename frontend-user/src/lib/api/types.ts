// lib/api/types.ts
// API 类型定义

// 通用 API 响应类型
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

// 分页响应类型
export interface PaginatedResponse<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// 用户类型
export interface User {
  id: string
  username: string
  email: string
  avatar_url?: string
  balance: number
  role: string
  status: string
  email_notifications?: boolean
  usage_alerts?: boolean
  billing_alerts?: boolean
  created_at: string
}

// Token 使用统计
export interface TokenUsageStats {
  request_count: number
  total_tokens: number
  total_cost: number
}

// Token 类型
export interface Token {
  id: string
  key: string
  name: string
  status: string
  remain_quota?: number
  unlimited_quota: boolean
  allowed_models?: string[]
  allowed_ips?: string[]
  rate_limit?: number
  expires_at?: string
  last_used_at?: string
  created_at: string
  usage?: TokenUsageStats
}

// 日志类型
export interface Log {
  id: string
  token_key?: string
  model: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  total_cost: number | string
  duration: number
  duration_ms?: number
  status_code?: number
  status: string
  is_stream?: boolean
  error_message?: string
  created_at: string
}

// 使用统计
export interface UsageStats {
  total_requests: number
  total_tokens: number
  total_cost: number
  avg_latency_ms?: number
  daily_stats: DailyStat[]
  model_stats: ModelStat[]
}

// 每日统计
export interface DailyStat {
  date: string
  requests: number
  tokens: number
  cost: number
}

// 模型统计
export interface ModelStat {
  model: string
  requests: number
  tokens: number
  cost: number
}

// 订单类型
export interface Order {
  id: string
  user_id: string
  order_no: string
  amount: number
  currency?: string
  payment_method: string
  status: string
  paid_at?: string
  created_at: string
}

// 账单概览
export interface BillingOverview {
  balance: number
  today_cost: number
  month_cost: number
  total_recharge: number
  today_requests: number
  month_requests: number
}

// 消费明细
export interface ConsumptionDetail {
  id: string
  model: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cost: number
  status: string
  created_at: string
}

// 模型类型
export interface Model {
  id: string
  provider: string
  name: string
  display_name: string
  input_price: number | string
  output_price: number | string
  price_unit: number
  max_tokens: number
  max_context: number
  context_length: number
  max_output_tokens: number
  status: string
  enabled: boolean
  description?: string
  created_at: string
}

// 公告类型
export interface Announcement {
  id: number
  title: string
  content: string
  type: string
  status: string
  created_at: string
}

// 创建订单响应
export interface CreateOrderResponse {
  order_id: string
  order_no: string
  amount: number
  currency: string
  payment_url: string
  expires_at: string
  created_at: string
}

// 图片生成相关类型
export interface ImageGenerationRequest {
  model: string
  prompt: string
  n?: number
  size?: string
  aspect_ratio?: string
  resolution?: string
  response_format?: string
  urls?: string[]
  image?: string
  seed?: number
  watermark?: boolean
}

export interface ImageGenerationResponse {
  created: number
  data: ImageData[]
  task_id?: string
}

export interface ImageData {
  url?: string
  b64_json?: string
  revised_prompt?: string
}

// 视频生成相关类型
export interface VideoGenerationRequest {
  model: string
  prompt: string
  aspect_ratio?: string
  duration?: number
  size?: string
  image_url?: string
}

export interface VideoGenerationResponse {
  id: string
  status: string
  progress: number
  created_at: number
  data?: VideoData
  error?: string
}

export interface VideoData {
  url?: string
  duration?: number
}

// 聊天完成相关类型
export interface ChatCompletionMessage {
  role: 'system' | 'user' | 'assistant' | string
  content: string
}

export interface ChatCompletionRequest {
  model: string
  messages: ChatCompletionMessage[]
  stream?: boolean
  temperature?: number
  max_tokens?: number
}

export interface ChatCompletionChoice {
  index: number
  message: ChatCompletionMessage
  finish_reason?: string | null
}

export interface ChatCompletionUsage {
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
}

export interface ChatCompletionResponse {
  id: string
  object: string
  created: number
  model: string
  choices: ChatCompletionChoice[]
  usage?: ChatCompletionUsage
}

// 生成任务响应
export interface GenerationTaskResponse {
  id: string
  type: string
  model: string
  status: string
  progress: number
  result_url?: string
  error_message?: string
  cost: number
  duration: number
  created_at: string
  completed_at?: string
}
