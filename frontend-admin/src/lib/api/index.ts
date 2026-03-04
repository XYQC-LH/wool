// lib/api/index.ts
// API 层统一导出

// 导出客户端配置
export { api, API_BASE_URL } from './client'

// 导出错误处理
export { getErrorMessage } from './error-handler'

// 导出类型定义
export type {
  ApiResponse,
  PaginatedResponse,
  Admin,
  User,
  UserStats,
  Channel,
  CreateChannelRequest,
  UpdateChannelRequest,
  ChannelTestResult,
  Model,
  CreateModelRequest,
  UpdateModelRequest,
  ModelCapability,
  CreateModelCapabilityRequest,
  UpdateModelCapabilityRequest,
  ModelProviderResponse,
  ProviderInstanceResponse,
  TopologyProviderMetrics,
  TopologyProviderPricingRule,
  TopologyInstance,
  TopologyProvider,
  TopologyOperation,
  TopologyModel,
  ModelProviderTopologyResponse,
  ProviderPricingRule,
  CreateProviderPricingRuleRequest,
  UpdateProviderPricingRuleRequest,
  ProviderRateLimitRule,
  CreateProviderRateLimitRuleRequest,
  UpdateProviderRateLimitRuleRequest,
  Log,
  LogStats,
  DailyStat,
  ModelStat,
  ChannelStat,
  Order,
  OrderStats,
  Announcement,
  CreateAnnouncementRequest,
  UpdateAnnouncementRequest,
  ResourceAccount,
  ResourceAccountStats,
  CreateResourceAccountRequest,
  UpdateResourceAccountRequest,
  SystemStats,
  FinanceOverview,
  RevenueData,
  CostData,
  ProfitData,
  TopUser,
  Alert,
  AlertStats,
} from './types'

// 导出 API 模块
export { authApi } from './modules/auth'
export { userApi } from './modules/user'
export { channelApi } from './modules/channel'
export { modelApi } from './modules/model'
export { modelCapabilityApi } from './modules/model-capability'
export { modelProviderApi } from './modules/model-provider'
export { providerInstanceApi } from './modules/provider-instance'
export { topologyApi } from './modules/topology'
export { providerPricingRuleApi } from './modules/provider-pricing-rule'
export { providerRateLimitRuleApi } from './modules/provider-rate-limit-rule'
export { logApi } from './modules/log'
export { orderApi } from './modules/order'
export { financeApi } from './modules/finance'
export { announcementApi } from './modules/announcement'
export { resourceAccountApi } from './modules/resource-account'
export { systemApi } from './modules/system'
export { alertApi } from './modules/alert'

// 默认导出 api 实例（保持向后兼容）
export { api as default } from './client'
