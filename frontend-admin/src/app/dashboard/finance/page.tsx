'use client'

import { useCallback, useEffect, useState } from 'react'
import { financeApi, FinanceOverview, RevenueData, CostData, ProfitData, TopUser, getErrorMessage } from '@/lib/api'
import api from '@/lib/api'
import {
  DollarSign,
  TrendingUp,
  TrendingDown,
  Calendar,
  Download,
  RefreshCw,
  Users,
  BarChart3,
  Zap,
  AlertTriangle,
  type LucideIcon,
} from 'lucide-react'
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { formatCurrency } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Loading } from '@/components/common/loading'
import { EmptyState } from '@/components/common/empty-state'
import { useToast } from '@/components/ui/use-toast'

// 模型成本分析类型
interface CostAnalysis {
  total_cost: number
  total_upstream_cost: number
  total_profit: number
  avg_margin: number
  total_tokens: number
  cost_per_1k_tokens: number
  top_cost_providers: ProviderCost[]
}

interface ProviderCost {
  provider_id: number
  provider_name: string
  total_cost: number
  request_count: number
  avg_cost: number
}

interface CostBreakdown {
  model_id: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  upstream_cost: number
  downstream_cost: number
  profit: number
  margin: number
  cost_per_1k_tokens: number
}

interface CostOptimizationSuggestion {
  model_id: string
  current_provider: string
  recommended_provider: string
  savings: number
  savings_percent: number
  reason: string
}

interface Model {
  id: string
  name: string
}

type FinanceTab = 'overview' | 'revenue' | 'cost' | 'profit' | 'users' | 'model-cost'

function getSuccessData<T>(response: unknown): T | null {
  if (!response || typeof response !== 'object') return null
  const record = response as Record<string, unknown>
  if (record.success !== true) return null
  return (record.data as T) ?? null
}

export default function FinancePage() {
  const { toast } = useToast()
  const [overview, setOverview] = useState<FinanceOverview | null>(null)
  const [revenueData, setRevenueData] = useState<RevenueData[]>([])
  const [costData, setCostData] = useState<CostData[]>([])
  const [profitData, setProfitData] = useState<ProfitData[]>([])
  const [topUsers, setTopUsers] = useState<TopUser[]>([])
  const [loading, setLoading] = useState(true)
  const [dateRange, setDateRange] = useState<'7d' | '30d' | '90d'>('30d')
  const [activeTab, setActiveTab] = useState<FinanceTab>('overview')

  // 模型成本分析状态
  const [models, setModels] = useState<Model[]>([])
  const [selectedModel, setSelectedModel] = useState<string>('')
  const [costAnalysis, setCostAnalysis] = useState<CostAnalysis | null>(null)
  const [costBreakdown, setCostBreakdown] = useState<CostBreakdown | null>(null)
  const [optimizationSuggestions, setOptimizationSuggestions] = useState<CostOptimizationSuggestion[]>([])
  const [costLoading, setCostLoading] = useState(false)
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  const loadFinanceData = useCallback(async () => {
    setLoading(true)
    try {
      const endDate = new Date()
      const startDate = new Date()
      
      switch (dateRange) {
        case '7d':
          startDate.setDate(startDate.getDate() - 7)
          break
        case '30d':
          startDate.setDate(startDate.getDate() - 30)
          break
        case '90d':
          startDate.setDate(startDate.getDate() - 90)
          break
      }

      const params = {
        start_date: startDate.toISOString().split('T')[0],
        end_date: endDate.toISOString().split('T')[0],
      }

      const [overviewRes, revenueRes, costRes, profitRes, usersRes] = await Promise.all([
        financeApi.overview(params),
        financeApi.revenue({ ...params, group_by: 'day' }),
        financeApi.cost(params),
        financeApi.profit(params),
        financeApi.topUsers({ ...params, limit: 10 }),
      ])

      if (overviewRes.data) setOverview(overviewRes.data)
      if (revenueRes.data) setRevenueData(revenueRes.data)
      if (costRes.data) setCostData(costRes.data)
      if (profitRes.data) setProfitData(profitRes.data)
      if (usersRes.data) setTopUsers(usersRes.data)
    } catch (error) {
      toast({
        title: '加载财务数据失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [dateRange, toast])

  // 加载模型列表
  const loadModels = useCallback(async () => {
    try {
      const response = await api.get('/api/admin/models', { params: { page_size: '100' } })
      const data = response as { data?: { list?: Model[] } }
      if (data.data?.list) {
        const list = data.data.list
        setModels(list)
        setSelectedModel((prev) => (prev ? prev : list[0]?.id || ''))
      }
    } catch (error) {
      toast({
        title: '加载模型失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }, [toast])

  // 加载成本分析
  const loadCostAnalysis = useCallback(async () => {
    if (!selectedModel) return

    setCostLoading(true)
    try {
      const params: Record<string, string> = {}
      if (startDate) params.start_date = startDate
      if (endDate) params.end_date = endDate

      const [analysisRes, breakdownRes, suggestionsRes] = await Promise.all([
        api.get(`/api/admin/providers/cost/analysis/${selectedModel}`, { params }),
        api.get(`/api/admin/providers/cost/breakdown/${selectedModel}`, {
          params: { prompt_tokens: '1000', completion_tokens: '500' }
        }),
        api.get(`/api/admin/providers/cost/optimization/${selectedModel}`),
      ]) as [unknown, unknown, unknown]

      const analysis = getSuccessData<CostAnalysis>(analysisRes)
      const breakdown = getSuccessData<CostBreakdown>(breakdownRes)
      const suggestions = getSuccessData<CostOptimizationSuggestion[]>(suggestionsRes)
      if (analysis) setCostAnalysis(analysis)
      if (breakdown) setCostBreakdown(breakdown)
      if (suggestions) setOptimizationSuggestions(suggestions)
    } catch {
      toast({
        title: '加载失败',
        description: '无法加载成本分析数据',
        variant: 'destructive',
      })
    } finally {
      setCostLoading(false)
    }
  }, [endDate, selectedModel, startDate, toast])

  useEffect(() => {
    loadFinanceData()
    loadModels()
  }, [loadFinanceData, loadModels])

  useEffect(() => {
    loadCostAnalysis()
  }, [loadCostAnalysis])

  const handleExport = async () => {
    try {
      const data = profitData.map(item => ({
        日期: item.date,
        收入: Number(item.revenue).toFixed(2),
        成本: Number(item.cost).toFixed(2),
        利润: Number(item.profit).toFixed(2),
      }))
      
      const headers = ['日期', '收入', '成本', '利润']
      const csvContent = [
        '\uFEFF', // UTF-8 BOM
        headers.join(','),
        ...data.map(row => Object.values(row).join(',')),
      ].join('\n')

      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `财务报表_${new Date().toISOString().split('T')[0]}.csv`
      link.click()
      URL.revokeObjectURL(url)
      toast({ title: '导出成功', description: '财务报表已导出' })
    } catch (error) {
      toast({
        title: '导出失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }

  const tabs: Array<{ id: FinanceTab; label: string; icon: LucideIcon }> = [
    { id: 'overview', label: '概览', icon: BarChart3 },
    { id: 'revenue', label: '收入', icon: DollarSign },
    { id: 'cost', label: '成本', icon: TrendingDown },
    { id: 'profit', label: '利润', icon: TrendingUp },
    { id: 'users', label: '用户排行', icon: Users },
    { id: 'model-cost', label: '模型成本', icon: Zap },
  ]

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-orange-500"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">财务报表</h1>
          <p className="text-muted-foreground">收入、成本和利润分析</p>
        </div>
        <div className="flex items-center gap-3">
          <select
            value={dateRange}
            onChange={(e) => setDateRange(e.target.value as '7d' | '30d' | '90d')}
            className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
          >
            <option value="7d">最近 7 天</option>
            <option value="30d">最近 30 天</option>
            <option value="90d">最近 90 天</option>
          </select>
          <button
            onClick={loadFinanceData}
            className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg hover:bg-accent transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
            刷新
          </button>
          <button
            onClick={handleExport}
            className="flex items-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
          >
            <Download className="w-4 h-4" />
            导出
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b border-border">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 px-4 py-2 border-b-2 transition-colors ${
              activeTab === tab.id
                ? 'border-orange-500 text-orange-500'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <tab.icon className="w-4 h-4" />
            {tab.label}
          </button>
        ))}
      </div>

      {/* Overview Tab */}
      {activeTab === 'overview' && overview && (
        <div className="space-y-6">
          {/* Stats Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div className="bg-card border border-border rounded-xl p-6">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">总收入</p>
                  <p className="text-2xl font-bold mt-1">{formatCurrency(overview.total_revenue)}</p>
                  <p className="text-sm text-muted-foreground mt-1">本月 {formatCurrency(overview.month_revenue)}</p>
                </div>
                <div className="p-3 bg-green-500/10 rounded-lg">
                  <DollarSign className="w-6 h-6 text-green-500" />
                </div>
              </div>
            </div>

            <div className="bg-card border border-border rounded-xl p-6">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">总成本</p>
                  <p className="text-2xl font-bold mt-1">{formatCurrency(overview.total_cost)}</p>
                  <p className="text-sm text-muted-foreground mt-1">本月 {formatCurrency(overview.month_cost)}</p>
                </div>
                <div className="p-3 bg-red-500/10 rounded-lg">
                  <TrendingDown className="w-6 h-6 text-red-500" />
                </div>
              </div>
            </div>

            <div className="bg-card border border-border rounded-xl p-6">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">净利润</p>
                  <p className="text-2xl font-bold mt-1">{formatCurrency(overview.total_profit)}</p>
                  <p className="text-sm text-muted-foreground mt-1">利润率 {Number(overview.profit_margin).toFixed(1)}%</p>
                </div>
                <div className="p-3 bg-orange-500/10 rounded-lg">
                  <TrendingUp className="w-6 h-6 text-orange-500" />
                </div>
              </div>
            </div>

            <div className="bg-card border border-border rounded-xl p-6">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">今日数据</p>
                  <p className="text-2xl font-bold mt-1">{formatCurrency(overview.today_profit)}</p>
                  <p className="text-sm text-muted-foreground mt-1">
                    收入 {formatCurrency(overview.today_revenue)}
                  </p>
                </div>
                <div className="p-3 bg-blue-500/10 rounded-lg">
                  <Calendar className="w-6 h-6 text-blue-500" />
                </div>
              </div>
            </div>
          </div>

          {/* Charts */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Revenue Trend */}
            <div className="bg-card border border-border rounded-xl p-6">
              <h3 className="text-lg font-semibold mb-4">收入趋势</h3>
              <div className="h-80">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={revenueData}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                    <XAxis dataKey="date" stroke="#9ca3af" />
                    <YAxis stroke="#9ca3af" />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#1f2937',
                        border: '1px solid #374151',
                        borderRadius: '8px',
                      }}
                    />
                    <Legend />
                    <Line type="monotone" dataKey="revenue" stroke="#22c55e" strokeWidth={2} name="收入" />
                    <Line type="monotone" dataKey="orders" stroke="#f97316" strokeWidth={2} name="订单数" />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </div>

            {/* Cost Breakdown */}
            <div className="bg-card border border-border rounded-xl p-6">
              <h3 className="text-lg font-semibold mb-4">成本分析</h3>
              <div className="h-80">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={costData}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                    <XAxis dataKey="date" stroke="#9ca3af" />
                    <YAxis stroke="#9ca3af" />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#1f2937',
                        border: '1px solid #374151',
                        borderRadius: '8px',
                      }}
                    />
                    <Bar dataKey="cost" fill="#ef4444" radius={[4, 4, 0, 0]} name="成本" />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>

          {/* Profit Chart */}
          <div className="bg-card border border-border rounded-xl p-6">
            <h3 className="text-lg font-semibold mb-4">利润分析</h3>
            <div className="h-80">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={profitData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                  <XAxis dataKey="date" stroke="#9ca3af" />
                  <YAxis stroke="#9ca3af" />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#1f2937',
                      border: '1px solid #374151',
                      borderRadius: '8px',
                    }}
                  />
                  <Legend />
                  <Bar dataKey="revenue" fill="#22c55e" radius={[4, 4, 0, 0]} name="收入" />
                  <Bar dataKey="cost" fill="#ef4444" radius={[4, 4, 0, 0]} name="成本" />
                  <Bar dataKey="profit" fill="#f97316" radius={[4, 4, 0, 0]} name="利润" />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>
      )}

      {/* Revenue Tab */}
      {activeTab === 'revenue' && (
        <div className="bg-card border border-border rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">收入详情</h3>
          <div className="h-96">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={revenueData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                <XAxis dataKey="date" stroke="#9ca3af" />
                <YAxis stroke="#9ca3af" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Legend />
                <Line type="monotone" dataKey="revenue" stroke="#22c55e" strokeWidth={2} name="收入" />
                <Line type="monotone" dataKey="orders" stroke="#f97316" strokeWidth={2} name="订单数" />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Cost Tab */}
      {activeTab === 'cost' && (
        <div className="bg-card border border-border rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">成本详情</h3>
          <div className="h-96">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={costData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                <XAxis dataKey="date" stroke="#9ca3af" />
                <YAxis stroke="#9ca3af" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Bar dataKey="cost" fill="#ef4444" radius={[4, 4, 0, 0]} name="成本" />
                <Bar dataKey="requests" fill="#3b82f6" radius={[4, 4, 0, 0]} name="请求数" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Profit Tab */}
      {activeTab === 'profit' && (
        <div className="bg-card border border-border rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">利润详情</h3>
          <div className="h-96">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={profitData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                <XAxis dataKey="date" stroke="#9ca3af" />
                <YAxis stroke="#9ca3af" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Legend />
                <Bar dataKey="revenue" fill="#22c55e" radius={[4, 4, 0, 0]} name="收入" />
                <Bar dataKey="cost" fill="#ef4444" radius={[4, 4, 0, 0]} name="成本" />
                <Bar dataKey="profit" fill="#f97316" radius={[4, 4, 0, 0]} name="利润" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Top Users Tab */}
      {activeTab === 'users' && (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <div className="p-6 border-b border-border">
            <h3 className="text-lg font-semibold">用户消费排行</h3>
            <p className="text-sm text-muted-foreground">消费金额最高的用户</p>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    排名
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    用户
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    总消费
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    请求数
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    平均成本
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {topUsers.map((user, index) => (
                  <tr key={user.user_id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold ${
                        index === 0 ? 'bg-yellow-500 text-white' :
                        index === 1 ? 'bg-gray-400 text-white' :
                        index === 2 ? 'bg-orange-600 text-white' :
                        'bg-muted text-muted-foreground'
                      }`}>
                        {index + 1}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div>
                        <div className="font-medium">{user.username}</div>
                        <div className="text-sm text-muted-foreground">{user.email}</div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="font-medium text-green-500">
                        {formatCurrency(user.total_spent)}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {user.total_requests.toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {formatCurrency(user.avg_cost_per_request)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Model Cost Tab */}
      {activeTab === 'model-cost' && (
        <div className="space-y-6">
          {/* Model Selection */}
          <Card className="p-6">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <div>
                <Label>选择模型</Label>
                <Select value={selectedModel} onValueChange={setSelectedModel}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择模型" />
                  </SelectTrigger>
                  <SelectContent>
                    {models.map((model) => (
                      <SelectItem key={model.id} value={model.id}>
                        {model.name || model.id}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label>开始日期</Label>
                <Input
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                />
              </div>
              <div>
                <Label>结束日期</Label>
                <Input
                  type="date"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                />
              </div>
              <div className="flex items-end">
                <Button onClick={loadCostAnalysis} className="w-full">
                  查询
                </Button>
              </div>
            </div>
          </Card>

          {costLoading ? (
            <Loading />
          ) : costAnalysis && costBreakdown ? (
            <>
              {/* Cost Overview Cards */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
                <Card className="p-6">
                  <div className="flex items-start justify-between">
                    <div>
                      <p className="text-sm text-muted-foreground">总成本</p>
                      <p className="text-2xl font-bold mt-1">{formatCurrency(costAnalysis.total_cost)}</p>
                      <p className="text-sm text-muted-foreground mt-1">
                        上游: {formatCurrency(costAnalysis.total_upstream_cost)}
                      </p>
                    </div>
                    <div className="p-3 bg-red-500/10 rounded-lg">
                      <TrendingDown className="w-6 h-6 text-red-500" />
                    </div>
                  </div>
                </Card>

                <Card className="p-6">
                  <div className="flex items-start justify-between">
                    <div>
                      <p className="text-sm text-muted-foreground">总利润</p>
                      <p className="text-2xl font-bold mt-1">{formatCurrency(costAnalysis.total_profit)}</p>
                      <p className="text-sm text-muted-foreground mt-1">
                        利润率: {Number(costAnalysis.avg_margin).toFixed(1)}%
                      </p>
                    </div>
                    <div className="p-3 bg-green-500/10 rounded-lg">
                      <TrendingUp className="w-6 h-6 text-green-500" />
                    </div>
                  </div>
                </Card>

                <Card className="p-6">
                  <div className="flex items-start justify-between">
                    <div>
                      <p className="text-sm text-muted-foreground">总Tokens</p>
                      <p className="text-2xl font-bold mt-1">
                        {costAnalysis.total_tokens.toLocaleString()}
                      </p>
                      <p className="text-sm text-muted-foreground mt-1">
                        每1K: {formatCurrency(costAnalysis.cost_per_1k_tokens)}
                      </p>
                    </div>
                    <div className="p-3 bg-blue-500/10 rounded-lg">
                      <Zap className="w-6 h-6 text-blue-500" />
                    </div>
                  </div>
                </Card>

                <Card className="p-6">
                  <div className="flex items-start justify-between">
                    <div>
                      <p className="text-sm text-muted-foreground">成本明细</p>
                      <p className="text-2xl font-bold mt-1">
                        {costBreakdown.total_tokens.toLocaleString()} tokens
                      </p>
                      <p className="text-sm text-muted-foreground mt-1">
                        利润率: {Number(costBreakdown.margin).toFixed(1)}%
                      </p>
                    </div>
                    <div className="p-3 bg-orange-500/10 rounded-lg">
                      <BarChart3 className="w-6 h-6 text-orange-500" />
                    </div>
                  </div>
                </Card>
              </div>

              {/* Cost Breakdown */}
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">成本明细</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <p className="text-sm text-muted-foreground mb-2">Prompt Tokens</p>
                    <p className="text-3xl font-bold">{costBreakdown.prompt_tokens.toLocaleString()}</p>
                    <p className="text-sm text-muted-foreground mt-1">
                      成本: {formatCurrency(costBreakdown.upstream_cost)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground mb-2">Completion Tokens</p>
                    <p className="text-3xl font-bold">{costBreakdown.completion_tokens.toLocaleString()}</p>
                    <p className="text-sm text-muted-foreground mt-1">
                      成本: {formatCurrency(costBreakdown.downstream_cost)}
                    </p>
                  </div>
                </div>
                <div className="mt-6 pt-6 border-t border-border">
                  <div className="flex justify-between items-center">
                    <div>
                      <p className="text-sm text-muted-foreground">总成本</p>
                      <p className="text-2xl font-bold">{formatCurrency(costBreakdown.downstream_cost)}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">利润</p>
                      <p className="text-2xl font-bold text-green-500">
                        {formatCurrency(costBreakdown.profit)}
                      </p>
                    </div>
                  </div>
                </div>
              </Card>

              {/* Top Cost Providers */}
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">成本最高的Provider</h3>
                {costAnalysis.top_cost_providers.length > 0 ? (
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead className="bg-muted/50">
                        <tr>
                          <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">
                            Provider
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">
                            总成本
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">
                            请求数
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">
                            平均成本
                          </th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border">
                        {costAnalysis.top_cost_providers.map((provider) => (
                          <tr key={provider.provider_id} className="hover:bg-muted/30">
                            <td className="px-4 py-3">{provider.provider_name}</td>
                            <td className="px-4 py-3">
                              <span className="font-medium text-red-500">
                                {formatCurrency(provider.total_cost)}
                              </span>
                            </td>
                            <td className="px-4 py-3">{provider.request_count.toLocaleString()}</td>
                            <td className="px-4 py-3">{formatCurrency(provider.avg_cost)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <EmptyState
                    title="暂无数据"
                    description="该模型暂无成本数据"
                  />
                )}
              </Card>

              {/* Optimization Suggestions */}
              {optimizationSuggestions.length > 0 && (
                <Card className="p-6">
                  <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <AlertTriangle className="w-5 h-5 text-orange-500" />
                    成本优化建议
                  </h3>
                  <div className="space-y-4">
                    {optimizationSuggestions.map((suggestion, index) => (
                      <div key={index} className="p-4 bg-orange-500/10 border border-orange-500/20 rounded-lg">
                        <div className="flex justify-between items-start mb-2">
                          <div>
                            <p className="font-medium">当前: {suggestion.current_provider}</p>
                            <p className="text-sm text-muted-foreground">
                              建议: {suggestion.recommended_provider}
                            </p>
                          </div>
                          <div className="text-right">
                            <p className="text-2xl font-bold text-green-500">
                              {formatCurrency(suggestion.savings)}
                            </p>
                            <p className="text-sm text-green-600">
                              节省 {Number(suggestion.savings_percent).toFixed(1)}%
                            </p>
                          </div>
                        </div>
                        <p className="text-sm text-muted-foreground">{suggestion.reason}</p>
                      </div>
                    ))}
                  </div>
                </Card>
              )}
            </>
          ) : (
            <EmptyState
              title="请选择模型"
              description="选择一个模型以查看成本分析"
            />
          )}
        </div>
      )}
    </div>
  )
}
