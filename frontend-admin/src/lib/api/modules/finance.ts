// lib/api/modules/finance.ts
// 财务报表 API

import { api } from '../client'
import { ApiResponse, FinanceOverview, RevenueData, CostData, ProfitData, TopUser } from '../types'

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
