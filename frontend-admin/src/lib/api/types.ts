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
  total_pages: number
}

// 管理员类型
export interface Admin {
  id: string
  username: string
  email: string
  role: string
  created_at: string
}

// 用户类型
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

// 用户统计
export interface UserStats {
  total_requests: number
  total_tokens: number
  total_cost: number
  avg_latency_ms: number
  daily_stats: DailyStat[]
  model_stats: ModelStat[]
}

// 渠道类型
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

// 创建渠道请求
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

// 更新渠道请求
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

// 渠道测试结果
export interface ChannelTestResult {
  success: boolean
  status: string
  latency: number
  message: string
  response?: string
}

// 模型类型
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

// 创建模型请求
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

// 更新模型请求
export interface UpdateModelRequest {
  display_name?: string
  input_price?: number
  output_price?: number
  max_tokens?: number
  max_context?: number
  status?: string
  description?: string
}

// 模型能力类型
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

// 创建模型能力请求
export interface CreateModelCapabilityRequest {
  model_id: string
  operation: string
  enabled?: boolean
}

// 更新模型能力请求
export interface UpdateModelCapabilityRequest {
  enabled?: boolean
}

// 模型源头响应
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

// Provider 实例响应
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

// 拓扑相关类型
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

// Provider 计费规则类型
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

// 创建 Provider 计费规则请求
export interface CreateProviderPricingRuleRequest {
  provider_id: number
  operation: string
  unit: string
  cost_per_unit: number
  price_per_unit: number
  meta?: Record<string, unknown>
  enabled?: boolean
}

// 更新 Provider 计费规则请求
export interface UpdateProviderPricingRuleRequest {
  operation?: string
  unit?: string
  cost_per_unit?: number
  price_per_unit?: number
  meta?: Record<string, unknown>
  enabled?: boolean
}

// Provider 限流规则类型
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

// 创建 Provider 限流规则请求
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

// 更新 Provider 限流规则请求
export interface UpdateProviderRateLimitRuleRequest {
  scope?: string
  instance_id?: number | null
  operation?: string
  unit?: string
  limit?: number
  window_seconds?: number
  enabled?: boolean
}

// 日志类型
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

// 日志统计
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

// 渠道统计
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

// 订单类型
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

// 订单统计
export interface OrderStats {
  total_orders: number
  paid_orders: number
  pending_orders: number
  failed_orders: number
  total_amount: number
  paid_amount: number
}

// 公告类型
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

// 创建公告请求
export interface CreateAnnouncementRequest {
  title: string
  content: string
  type?: string
  status?: string
  priority?: number
  expires_at?: string | Date
}

// 更新公告请求
export interface UpdateAnnouncementRequest {
  title?: string
  content?: string
  type?: string
  status?: string
  priority?: number
  expires_at?: string | Date
}

// 资源账户类型
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

// 创建资源账户请求
export interface CreateResourceAccountRequest {
  channel_id: number
  account_name: string
  credentials: Record<string, string>
  status?: string
  expires_at?: string | Date
  metadata?: Record<string, unknown>
}

// 更新资源账户请求
export interface UpdateResourceAccountRequest {
  account_name?: string
  credentials?: Record<string, string>
  status?: string
  expires_at?: string | Date
  metadata?: Record<string, unknown>
}

// 资源账户统计
export interface ResourceAccountStats {
  total_accounts: number
  active_accounts: number
  inactive_accounts: number
  expired_accounts: number
  banned_accounts: number
}

// 系统统计
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

// 财务概览
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

// 收入数据
export interface RevenueData {
  date: string
  revenue: number
  orders: number
}

// 成本数据
export interface CostData {
  date: string
  cost: number
  requests: number
}

// 利润数据
export interface ProfitData {
  date: string
  revenue: number
  cost: number
  profit: number
}

// 顶级用户
export interface TopUser {
  user_id: string
  username: string
  email: string
  total_spent: number
  total_requests: number
  avg_cost_per_request: number
}

// 告警类型
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

// 告警统计
export interface AlertStats {
  total_alerts: number
  active_alerts: number
  critical_alerts: number
  warning_alerts: number
}
