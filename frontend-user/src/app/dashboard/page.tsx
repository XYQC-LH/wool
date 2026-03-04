'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useToast } from '@/components/ui/use-toast'
import { useAuthStore } from '@/store/auth'
import { getGatewayPublicBaseUrl } from '@/lib/api'
import { formatCurrency, formatNumber } from '@/lib/utils'
import { useUsageStats } from '@/lib/query'
import {
  Activity,
  Zap,
  DollarSign,
  TrendingUp,
  Copy,
  ExternalLink,
  Loader2,
} from 'lucide-react'
import Link from 'next/link'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'
import { Announcements } from '@/components/announcements'

export default function DashboardPage() {
  const { toast } = useToast()
  const { isAuthenticated } = useAuthStore()
  const gatewayBaseUrl = getGatewayPublicBaseUrl()

  // 使用 TanStack Query 获取统计数据（自动缓存）
  const { data: stats, isLoading, error } = useUsageStats()

  // 错误处理
  if (error) {
    console.error('加载统计数据失败:', error)
  }

  const copyApiEndpoint = async () => {
    try {
      await navigator.clipboard.writeText(gatewayBaseUrl)
      toast({ title: '已复制', description: '网关地址已复制到剪贴板' })
    } catch (error) {
      toast({ 
        title: '复制失败', 
        description: '请手动复制',
        variant: 'destructive'
      })
    }
  }

  if (!isAuthenticated) {
    return (
      <div className="space-y-6 animate-fade-in">
        <div>
          <h1 className="text-2xl font-bold">仪表盘</h1>
          <p className="text-muted-foreground">查看您的使用统计与快速开始</p>
        </div>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center py-12">
              <Activity className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">需要登录</h3>
              <p className="mt-2 text-sm text-muted-foreground">请先登录以查看您的统计数据和仪表盘信息</p>
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
      <Announcements />

      {/* 统计卡片 */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="card-hover">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总请求数</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {isLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                formatNumber(stats?.total_requests || 0)
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              本月 API 调用次数
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
              {isLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                formatNumber(stats?.total_tokens || 0)
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              本月 Token 使用量
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
              {isLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                formatCurrency(stats?.total_cost || 0)
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              本月消费总额
            </p>
          </CardContent>
        </Card>

        <Card className="card-hover">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">平均响应</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {isLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : stats?.avg_latency_ms !== undefined ? (
                `${Math.round(stats.avg_latency_ms)}ms`
              ) : (
                '—'
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              平均响应时间
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 使用趋势图表 */}
      <Card>
        <CardHeader>
          <CardTitle>使用趋势</CardTitle>
          <CardDescription>过去 30 天的 API 调用趋势</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-[300px]">
            {isLoading ? (
              <div className="flex items-center justify-center h-full">
                <div className="animate-pulse-slow text-muted-foreground">
                  加载中...
                </div>
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
                  <YAxis
                    stroke="hsl(var(--muted-foreground))"
                    fontSize={12}
                  />
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

      {/* 快速开始 */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>快速开始</CardTitle>
            <CardDescription>开始使用 Nexus API</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">API 端点</label>
              <div className="flex items-center gap-2">
                <code className="flex-1 rounded-md bg-muted px-3 py-2 text-sm">
                  {gatewayBaseUrl}
                </code>
                <Button size="icon" variant="outline" onClick={copyApiEndpoint}>
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">示例代码</label>
              <pre className="rounded-md bg-muted p-3 text-sm overflow-x-auto">
{`curl ${gatewayBaseUrl}/chat/completions \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`}
              </pre>
            </div>
            <div className="flex gap-2">
              <Link href="/dashboard/tokens">
                <Button>创建 API Key</Button>
              </Link>
              <Link href="/dashboard/docs">
                <Button variant="outline">
                  <ExternalLink className="mr-2 h-4 w-4" />
                  查看文档
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>模型使用分布</CardTitle>
            <CardDescription>各模型的调用占比</CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-3">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="animate-pulse">
                    <div className="h-4 bg-muted rounded w-full mb-2" />
                    <div className="h-2 bg-muted rounded w-3/4" />
                  </div>
                ))}
              </div>
            ) : (
              <div className="space-y-4">
                {(stats?.model_stats || []).slice(0, 5).map((model) => (
                  <div key={model.model} className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium">{model.model}</span>
                      <span className="text-muted-foreground">
                        {formatNumber(model.requests)} 次
                      </span>
                    </div>
                    <div className="h-2 rounded-full bg-muted overflow-hidden">
                      <div
                        className="h-full bg-primary rounded-full transition-all"
                        style={{
                          width: `${Math.min(
                            (model.requests / (stats?.total_requests || 1)) * 100,
                            100
                          )}%`,
                        }}
                      />
                    </div>
                  </div>
                ))}
                {(!stats?.model_stats || stats.model_stats.length === 0) && (
                  <div className="text-center text-muted-foreground py-8">
                    暂无使用数据
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
