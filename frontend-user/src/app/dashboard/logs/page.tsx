'use client'

import { useState } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/store/auth'
import { useLogs, useUsageStats } from '@/lib/query'
import { formatCurrency, formatNumber, formatDate } from '@/lib/utils'
import {
  FileText,
  Activity,
  Zap,
  DollarSign,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  CheckCircle,
  XCircle,
  Clock,
  Filter,
  Loader2,
} from 'lucide-react'
import { LogDetailDialog } from '@/components/log-detail-dialog'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
} from 'recharts'

// 状态配置
const statusConfig: Record<string, { label: string; color: string; icon: React.ComponentType<{ className?: string }> }> = {
  success: { label: '成功', color: 'text-green-500', icon: CheckCircle },
  error: { label: '失败', color: 'text-red-500', icon: XCircle },
  pending: { label: '处理中', color: 'text-yellow-500', icon: Clock },
}

export default function LogsPage() {
  const { isAuthenticated } = useAuthStore()
  const [page, setPage] = useState(1)
  const [modelFilter, setModelFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [showFilters, setShowFilters] = useState(false)
  const [selectedLog, setSelectedLog] = useState<{ id: string; model: string; prompt_tokens: number; completion_tokens: number; total_tokens: number; total_cost: string | number; duration: number; status: string; created_at: string } | null>(null)
  const [showDetailDialog, setShowDetailDialog] = useState(false)

  // 使用 TanStack Query 获取数据
  const { data: logsData, isLoading: logsLoading, refetch: refetchLogs } = useLogs({
    page,
    page_size: 20,
    model: modelFilter || undefined,
    status: statusFilter || undefined,
    start_date: startDate || undefined,
    end_date: endDate || undefined,
  })

  const { data: stats, isLoading: statsLoading, refetch: refetchStats } = useUsageStats({
    start_date: startDate || undefined,
    end_date: endDate || undefined,
  })

  const logs = logsData?.list || []
  const totalPages = Math.max(1, Math.ceil((logsData?.total || 0) / 20))

  const handleRefresh = () => {
    refetchLogs()
    refetchStats()
  }

  const handleResetFilters = () => {
    setModelFilter('')
    setStatusFilter('')
    setStartDate('')
    setEndDate('')
    setPage(1)
  }

  const getStatusInfo = (status: string) => {
    return statusConfig[status] || { label: status, color: 'text-gray-500', icon: Clock }
  }

  const handleLogClick = (log: { id: string; model: string; prompt_tokens: number; completion_tokens: number; total_tokens: number; total_cost: string | number; duration: number; status: string; created_at: string }) => {
    setSelectedLog(log)
    setShowDetailDialog(true)
  }

  // 未登录状态
  if (!isAuthenticated) {
    return (
      <div className="space-y-6 animate-fade-in">
        <div>
          <h1 className="text-2xl font-bold">调用日志</h1>
          <p className="text-muted-foreground">查看您的 API 调用记录和使用统计</p>
        </div>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center py-12">
              <FileText className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">需要登录</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                请先登录以查看您的调用日志
              </p>
              <div className="flex justify-center gap-2 mt-4">
                <Link href="/login">
                  <Button>登录</Button>
                </Link>
                <Link href="/register">
                  <Button variant="outline">注册</Button>
                </Link>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">调用日志</h1>
          <p className="text-muted-foreground">查看您的 API 调用记录和使用统计</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setShowFilters(!showFilters)}>
            <Filter className="mr-2 h-4 w-4" />
            筛选
          </Button>
          <Button variant="outline" size="icon" onClick={handleRefresh}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="card-hover">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总请求数</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {statsLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                formatNumber(stats?.total_requests || 0)
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              API 调用次数
            </p>
          </CardContent>
        </Card>

        <Card className="card-hover">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Token 消耗</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {statsLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                formatNumber(stats?.total_tokens || 0)
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              Token 使用量
            </p>
          </CardContent>
        </Card>

        <Card className="card-hover">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">消费金额</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {statsLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                formatCurrency(stats?.total_cost || 0)
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              消费总额
            </p>
          </CardContent>
        </Card>

        <Card className="card-hover">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">模型数量</CardTitle>
            <FileText className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {statsLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                stats?.model_stats?.length || 0
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              使用的模型数
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 筛选器 */}
      {showFilters && (
        <Card>
          <CardContent className="pt-6">
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
              <div className="space-y-2">
                <label className="text-sm font-medium">模型</label>
                <input
                  type="text"
                  placeholder="输入模型名称"
                  value={modelFilter}
                  onChange={(e) => {
                    setModelFilter(e.target.value)
                    setPage(1)
                  }}
                  className="w-full px-3 py-2 rounded-md border border-input bg-background text-sm"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">状态</label>
                <select
                  value={statusFilter}
                  onChange={(e) => {
                    setStatusFilter(e.target.value)
                    setPage(1)
                  }}
                  className="w-full px-3 py-2 rounded-md border border-input bg-background text-sm"
                >
                  <option value="">全部状态</option>
                  <option value="success">成功</option>
                  <option value="error">失败</option>
                </select>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">开始日期</label>
                <input
                  type="date"
                  value={startDate}
                  onChange={(e) => {
                    setStartDate(e.target.value)
                    setPage(1)
                  }}
                  className="w-full px-3 py-2 rounded-md border border-input bg-background text-sm"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">结束日期</label>
                <input
                  type="date"
                  value={endDate}
                  onChange={(e) => {
                    setEndDate(e.target.value)
                    setPage(1)
                  }}
                  className="w-full px-3 py-2 rounded-md border border-input bg-background text-sm"
                />
              </div>
              <div className="space-y-2 flex items-end">
                <Button variant="outline" onClick={handleResetFilters} className="w-full">
                  重置筛选
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 使用趋势图表 */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>请求趋势</CardTitle>
            <CardDescription>每日 API 调用次数</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[250px]">
              {statsLoading ? (
                <div className="flex items-center justify-center h-full">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={stats?.daily_stats || []}>
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                    <XAxis
                      dataKey="date"
                      stroke="hsl(var(--muted-foreground))"
                      fontSize={12}
                      tickFormatter={(value: string) => value.slice(5)}
                    />
                    <YAxis stroke="hsl(var(--muted-foreground))" fontSize={12} />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: 'hsl(var(--card))',
                        border: '1px solid hsl(var(--border))',
                        borderRadius: '8px',
                      }}
                    />
                    <Line
                      type="monotone"
                      dataKey="requests"
                      stroke="hsl(var(--primary))"
                      strokeWidth={2}
                      dot={false}
                    />
                  </LineChart>
                </ResponsiveContainer>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>模型使用分布</CardTitle>
            <CardDescription>各模型的调用次数</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[250px]">
              {statsLoading ? (
                <div className="flex items-center justify-center h-full">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={(stats?.model_stats || []).slice(0, 5)} layout="vertical">
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                    <XAxis type="number" stroke="hsl(var(--muted-foreground))" fontSize={12} />
                    <YAxis
                      type="category"
                      dataKey="model"
                      stroke="hsl(var(--muted-foreground))"
                      fontSize={12}
                      width={100}
                      tickFormatter={(value: string) => value.length > 12 ? value.slice(0, 12) + '...' : value}
                    />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: 'hsl(var(--card))',
                        border: '1px solid hsl(var(--border))',
                        borderRadius: '8px',
                      }}
                    />
                    <Bar dataKey="requests" fill="hsl(var(--primary))" radius={[0, 4, 4, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 日志列表 */}
      <Card>
        <CardHeader>
          <CardTitle>调用记录</CardTitle>
          <CardDescription>您的 API 调用详细记录</CardDescription>
        </CardHeader>
        <CardContent>
          {logsLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : logs.length === 0 ? (
            <div className="text-center py-12">
              <FileText className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">暂无调用记录</h3>
              <p className="mt-2 text-muted-foreground">开始使用 API 后，调用记录将显示在这里</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left py-3 px-4 font-medium text-muted-foreground">模型</th>
                    <th className="text-left py-3 px-4 font-medium text-muted-foreground">Token</th>
                    <th className="text-left py-3 px-4 font-medium text-muted-foreground">费用</th>
                    <th className="text-left py-3 px-4 font-medium text-muted-foreground">耗时</th>
                    <th className="text-left py-3 px-4 font-medium text-muted-foreground">状态</th>
                    <th className="text-left py-3 px-4 font-medium text-muted-foreground">时间</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((log: { id: string; model: string; prompt_tokens: number; completion_tokens: number; total_tokens: number; total_cost: string | number; duration: number; status: string; created_at: string }) => {
                    const statusInfo = getStatusInfo(log.status)
                    const StatusIcon = statusInfo.icon
                    return (
                      <tr
                        key={log.id}
                        className="border-b hover:bg-accent/50 transition-colors cursor-pointer"
                        onClick={() => handleLogClick(log)}
                      >
                        <td className="py-3 px-4">
                          <div className="font-medium">{log.model}</div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="text-sm">
                            <span className="text-muted-foreground">输入:</span> {formatNumber(log.prompt_tokens)}
                          </div>
                          <div className="text-sm">
                            <span className="text-muted-foreground">输出:</span> {formatNumber(log.completion_tokens)}
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="font-medium">{formatCurrency(Number(log.total_cost))}</div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="text-sm">{log.duration}ms</div>
                        </td>
                        <td className="py-3 px-4">
                          <div className={`flex items-center gap-1 ${statusInfo.color}`}>
                            <StatusIcon className="h-4 w-4" />
                            <span className="text-sm">{statusInfo.label}</span>
                          </div>
                        </td>
                        <td className="py-3 px-4">
                          <div className="text-sm text-muted-foreground">
                            {formatDate(log.created_at)}
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}

          {/* 分页 */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-6">
              <Button
                variant="outline"
                size="icon"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <span className="text-sm text-muted-foreground">
                第 {page} 页，共 {totalPages} 页
              </span>
              <Button
                variant="outline"
                size="icon"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 日志详情弹窗 */}
      {showDetailDialog && selectedLog && (
        <LogDetailDialog
          log={selectedLog}
          open={showDetailDialog}
          onClose={() => setShowDetailDialog(false)}
        />
      )}
    </div>
  )
}