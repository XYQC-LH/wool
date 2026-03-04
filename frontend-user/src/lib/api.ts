import axios, { AxiosInstance, AxiosError } from 'axios'
import Cookies from 'js-cookie'

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
const GATEWAY_API_KEY_STORAGE_KEY = 'nexus_gateway_api_key'
const GATEWAY_PUBLIC_BASE_URL_STORAGE_KEY = 'nexus_gateway_public_base_url'
const AUTH_STORAGE_KEY = 'auth-storage'

export function getGatewayPublicBaseUrl(): string {
  const env = process.env.NEXT_PUBLIC_GATEWAY_PUBLIC_URL
  const fallback = `${API_BASE_URL}/v1`

  const normalize = (value: string) => value.trim().replace(/\/+$/, '')

  if (typeof window === 'undefined') return normalize(env || fallback)

  const stored = window.localStorage.getItem(GATEWAY_PUBLIC_BASE_URL_STORAGE_KEY)
  return normalize(stored || env || fallback)
}

export function setGatewayPublicBaseUrl(baseUrl: string | null) {
  if (typeof window === 'undefined') return
  if (!baseUrl) {
    window.localStorage.removeItem(GATEWAY_PUBLIC_BASE_URL_STORAGE_KEY)
    return
  }
  const normalized = baseUrl.trim().replace(/\/+$/, '')
  window.localStorage.setItem(GATEWAY_PUBLIC_BASE_URL_STORAGE_KEY, normalized)
}

export function getSwaggerUrl(): string {
  const gatewayBaseUrl = getGatewayPublicBaseUrl()
  const base = gatewayBaseUrl.replace(/\/v1\/?$/, '')
  return `${base}/swagger/index.html`
}

export function getGatewayApiKey(): string | null {
  if (typeof window === 'undefined') return null
  return window.localStorage.getItem(GATEWAY_API_KEY_STORAGE_KEY)
}

export function setGatewayApiKey(apiKey: string | null) {
  if (typeof window === 'undefined') return
  if (!apiKey) {
    window.localStorage.removeItem(GATEWAY_API_KEY_STORAGE_KEY)
    return
  }
  window.localStorage.setItem(GATEWAY_API_KEY_STORAGE_KEY, apiKey)
}

// 创建 axios 实例
const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = Cookies.get('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      // Token 过期，清除登录状态
      Cookies.remove('token')
      try {
        if (typeof window !== 'undefined') {
          window.localStorage.removeItem(AUTH_STORAGE_KEY)
          window.dispatchEvent(new Event('auth:logout'))
        }
      } catch {
      }
      // 不再自动跳转，让页面自行处理未登录状态
    }
    return Promise.reject(error)
  }
)

// Gateway API 专用客户端（使用 API Key，而不是登录态 JWT）
const gatewayApi: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

gatewayApi.interceptors.request.use(
  (config) => {
    const apiKey = getGatewayApiKey()
    if (apiKey) {
      config.headers = config.headers ?? {}
      ;(config.headers as Record<string, string>).Authorization = `Bearer ${apiKey}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

gatewayApi.interceptors.response.use(
  (response) => response.data,
  (error: AxiosError) => Promise.reject(error)
)

// API 响应类型
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PaginatedResponse<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// 用户相关 API
export const userApi = {
  // 注册
  register: (data: { username: string; email: string; password: string }) =>
    api.post<unknown, ApiResponse>('/api/user/register', data),

  // 登录
  login: (data: { username: string; password: string }) =>
    api.post<unknown, ApiResponse<{ token: string; user: User }>>('/api/user/login', data),

  // 获取用户信息
  getProfile: () =>
    api.get<unknown, ApiResponse<User>>('/api/user/profile'),

  // 更新用户信息
  updateProfile: (data: { username?: string; email?: string; avatar_url?: string }) =>
    api.put<unknown, ApiResponse>('/api/user/profile', data),

  // 修改密码
  changePassword: (data: { old_password: string; new_password: string }) =>
    api.put<unknown, ApiResponse>('/api/user/password', data),

  // 获取账单概览
  getBillingOverview: () =>
    api.get<unknown, ApiResponse<BillingOverview>>('/api/user/billing/overview'),

  // 更新通知设置
  updateNotifications: (data: { email_notifications?: boolean; usage_alerts?: boolean; billing_alerts?: boolean }) =>
    api.put<unknown, ApiResponse<User>>('/api/user/notifications', data),
}

// Token 相关 API
export const tokenApi = {
  // 获取 Token 列表
  list: () =>
    api.get<unknown, ApiResponse<PaginatedResponse<Token>>>('/api/user/tokens'),

  // 创建 Token
  create: (data: { name: string; quota?: number; expires_at?: string | Date; allowed_ips?: string[]; allowed_models?: string[]; rate_limit?: number }) =>
    api.post<unknown, ApiResponse<Token>>('/api/user/tokens', data),

  // 删除 Token
  delete: (id: string) =>
    api.delete<unknown, ApiResponse>(`/api/user/tokens/${id}`),

  // 更新 Token 状态
  updateStatus: (id: string, status: string) =>
    api.put<unknown, ApiResponse>(`/api/user/tokens/${id}/status`, { status }),

  // 更新 Token 高级设置
  update: (id: string, data: {
    quota?: number
    expires_at?: string | Date
    allowed_ips?: string[]
    allowed_models?: string[]
    rate_limit?: number
  }) =>
    api.put<unknown, ApiResponse<Token>>(`/api/user/tokens/${id}`, data),
}

// 日志相关 API
export const logApi = {
  // 获取日志列表
  list: (params: { page?: number; page_size?: number; model?: string; status?: string; start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Log>>>('/api/user/logs', { params }),

  // 获取使用统计
  stats: (params: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<UsageStats>>('/api/user/logs/stats', { params }),
}

// 订单相关 API
export const orderApi = {
  // 获取订单列表
  list: (params: { page?: number; page_size?: number; status?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Order>>>('/api/user/orders', { params }),

  // 创建订单
  create: (data: { amount: number; payment_method: string; currency?: string }) =>
    api.post<unknown, ApiResponse<CreateOrderResponse>>('/api/user/orders', data),

  // 获取订单详情
  get: (id: string) =>
    api.get<unknown, ApiResponse<Order>>(`/api/user/orders/${id}`),

  // 根据订单号获取订单详情
  getByOrderNo: (orderNo: string) =>
    api.get<unknown, ApiResponse<Order>>(`/api/user/orders/by-no/${encodeURIComponent(orderNo)}`),

  // 取消订单
  cancel: (id: string) =>
    api.post<unknown, ApiResponse<void>>(`/api/user/orders/${id}/cancel`),

  // 订单支付（模拟/手动确认）
  payByOrderNo: (orderNo: string) =>
    api.post<unknown, ApiResponse<{ message?: string }>>(`/api/user/orders/by-no/${encodeURIComponent(orderNo)}/pay`),

  // 获取消费明细
  getConsumptionDetails: (params: { page?: number; page_size?: number; start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ConsumptionDetail>>>('/api/user/billing/consumption', { params }),
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

// 模型相关 API
export const modelApi = {
  // 获取可用模型列表
  list: () =>
    api.get<unknown, ApiResponse<Model[]>>('/api/public/models'),
}

// 图片生成相关 API
export const imageApi = {
  // 生成图片
  generate: (data: ImageGenerationRequest) =>
    gatewayApi.post<unknown, ImageGenerationResponse>('/v1/images/generations', data),

  // 获取任务状态
  getTaskStatus: (taskId: string) =>
    gatewayApi.get<unknown, GenerationTaskResponse>(`/v1/generations/${taskId}`),

  // 获取任务列表
  listTasks: (params: { type?: string; page?: number; page_size?: number }) =>
    gatewayApi.get<unknown, { data: GenerationTaskResponse[]; total: number; page: number; page_size: number }>('/v1/generations', { params }),
}

// 视频生成相关 API
export const videoApi = {
  // 生成视频
  generate: (data: VideoGenerationRequest) =>
    gatewayApi.post<unknown, VideoGenerationResponse>('/v1/videos/generations', data),

  // 获取任务状态
  getTaskStatus: (taskId: string) =>
    gatewayApi.get<unknown, GenerationTaskResponse>(`/v1/generations/${taskId}`),

  // 获取任务列表
  listTasks: (params: { type?: string; page?: number; page_size?: number }) =>
    gatewayApi.get<unknown, { data: GenerationTaskResponse[]; total: number; page: number; page_size: number }>('/v1/generations', { params }),
}

// 聊天相关 API（OpenAI 兼容 /v1）
export const chatApi = {
  // 聊天完成
  chatCompletions: (data: ChatCompletionRequest) =>
    gatewayApi.post<unknown, ChatCompletionResponse>('/v1/chat/completions', data),
}

// 公告相关 API
export const announcementApi = {
  // 获取公告列表
  list: () =>
    api.get<unknown, ApiResponse<Announcement[]>>('/api/public/announcements'),
}

// 公开 API（不需要认证）
export const publicApi = {
  // 获取模型列表
  getModels: () =>
    api.get<unknown, ApiResponse<Model[]>>('/api/public/models'),
  
  // 获取公告列表
  getAnnouncements: () =>
    api.get<unknown, ApiResponse<Announcement[]>>('/api/public/announcements'),
}

// 类型定义
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

export interface TokenUsageStats {
  request_count: number
  total_tokens: number
  total_cost: number
}

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

export interface UsageStats {
  total_requests: number
  total_tokens: number
  total_cost: number
  avg_latency_ms?: number
  daily_stats: DailyStat[]
  model_stats: ModelStat[]
}

export interface DailyStat {
  date: string
  requests: number
  tokens: number
  cost: number
}

export interface ModelStat {
  model: string
  requests: number
  tokens: number
  cost: number
}

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

export interface BillingOverview {
  balance: number
  today_cost: number
  month_cost: number
  total_recharge: number
  today_requests: number
  month_requests: number
}

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

export interface Announcement {
  id: number
  title: string
  content: string
  type: string
  status: string
  created_at: string
}

// 图片生成请求
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

// 图片生成响应
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

// 视频生成请求
export interface VideoGenerationRequest {
  model: string
  prompt: string
  aspect_ratio?: string
  duration?: number
  size?: string
  image_url?: string
}

// 视频生成响应
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

export default api
