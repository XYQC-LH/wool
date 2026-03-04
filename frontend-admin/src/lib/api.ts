import axios, { AxiosInstance, AxiosError } from 'axios'
import Cookies from 'js-cookie'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

function extractErrorMessageFromResponseData(data: unknown): string | undefined {
  if (!data) return undefined

  if (typeof data === 'string') {
    const trimmed = data.trim()
    return trimmed ? trimmed : undefined
  }

  if (typeof data !== 'object') return undefined

  const record = data as Record<string, unknown>
  const message = record.message
  if (typeof message === 'string' && message.trim()) return message

  const error = record.error
  if (typeof error === 'string' && error.trim()) return error
  if (error && typeof error === 'object') {
    const nestedMessage = (error as Record<string, unknown>).message
    if (typeof nestedMessage === 'string' && nestedMessage.trim()) return nestedMessage
  }

  return undefined
}

export function getErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    return (
      extractErrorMessageFromResponseData(error.response?.data) ||
      (typeof error.message === 'string' && error.message.trim() ? error.message : undefined) ||
      '请求失败'
    )
  }

  if (error instanceof Error) {
    return error.message && error.message.trim() ? error.message : '请求失败'
  }

  if (typeof error === 'string') {
    return error.trim() ? error : '请求失败'
  }

  return '请求失败'
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
    const token = Cookies.get('admin_token')
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
      Cookies.remove('admin_token')
      try {
        if (typeof window !== 'undefined') {
          window.localStorage.removeItem('admin-auth-storage')
        }
      } catch {
      }
      if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
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
  total_pages: number
}

export interface ModelProviderResponse {
  id: number
  operation: string
  model_id: string
  model_name?: string
  channel_id: number
  channel_name?: string
  upstream_model_name?: string
  status?: string
  model?: { id: string; name: string }
  channel?: { id: number; name: string }
}

export interface TopologyProviderMetrics {
  window_seconds: number
  request_count: number
  success_rate: number
  avg_latency_ms: number
}

export interface TopologyProviderPricingRule {
  id: number
  provider_id: number
  operation: string
  unit: string
  cost_per_unit: string | number
  price_per_unit: string | number
  meta?: Record<string, unknown>
  enabled: boolean
  updated_at: string
}

export interface TopologyInstance {
  id: number
  provider_id: number
  name: string
  instance_type: string
  status: string
  weight: number
  max_concurrency: number
  rpm_limit: number
  tpm_limit: number
  resource_account_id?: number
  resource_account_name?: string
}

export interface TopologyProvider {
  id: number
  operation: string
  model_id: string
  model_name: string
  channel_id: number
  channel_name: string
  upstream_model_name: string
  actual_cost_per_1k_input: string | number
  actual_cost_per_1k_output: string | number
  status: string
  circuit_state: string
  health_score: string | number
  total_requests: number
  metrics?: TopologyProviderMetrics
  pricing_rules?: TopologyProviderPricingRule[]
  instances: TopologyInstance[]
}

export interface TopologyOperation {
  operation: string
  providers: TopologyProvider[]
}

export interface TopologyModel {
  id: string
  name: string
  operations: TopologyOperation[]
}

export interface ModelProviderTopologyResponse {
  generated_at: string
  models: TopologyModel[]
}

export interface ProviderPricingRule {
  id: number
  provider_id: number
  operation: string
  unit: string
  cost_per_unit: string | number
  price_per_unit: string | number
  meta?: Record<string, unknown>
  enabled: boolean
  created_at: string
  updated_at: string
  provider?: {
    id: number
    operation?: string
    model_id?: string
    channel_id?: number
    channel?: { id: number; name: string }
    model?: { id: string; name: string }
  }
}

export interface CreateProviderPricingRuleRequest {
  provider_id: number
  operation: string
  unit: string
  cost_per_unit: number
  price_per_unit: number
  meta?: Record<string, unknown>
  enabled?: boolean
}

export interface UpdateProviderPricingRuleRequest {
  operation?: string
  unit?: string
  cost_per_unit?: number
  price_per_unit?: number
  meta?: Record<string, unknown>
  enabled?: boolean
}

export interface ProviderRateLimitRule {
  id: number
  scope?: string
  provider_id: number
  instance_id?: number | null
  operation: string
  unit: string
  limit: number
  window_seconds: number
  enabled: boolean
  created_at: string
  updated_at: string
  provider?: {
    id: number
    operation?: string
    model_id?: string
    channel_id?: number
    channel?: { id: number; name: string }
    model?: { id: string; name: string }
  }
}

export interface CreateProviderRateLimitRuleRequest {
  scope?: string
  provider_id: number
  instance_id?: number
  operation: string
  unit: string
  limit: number
  window_seconds: number
  enabled?: boolean
}

export interface UpdateProviderRateLimitRuleRequest {
  scope?: string
  instance_id?: number | null
  operation?: string
  unit?: string
  limit?: number
  window_seconds?: number
  enabled?: boolean
}

export interface ModelCapability {
  id: number
  model_id: string
  operation: string
  enabled: boolean
  created_at: string
  updated_at: string
  model?: {
    id: string
    name: string
  }
}

export interface CreateModelCapabilityRequest {
  model_id: string
  operation: string
  enabled?: boolean
}

export interface UpdateModelCapabilityRequest {
  enabled?: boolean
}

// 管理员认证 API
export const authApi = {
  login: (data: { username: string; password: string }) =>
    api.post<unknown, ApiResponse<{ token: string; user: Admin }>>('/api/admin/login', data),
}

// 用户管理 API
export const userApi = {
  list: (params: { page?: number; page_size?: number; keyword?: string; status?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<User>>>('/api/admin/users', { params }),

  get: (id: string) =>
    api.get<unknown, ApiResponse<User>>(`/api/admin/users/${id}`),

  stats: (id: string, params?: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<UserStats>>(`/api/admin/users/${id}/stats`, { params }),

  update: (id: string, data: Partial<User>) =>
    api.put<unknown, ApiResponse>(`/api/admin/users/${id}`, data),

  updateBalance: (id: string, amount: number, reason: string) =>
    api.put<unknown, ApiResponse>(`/api/admin/users/${id}/balance`, { amount, reason }),

  updateStatus: (id: string, status: string) =>
    api.put<unknown, ApiResponse>(`/api/admin/users/${id}/status`, { status }),
}

// 渠道管理 API
export const channelApi = {
  list: (params?: { page?: number; page_size?: number; status?: string; type?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Channel>>>('/api/admin/channels', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<Channel>>(`/api/admin/channels/${id}`),

  create: (data: CreateChannelRequest) =>
    api.post<unknown, ApiResponse<Channel>>('/api/admin/channels', data),

  update: (id: number, data: UpdateChannelRequest) =>
    api.put<unknown, ApiResponse<Channel>>(`/api/admin/channels/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/channels/${id}`),

  test: (id: number) =>
    api.post<unknown, ApiResponse<ChannelTestResult>>(`/api/admin/channels/${id}/test`),

  updateStatus: (id: number, status: string) =>
    api.put<unknown, ApiResponse<Channel>>(`/api/admin/channels/${id}/status`, { status }),
}

// 模型管理 API
export const modelApi = {
  list: (params?: { page?: number; page_size?: number; enabled?: boolean; type?: string; keyword?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Model>>>('/api/admin/models', { params }),

  get: (id: string) =>
    api.get<unknown, ApiResponse<Model>>(`/api/admin/models/${id}`),

  create: (data: CreateModelRequest) =>
    api.post<unknown, ApiResponse<Model>>('/api/admin/models', data),

  update: (id: string, data: UpdateModelRequest) =>
    api.put<unknown, ApiResponse<Model>>(`/api/admin/models/${id}`, data),

  delete: (id: string) =>
    api.delete<unknown, ApiResponse>(`/api/admin/models/${id}`),

  updateStatus: (id: string, enabled: boolean) =>
    api.put<unknown, ApiResponse>(`/api/admin/models/${id}/status`, { enabled }),
}

// 模型能力（operation 开关）API
export const modelCapabilityApi = {
  list: (params?: { model_id?: string; operation?: string; page?: number; page_size?: number }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ModelCapability>>>('/api/admin/model-capabilities', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<ModelCapability>>(`/api/admin/model-capabilities/${id}`),

  create: (data: CreateModelCapabilityRequest) =>
    api.post<unknown, ApiResponse<ModelCapability>>('/api/admin/model-capabilities', data),

  update: (id: number, data: UpdateModelCapabilityRequest) =>
    api.put<unknown, ApiResponse<ModelCapability>>(`/api/admin/model-capabilities/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/model-capabilities/${id}`),
}

// 模型源头（ProviderGroup）API：用于规则页选择 provider_id
export const modelProviderApi = {
  list: (params?: {
    page?: number
    page_size?: number
    operation?: string
    model_id?: string
    channel_id?: number
    status?: string
    circuit_state?: string
  }) => api.get<unknown, ApiResponse<PaginatedResponse<ModelProviderResponse>>>('/api/admin/providers', { params }),
}

export interface ProviderInstanceResponse {
  id: number
  provider_id: number
  provider_name?: string
  name: string
  instance_type: string
  resource_account_id?: number
  account_name?: string
  weight: number
  status: string
  max_concurrency: number
  rpm_limit: number
  tpm_limit: number
  total_requests: number
  success_requests: number
  failed_requests: number
  success_rate: number
  avg_latency_ms: number
  created_at: string
  updated_at: string
}

export const providerInstanceApi = {
  list: (providerId: number, params?: { page?: number; page_size?: number; status?: string; instance_type?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ProviderInstanceResponse>>>(`/api/admin/providers/${providerId}/instances`, { params }),
}

// 拓扑（模型层 -> 源头层 -> 实例层）
export const topologyApi = {
  modelProviders: (params?: {
    model_id?: string
    operation?: string
    metrics_window_seconds?: number
    include_instances?: boolean
    include_pricing_rules?: boolean
  }) =>
    api.get<unknown, ApiResponse<ModelProviderTopologyResponse>>('/api/admin/topology/model-providers', { params }),
}

// 多模态计费规则 API
export const providerPricingRuleApi = {
  list: (params?: { provider_id?: number; operation?: string; page?: number; page_size?: number }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ProviderPricingRule>>>('/api/admin/provider-pricing-rules', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<ProviderPricingRule>>(`/api/admin/provider-pricing-rules/${id}`),

  create: (data: CreateProviderPricingRuleRequest) =>
    api.post<unknown, ApiResponse<ProviderPricingRule>>('/api/admin/provider-pricing-rules', data),

  update: (id: number, data: UpdateProviderPricingRuleRequest) =>
    api.put<unknown, ApiResponse<ProviderPricingRule>>(`/api/admin/provider-pricing-rules/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/provider-pricing-rules/${id}`),
}

// 多模态限流规则 API
export const providerRateLimitRuleApi = {
  list: (params?: { provider_id?: number; instance_id?: number; scope?: string; operation?: string; page?: number; page_size?: number }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ProviderRateLimitRule>>>('/api/admin/provider-rate-limit-rules', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<ProviderRateLimitRule>>(`/api/admin/provider-rate-limit-rules/${id}`),

  create: (data: CreateProviderRateLimitRuleRequest) =>
    api.post<unknown, ApiResponse<ProviderRateLimitRule>>('/api/admin/provider-rate-limit-rules', data),

  update: (id: number, data: UpdateProviderRateLimitRuleRequest) =>
    api.put<unknown, ApiResponse<ProviderRateLimitRule>>(`/api/admin/provider-rate-limit-rules/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/provider-rate-limit-rules/${id}`),
}

// 日志管理 API
export const logApi = {
  list: (params: {
    page?: number
    page_size?: number
    user_id?: string
    model?: string
    channel_id?: number
    status?: string
    start_date?: string
    end_date?: string
  }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Log>>>('/api/admin/logs', { params }),

  stats: (params?: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<LogStats>>('/api/admin/logs/stats', { params }),
}

// 订单管理 API
export const orderApi = {
  list: (params: { page?: number; page_size?: number; status?: string; user_id?: string; payment_method?: string; start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Order>>>('/api/admin/orders', { params }),

  get: (id: string) =>
    api.get<unknown, ApiResponse<Order>>(`/api/admin/orders/${id}`),

  updateStatus: (id: string, status: string) =>
    api.put<unknown, ApiResponse>(`/api/admin/orders/${id}/status`, { status }),

  stats: (params?: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<OrderStats>>('/api/admin/orders/stats', { params }),
}

// 财务报表 API
export const financeApi = {
  overview: (params?: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<FinanceOverview>>('/api/admin/finance/overview', { params }),

  revenue: (params?: { start_date?: string; end_date?: string; group_by?: 'day' | 'week' | 'month' }) =>
    api.get<unknown, ApiResponse<RevenueData[]>>('/api/admin/finance/revenue', { params }),

  cost: (params?: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<CostData[]>>('/api/admin/finance/cost', { params }),

  profit: (params?: { start_date?: string; end_date?: string }) =>
    api.get<unknown, ApiResponse<ProfitData[]>>('/api/admin/finance/profit', { params }),

  topUsers: (params?: { start_date?: string; end_date?: string; limit?: number }) =>
    api.get<unknown, ApiResponse<TopUser[]>>('/api/admin/finance/top-users', { params }),
}

// 公告管理 API
export const announcementApi = {
  list: (params?: { page?: number; page_size?: number; status?: string; type?: string; keyword?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Announcement>>>('/api/admin/announcements', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<Announcement>>(`/api/admin/announcements/${id}`),

  create: (data: CreateAnnouncementRequest) =>
    api.post<unknown, ApiResponse<Announcement>>('/api/admin/announcements', data),

  update: (id: number, data: UpdateAnnouncementRequest) =>
    api.put<unknown, ApiResponse<Announcement>>(`/api/admin/announcements/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/announcements/${id}`),

  publish: (id: number) =>
    api.post<unknown, ApiResponse>(`/api/admin/announcements/${id}/publish`),

  archive: (id: number) =>
    api.post<unknown, ApiResponse>(`/api/admin/announcements/${id}/archive`),
}

// 资源账户管理 API
export const resourceAccountApi = {
  list: (params?: { page?: number; page_size?: number; status?: string; channel_id?: number; keyword?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<ResourceAccount>>>('/api/admin/resource-accounts', { params }),

  get: (id: number) =>
    api.get<unknown, ApiResponse<ResourceAccount>>(`/api/admin/resource-accounts/${id}`),

  create: (data: CreateResourceAccountRequest) =>
    api.post<unknown, ApiResponse<ResourceAccount>>('/api/admin/resource-accounts', data),

  update: (id: number, data: UpdateResourceAccountRequest) =>
    api.put<unknown, ApiResponse<ResourceAccount>>(`/api/admin/resource-accounts/${id}`, data),

  delete: (id: number) =>
    api.delete<unknown, ApiResponse>(`/api/admin/resource-accounts/${id}`),

  refresh: (id: number) =>
    api.post<unknown, ApiResponse>(`/api/admin/resource-accounts/${id}/refresh`),

  stats: () =>
    api.get<unknown, ApiResponse<ResourceAccountStats>>('/api/admin/resource-accounts/stats'),
}

// 系统监控 API
export const systemApi = {
  // 获取系统监控数据
  getMonitor: () =>
    api.get<unknown, ApiResponse<{
      cpu_percent: number
      memory_percent: number
      redis_connections: number
      db_connections: number
    }>>('/api/admin/dashboard/system'),

  // 获取异常告警
  getAlerts: () =>
    api.get<unknown, ApiResponse<Array<{
      id: string
      message: string
      level: 'info' | 'warning' | 'error' | 'critical'
      created_at: string
    }>>>('/api/admin/dashboard/alerts'),

  // 保存系统设置
  saveSettings: (section: string, data: Record<string, unknown>) =>
    api.put<unknown, ApiResponse>(`/api/admin/settings/${section}`, data),

  // 获取系统设置
  getSettings: (section: string) =>
    api.get<unknown, ApiResponse<Record<string, unknown>>>(`/api/admin/settings/${section}`),
}

// 告警管理 API
export const alertApi = {
  list: (params?: { page?: number; page_size?: number; type?: string; severity?: string; status?: string }) =>
    api.get<unknown, ApiResponse<PaginatedResponse<Alert>>>('/api/admin/alerts', { params }),

  get: (id: string) =>
    api.get<unknown, ApiResponse<Alert>>(`/api/admin/alerts/${id}`),

  resolve: (id: string) =>
    api.put<unknown, ApiResponse>(`/api/admin/alerts/${id}/resolve`, {}),

  stats: () =>
    api.get<unknown, ApiResponse<AlertStats>>('/api/admin/alerts/stats'),

  active: () =>
    api.get<unknown, ApiResponse<Alert[]>>('/api/admin/alerts/active'),
}

// 类型定义
export interface Admin {
  id: string
  username: string
  email: string
  role: string
  created_at: string
}

export interface User {
  id: string
  username: string
  email: string
  avatar?: string
  balance: number
  role: string
  status: string
  created_at: string
  updated_at: string
}

export interface Alert {
  id: string
  type: string
  severity: 'info' | 'warning' | 'error' | 'critical' | string
  status: 'active' | 'resolved' | 'ignored' | string
  title: string
  message: string
  metadata?: Record<string, unknown>
  resolved_at?: string
  resolved_by?: string
  created_at: string
}

export interface AlertStats {
  total_alerts: number
  active_alerts: number
  critical_alerts: number
  warning_alerts: number
}

export interface UserStats {
  total_requests: number
  total_tokens: number
  total_cost: number
  avg_latency_ms: number
  daily_stats: DailyStat[]
  model_stats: ModelStat[]
}

export interface Channel {
  id: number
  name: string
  type: string
  base_url: string
  models: string[]
  weight: number
  priority: number
  status: string
  latency: number
  error_count: number
  max_concurrent: number
  rate_limit: number
  timeout_ms: number
  retry_count: number
  last_test_at?: string
  last_test_latency?: number
  success_rate?: number | string
  created_at: string
}

export interface CreateChannelRequest {
  name: string
  type: string
  base_url: string
  api_key?: string
  models: string[]
  weight: number
  priority: number
  max_concurrent: number
  rate_limit: number
  timeout_ms: number
  retry_count: number
  config?: Record<string, unknown>
}

export interface UpdateChannelRequest {
  name?: string
  type?: string
  base_url?: string
  api_key?: string
  models?: string[]
  weight?: number
  priority?: number
  status?: string
  max_concurrent?: number
  rate_limit?: number
  timeout_ms?: number
  retry_count?: number
  config?: Record<string, unknown>
}

export interface ChannelTestResult {
  success: boolean
  status: string
  latency: number
  message: string
  response?: string
}

export interface Model {
  id: string
  name: string
  display_name: string
  provider: string
  input_price: number
  output_price: number
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

export interface CreateModelRequest {
  name: string
  display_name: string
  provider: string
  input_price: number
  output_price: number
  max_tokens: number
  max_context: number
  description?: string
}

export interface UpdateModelRequest {
  display_name?: string
  input_price?: number
  output_price?: number
  max_tokens?: number
  max_context?: number
  status?: string
  description?: string
}

export interface Log {
  id: string
  user_id: string
  username?: string
  token_key?: string
  channel_id: number
  channel_name?: string
  model: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  total_cost: number
  upstream_cost: number
  profit: number
  duration: number
  duration_ms?: number
  status_code?: number
  status: string
  is_stream: boolean
  error_message?: string
  request_ip?: string
  created_at: string
}

export interface LogStats {
  summary: {
    total_requests: number
    success_requests: number
    failed_requests: number
    total_tokens: number
    total_cost: number
    upstream_cost: number
    profit: number
    avg_latency: number
    p95_latency?: number
    p99_latency?: number
  }
  daily_stats: DailyStat[]
  model_stats: ModelStat[]
  channel_stats: ChannelStat[]
}

export interface Order {
  id: string
  user_id: string
  username?: string
  order_no: string
  amount: number
  currency: string
  payment_method: string
  status: string
  paid_at?: string
  created_at: string
}

export interface Announcement {
  id: number
  title: string
  content: string
  type: string
  status: string
  priority: number
  published_at?: string
  expires_at?: string
  created_at: string
  updated_at: string
}

export interface CreateAnnouncementRequest {
  title: string
  content: string
  type?: string
  status?: string
  priority?: number
  expires_at?: string | Date
}

export interface UpdateAnnouncementRequest {
  title?: string
  content?: string
  type?: string
  status?: string
  priority?: number
  expires_at?: string | Date
}

export interface ResourceAccount {
  id: number
  channel_id: number
  channel?: Channel
  account_name: string
  credentials?: Record<string, string>
  status: string
  last_active_at?: string
  error_count: number
  last_error?: string
  expires_at?: string
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateResourceAccountRequest {
  channel_id: number
  account_name: string
  credentials: Record<string, string>
  status?: string
  expires_at?: string | Date
  metadata?: Record<string, unknown>
}

export interface UpdateResourceAccountRequest {
  account_name?: string
  credentials?: Record<string, string>
  status?: string
  expires_at?: string | Date
  metadata?: Record<string, unknown>
}

export interface ResourceAccountStats {
  total_accounts: number
  active_accounts: number
  inactive_accounts: number
  expired_accounts: number
  banned_accounts: number
}

export interface SystemStats {
  users: {
    total: number
    active: number
  }
  channels: {
    total: number
    healthy: number
    unhealthy: number
  }
  resource_accounts?: ResourceAccountStats
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

export interface ChannelStat {
  channel_id: number
  channel_name?: string
  request_count: number
  success_count: number
  fail_count: number
  success_rate: number
  avg_latency: number
  total_cost: number
}

export interface OrderStats {
  total_orders: number
  paid_orders: number
  pending_orders: number
  failed_orders: number
  total_amount: number
  paid_amount: number
}

export interface FinanceOverview {
  total_revenue: number
  total_cost: number
  total_profit: number
  profit_margin: number
  today_revenue: number
  today_cost: number
  today_profit: number
  month_revenue: number
  month_cost: number
  month_profit: number
}

export interface RevenueData {
  date: string
  revenue: number
  orders: number
}

export interface CostData {
  date: string
  cost: number
  requests: number
}

export interface ProfitData {
  date: string
  revenue: number
  cost: number
  profit: number
}

export interface TopUser {
  user_id: string
  username: string
  email: string
  total_spent: number
  total_requests: number
  avg_cost_per_request: number
}

export default api
