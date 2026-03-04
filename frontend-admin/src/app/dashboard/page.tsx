'use client'

import { useCallback, useEffect, useState } from 'react'
import api, { logApi, systemApi, getErrorMessage, type ApiResponse } from '@/lib/api'
import { useToast } from '@/components/ui/use-toast'
import {
  Users,
  Server,
  DollarSign,
  Activity,
  TrendingUp,
  ArrowUpRight,
  ArrowDownRight,
  Zap,
  Clock,
  AlertCircle,
} from 'lucide-react'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
} from 'recharts'

interface DashboardStats {
  totalUsers: number
  activeUsers: number
  totalChannels: number
  healthyChannels: number
  totalRevenue: number
  todayRevenue: number
  totalRequests: number
  todayRequests: number
}

interface DashboardData {
  total_users?: number
  active_users?: number
  healthy_channels?: number
  unhealthy_channels?: number
}

interface SystemMonitor {
  cpu_percent: number
  memory_percent: number
  redis_connections: number
  db_connections: number
}

interface Alert {
  id: string
  message: string
  level: 'info' | 'warning' | 'error' | 'critical'
  created_at: string
}

interface ChartData {
  name: string
  requests: number
  revenue: number
}

const COLORS = ['#f97316', '#22c55e', '#3b82f6', '#a855f7', '#ef4444']

function formatDayLabel(date: string) {
  if (!date) return ''
  const parts = date.split('-')
  if (parts.length !== 3) return date
  return `${parts[1]}/${parts[2]}`
}

export default function DashboardPage() {
  const { toast } = useToast()
  const [stats, setStats] = useState<DashboardStats>({
    totalUsers: 0,
    activeUsers: 0,
    totalChannels: 0,
    healthyChannels: 0,
    totalRevenue: 0,
    todayRevenue: 0,
    totalRequests: 0,
    todayRequests: 0,
  })
  const [chartData, setChartData] = useState<ChartData[]>([])
  const [modelUsage, setModelUsage] = useState<{ name: string; value: number }[]>([])
  const [systemMonitor, setSystemMonitor] = useState<SystemMonitor>({
    cpu_percent: 0,
    memory_percent: 0,
    redis_connections: 0,
    db_connections: 0,
  })
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [avgLatencyMs, setAvgLatencyMs] = useState(0)
  const [p95LatencyMs, setP95LatencyMs] = useState(0)
  const [p99LatencyMs, setP99LatencyMs] = useState(0)
  const [successRate, setSuccessRate] = useState(0)
  const [loading, setLoading] = useState(true)

  const loadDashboardData = useCallback(async () => {
    try {
      setLoading(true)

      const dashboardPromise = api.get<unknown, ApiResponse<DashboardData>>('/api/admin/dashboard')
      const logStatsPromise = logApi.stats()
      const monitorPromise = systemApi.getMonitor()
      const alertsPromise = systemApi.getAlerts()

      const [dashboardRes, logStatsRes, monitorRes, alertsRes] = await Promise.all([
        dashboardPromise,
        logStatsPromise,
        monitorPromise,
        alertsPromise,
      ] as const)

      const dashboardData = dashboardRes.data ?? {}
      const logStats = logStatsRes?.data

      const summary = logStats?.summary
      const totalRevenue = summary?.total_cost ? Number(summary.total_cost) : 0
      const totalRequests = summary?.total_requests ? Number(summary.total_requests) : 0
      const successRequests = summary?.success_requests ? Number(summary.success_requests) : 0
      const avgLatency = summary?.avg_latency ? Number(summary.avg_latency) : 0
      const p95Latency = summary?.p95_latency ? Number(summary.p95_latency) : 0
      const p99Latency = summary?.p99_latency ? Number(summary.p99_latency) : 0

      setAvgLatencyMs(avgLatency)
      setP95LatencyMs(p95Latency)
      setP99LatencyMs(p99Latency)
      setSuccessRate(totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0)

      const dailyStats = logStats?.daily_stats ?? []
      const todayKey = new Date().toLocaleDateString('en-CA') // YYYY-MM-DD (local)
      const todayStat = dailyStats.find((d) => d.date === todayKey)
      const todayRevenue = todayStat?.cost ? Number(todayStat.cost) : 0
      const todayRequests = todayStat?.requests ? Number(todayStat.requests) : 0

      const healthyChannels = Number(dashboardData.healthy_channels || 0)
      const unhealthyChannels = Number(dashboardData.unhealthy_channels || 0)
      const totalChannels = healthyChannels + unhealthyChannels

      setStats({
        totalUsers: Number(dashboardData.total_users || 0),
        activeUsers: Number(dashboardData.active_users || 0),
        totalChannels,
        healthyChannels,
        totalRevenue,
        todayRevenue,
        totalRequests,
        todayRequests,
      })

      const last7Days = dailyStats.slice(-7).map((d) => ({
        name: formatDayLabel(d.date),
        requests: Number(d.requests || 0),
        revenue: Number(d.cost || 0),
      }))
      setChartData(last7Days)

      const modelStats = logStats?.model_stats ?? []
      const totalModelRequests = modelStats.reduce((sum, s) => sum + Number(s.requests || 0), 0)
      const sortedModels = [...modelStats].sort((a, b) => Number(b.requests || 0) - Number(a.requests || 0))

      const topModels = sortedModels.slice(0, 4)
      const restModels = sortedModels.slice(4)
      const restRequests = restModels.reduce((sum, s) => sum + Number(s.requests || 0), 0)

      const usageItems: { name: string; value: number }[] = topModels
        .map((m) => ({
          name: String(m.model || '未知'),
          value: totalModelRequests > 0 ? Number(((Number(m.requests || 0) / totalModelRequests) * 100).toFixed(1)) : 0,
        }))
        .filter((item) => item.value > 0)

      if (restRequests > 0 && totalModelRequests > 0) {
        usageItems.push({
          name: '其他',
          value: Number(((restRequests / totalModelRequests) * 100).toFixed(1)),
        })
      }

      setModelUsage(usageItems)

      if (monitorRes?.data) {
        setSystemMonitor(monitorRes.data)
      }
      if (alertsRes?.data) {
        setAlerts(alertsRes.data)
      }
    } catch (error) {
      toast({
        title: '加载仪表盘失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
      // 设置默认值，避免页面崩溃
      setStats({
        totalUsers: 0,
        activeUsers: 0,
        totalChannels: 0,
        healthyChannels: 0,
        totalRevenue: 0,
        todayRevenue: 0,
        totalRequests: 0,
        todayRequests: 0,
      })
      setChartData([])
      setModelUsage([])
      setSystemMonitor({
        cpu_percent: 0,
        memory_percent: 0,
        redis_connections: 0,
        db_connections: 0,
      })
      setAlerts([])
      setAvgLatencyMs(0)
      setP95LatencyMs(0)
      setP99LatencyMs(0)
      setSuccessRate(0)
    } finally {
      setLoading(false)
    }
  }, [toast])

  useEffect(() => {
    loadDashboardData()
  }, [loadDashboardData])

  const statCards = [
    {
      title: '总用户数',
      value: stats.totalUsers,
      subValue: `${stats.activeUsers} 活跃`,
      icon: Users,
      color: 'bg-blue-500',
      trend: '+12%',
      trendUp: true,
    },
    {
      title: '渠道数量',
      value: stats.totalChannels,
      subValue: `${stats.healthyChannels} 健康`,
      icon: Server,
      color: 'bg-green-500',
      trend: '+3',
      trendUp: true,
    },
    {
      title: '总收入',
      value: `¥${stats.totalRevenue.toFixed(2)}`,
      subValue: `今日 ¥${stats.todayRevenue.toFixed(2)}`,
      icon: DollarSign,
      color: 'bg-orange-500',
      trend: '+8.5%',
      trendUp: true,
    },
    {
      title: '请求总数',
      value: stats.totalRequests.toLocaleString(),
      subValue: `今日 ${stats.todayRequests.toLocaleString()}`,
      icon: Activity,
      color: 'bg-purple-500',
      trend: '+15%',
      trendUp: true,
    },
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
      <div>
        <h1 className="text-2xl font-bold">仪表盘</h1>
        <p className="text-muted-foreground">系统运行状态概览</p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {statCards.map((card, index) => (
          <div
            key={index}
            className="bg-card border border-border rounded-xl p-6 hover:shadow-lg transition-shadow"
          >
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">{card.title}</p>
                <p className="text-2xl font-bold mt-1">{card.value}</p>
                <p className="text-sm text-muted-foreground mt-1">{card.subValue}</p>
              </div>
              <div className={`${card.color} p-3 rounded-lg`}>
                <card.icon className="w-6 h-6 text-white" />
              </div>
            </div>
            <div className="flex items-center mt-4 text-sm">
              {card.trendUp ? (
                <ArrowUpRight className="w-4 h-4 text-green-500 mr-1" />
              ) : (
                <ArrowDownRight className="w-4 h-4 text-red-500 mr-1" />
              )}
              <span className={card.trendUp ? 'text-green-500' : 'text-red-500'}>
                {card.trend}
              </span>
              <span className="text-muted-foreground ml-1">较上周</span>
            </div>
          </div>
        ))}
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Request Trend Chart */}
        <div className="lg:col-span-2 bg-card border border-border rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">请求趋势</h3>
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="colorRequests" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#f97316" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#f97316" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                <XAxis dataKey="name" stroke="#9ca3af" />
                <YAxis stroke="#9ca3af" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="requests"
                  stroke="#f97316"
                  fillOpacity={1}
                  fill="url(#colorRequests)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Model Usage Pie Chart */}
        <div className="bg-card border border-border rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">模型使用分布</h3>
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={modelUsage}
                  cx="50%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={100}
                  paddingAngle={5}
                  dataKey="value"
                >
                  {modelUsage.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
            <div className="flex flex-wrap justify-center gap-4 mt-4">
              {modelUsage.map((entry, index) => (
                <div key={entry.name} className="flex items-center gap-2">
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: COLORS[index % COLORS.length] }}
                  />
                  <span className="text-sm text-muted-foreground">
                    {entry.name} ({entry.value}%)
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Revenue Chart */}
      <div className="bg-card border border-border rounded-xl p-6">
        <h3 className="text-lg font-semibold mb-4">收入趋势</h3>
        <div className="h-64">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
              <XAxis dataKey="name" stroke="#9ca3af" />
              <YAxis stroke="#9ca3af" />
              <Tooltip
                contentStyle={{
                  backgroundColor: '#1f2937',
                  border: '1px solid #374151',
                  borderRadius: '8px',
                }}
              />
              <Bar dataKey="revenue" fill="#22c55e" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-card border border-border rounded-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-orange-500/10 rounded-lg">
              <Zap className="w-5 h-5 text-orange-500" />
            </div>
            <h3 className="font-semibold">系统状态</h3>
          </div>
          <div className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">CPU 使用率</span>
              <span className="font-medium">{systemMonitor.cpu_percent.toFixed(1)}%</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">内存使用率</span>
              <span className="font-medium">{systemMonitor.memory_percent.toFixed(1)}%</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">Redis 连接数</span>
              <span className="font-medium">{systemMonitor.redis_connections}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">DB 连接数</span>
              <span className="font-medium">{systemMonitor.db_connections}</span>
            </div>
          </div>
        </div>

        {/* 异常告警面板 */}
        <div className="bg-card border border-border rounded-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-red-500/10 rounded-lg">
              <AlertCircle className="w-5 h-5 text-red-500" />
            </div>
            <h3 className="font-semibold">异常告警</h3>
          </div>
          <div className="space-y-3">
            {alerts.length === 0 ? (
              <p className="text-sm text-muted-foreground">暂无异常告警</p>
            ) : (
              alerts.slice(0, 3).map((alert) => (
                <div key={alert.id} className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium">{alert.message}</span>
                    <span className="text-xs text-muted-foreground">
                      {new Date(alert.created_at).toLocaleString('zh-CN')}
                    </span>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        <div className="bg-card border border-border rounded-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-blue-500/10 rounded-lg">
              <Clock className="w-5 h-5 text-blue-500" />
            </div>
            <h3 className="font-semibold">响应时间</h3>
          </div>
          <div className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">平均响应</span>
              <span className="font-medium">{avgLatencyMs > 0 ? `${Math.round(avgLatencyMs)}ms` : '-'}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">P95 响应</span>
              <span className="font-medium">{p95LatencyMs > 0 ? `${Math.round(p95LatencyMs)}ms` : '-'}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">P99 响应</span>
              <span className="font-medium">{p99LatencyMs > 0 ? `${Math.round(p99LatencyMs)}ms` : '-'}</span>
            </div>
          </div>
        </div>

        <div className="bg-card border border-border rounded-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-purple-500/10 rounded-lg">
              <TrendingUp className="w-5 h-5 text-purple-500" />
            </div>
            <h3 className="font-semibold">今日概览</h3>
          </div>
          <div className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">今日请求</span>
              <span className="font-medium">{stats.todayRequests.toLocaleString()}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">今日收入</span>
              <span className="font-medium">¥{stats.todayRevenue.toFixed(2)}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">成功率</span>
              <span className="font-medium text-green-500">{successRate.toFixed(1)}%</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
