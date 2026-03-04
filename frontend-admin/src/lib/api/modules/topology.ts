// lib/api/modules/topology.ts
// 拓扑（模型层 -> 源头层 -> 实例层）API

import { api } from '../client'
import { ApiResponse, ModelProviderTopologyResponse } from '../types'

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
