'use client'

import { useCallback, useEffect, useState } from 'react'
import { channelApi, Channel, ChannelStat, ChannelTestResult, logApi, getErrorMessage } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useToast } from '@/components/ui/use-toast'
import { formatCurrency, formatDate, formatNumber, getStatusColor, getStatusText } from '@/lib/utils'
import {
  ArrowLeft,
  Server,
  Activity,
  Zap,
  Clock,
  RefreshCw,
  Edit,
  TestTube,
} from 'lucide-react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'

type StatsRange = 'today' | '7d' | '30d'

function getDateKey(date: Date) {
  return date.toISOString().split('T')[0] // YYYY-MM-DD (UTC)
}

function buildRange(range: StatsRange) {
  const end = new Date()
  const start = new Date()

  switch (range) {
    case 'today':
      return { start_date: getDateKey(end), end_date: getDateKey(end) }
    case '7d':
      start.setDate(start.getDate() - 7)
      break
    case '30d':
      start.setDate(start.getDate() - 30)
      break
  }

  return { start_date: getDateKey(start), end_date: getDateKey(end) }
}

export default function ChannelDetailPage() {
  const { toast } = useToast()
  const router = useRouter()
  const params = useParams()
  const id = Array.isArray(params.id) ? params.id[0] : params.id
  const [channel, setChannel] = useState<Channel | null>(null)
  const [loading, setLoading] = useState(true)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<ChannelTestResult | null>(null)
  const [statsRange, setStatsRange] = useState<StatsRange>('7d')
  const [statsLoading, setStatsLoading] = useState(false)
  const [channelStats, setChannelStats] = useState<ChannelStat | null>(null)

  const loadChannel = useCallback(async (channelID: number) => {
    setLoading(true)
    try {
      const res = await channelApi.get(channelID)
      if (res.data) {
        setChannel(res.data)
      }
    } catch (error) {
      toast({
        title: '加载渠道失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [toast])

  const loadChannelStats = useCallback(async (channelID: number) => {
    setStatsLoading(true)
    try {
      const range = buildRange(statsRange)
      const res = await logApi.stats(range)
      const list = Array.isArray(res.data?.channel_stats) ? res.data.channel_stats : []
      const stat = list.find((s) => s.channel_id === channelID) || null
      setChannelStats(stat)
    } catch (error) {
      toast({
        title: '加载统计失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
      setChannelStats(null)
    } finally {
      setStatsLoading(false)
    }
  }, [statsRange, toast])

  useEffect(() => {
    if (!id) return

    const channelID = Number(id)
    if (!Number.isFinite(channelID) || channelID <= 0) {
      router.replace('/dashboard/channels')
      return
    }

    loadChannel(channelID)
  }, [id, loadChannel, router])

  useEffect(() => {
    if (!id) return
    const channelID = Number(id)
    if (!Number.isFinite(channelID)) return
    loadChannelStats(channelID)
  }, [id, loadChannelStats])

  const handleTest = async () => {
    if (!id) return
    setTesting(true)
    setTestResult(null)
    try {
      const res = await channelApi.test(parseInt(id))
      if (res.data) {
        setTestResult(res.data)
      }
    } catch (error) {
      toast({
        title: '测试失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
      setTestResult({ success: false, status: 'error', latency: 0, message: '测试失败' })
    } finally {
      setTesting(false)
    }
  }

  const handleRefresh = async () => {
    if (!id) return
    const channelID = Number(id)
    if (!Number.isFinite(channelID)) return
    await Promise.all([loadChannel(channelID), loadChannelStats(channelID)])
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-orange-500"></div>
      </div>
    )
  }

  if (!channel) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">渠道不存在</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/dashboard/channels">
            <Button variant="ghost" size="icon">
              <ArrowLeft className="w-4 h-4" />
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-bold">{channel.name}</h1>
            <p className="text-muted-foreground">渠道详情和配置</p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleRefresh}>
            <RefreshCw className="w-4 h-4 mr-2" />
            刷新
          </Button>
          <Button variant="outline" onClick={handleTest} disabled={testing}>
            <TestTube className="w-4 h-4 mr-2" />
            {testing ? '测试中...' : '测试连接'}
          </Button>
          <Button onClick={() => router.push(`/dashboard/channels?edit=${encodeURIComponent(String(id || ''))}`)}>
            <Edit className="w-4 h-4 mr-2" />
            编辑
          </Button>
        </div>
      </div>

      {/* Channel Info */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="w-5 h-5" />
              基本信息
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm text-muted-foreground">渠道类型</p>
              <p className="font-medium">
                {channel.type === 'official' ? '官方 API' : channel.type === 'reverse_engineered' ? '逆向接口' : channel.type === 'proxy' ? '代理' : channel.type}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">状态</p>
              <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(channel.status)}`}>
                {getStatusText(channel.status)}
              </span>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Base URL</p>
              <p className="font-mono text-sm">{channel.base_url}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">创建时间</p>
              <p className="text-sm">{formatDate(channel.created_at)}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">最近测试</p>
              <p className="text-sm">
                {channel.last_test_at ? formatDate(channel.last_test_at) : '暂无'}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="flex items-center gap-2">
              <Activity className="w-5 h-5" />
              性能指标
            </CardTitle>
            <div className="flex items-center gap-2">
              <select
                value={statsRange}
                onChange={(e) => setStatsRange(e.target.value as StatsRange)}
                className="h-9 px-3 rounded-lg border border-border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-orange-500"
              >
                <option value="today">今天</option>
                <option value="7d">最近7天</option>
                <option value="30d">最近30天</option>
              </select>
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  const channelID = Number(id)
                  if (Number.isFinite(channelID)) {
                    loadChannelStats(channelID)
                  }
                }}
                disabled={statsLoading}
              >
                <RefreshCw className="w-4 h-4" />
                {statsLoading ? '刷新中' : '刷新'}
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {statsLoading ? (
              <div className="flex items-center justify-center py-6 text-sm text-muted-foreground">
                加载中...
              </div>
            ) : channelStats ? (
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">请求数</p>
                  <p className="text-xl font-bold">{formatNumber(channelStats.request_count)}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">成功率</p>
                  <p className="text-xl font-bold">{Number(channelStats.success_rate || 0).toFixed(1)}%</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">平均延迟</p>
                  <p className="text-xl font-bold">{Math.round(Number(channelStats.avg_latency || 0))}ms</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">失败数</p>
                  <p className="text-xl font-bold text-red-500">{formatNumber(channelStats.fail_count)}</p>
                </div>
                <div className="col-span-2">
                  <p className="text-sm text-muted-foreground">上游成本</p>
                  <p className="text-xl font-bold">{formatCurrency(Number(channelStats.total_cost || 0))}</p>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center py-6 text-sm text-muted-foreground">
                当前统计周期暂无数据
              </div>
            )}

            <div className="pt-4 border-t space-y-2">
              {testResult ? (
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <TestTube className="w-4 h-4" />
                    <span>最近测试</span>
                  </div>
                  <span className={`text-sm font-medium ${testResult.status === 'success' ? 'text-green-500' : 'text-red-500'}`}>
                    {testResult.status === 'success' ? `${testResult.latency}ms` : '测试失败'}
                  </span>
                </div>
              ) : null}

              <div className="grid grid-cols-2 gap-2 text-sm">
                <div className="flex items-center gap-2">
                  <Zap className="w-4 h-4 text-muted-foreground" />
                  <span className="text-muted-foreground">权重:</span>
                  <span>{channel.weight}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Zap className="w-4 h-4 text-muted-foreground" />
                  <span className="text-muted-foreground">优先级:</span>
                  <span>{channel.priority}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Clock className="w-4 h-4 text-muted-foreground" />
                  <span className="text-muted-foreground">超时:</span>
                  <span>{Math.max(1, Math.round((channel.timeout_ms ?? 30000) / 1000))}s</span>
                </div>
                <div className="flex items-center gap-2">
                  <Clock className="w-4 h-4 text-muted-foreground" />
                  <span className="text-muted-foreground">重试:</span>
                  <span>{channel.retry_count ?? 0}次</span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Supported Models */}
      <Card>
        <CardHeader>
          <CardTitle>支持的模型</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {channel.models.map((model) => (
              <span key={model} className="px-3 py-1 bg-muted rounded-md text-sm">
                {model}
              </span>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
