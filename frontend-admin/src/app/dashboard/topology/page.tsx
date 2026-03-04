'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
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
import { cn } from '@/lib/utils'
import {
  getErrorMessage,
  modelApi,
  topologyApi,
  type Model,
  type ModelProviderTopologyResponse,
  type TopologyProvider,
} from '@/lib/api'
import { Activity, GitBranch, RefreshCw, Share2 } from 'lucide-react'

type FlatLinkRow = {
  model_id: string
  model_name: string
  operation: string
  provider_id: number
  channel_name: string
  upstream_model_name: string
  status: string
  circuit_state: string
  health_score: string | number
  request_count?: number
  success_rate?: number
  avg_latency_ms?: number
  instance_count: number
}

function ProviderStateBadge({ status, circuitState }: { status: string; circuitState: string }) {
  if (circuitState === 'open') {
    return (
      <span className="px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800">
        熔断
      </span>
    )
  }
  if (circuitState === 'half_open') {
    return (
      <span className="px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
        半开
      </span>
    )
  }
  if (status === 'active') {
    return (
      <span className="px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
        健康
      </span>
    )
  }
  return (
    <span className="px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
      禁用
    </span>
  )
}

function InstanceStatusBadge({ status }: { status: string }) {
  if (status === 'active') {
    return (
      <span className="px-2 py-0.5 rounded-full text-xs bg-green-50 text-green-700 border border-green-200">
        active
      </span>
    )
  }
  if (status === 'cooling') {
    return (
      <span className="px-2 py-0.5 rounded-full text-xs bg-yellow-50 text-yellow-700 border border-yellow-200">
        cooling
      </span>
    )
  }
  return (
    <span className="px-2 py-0.5 rounded-full text-xs bg-gray-50 text-gray-700 border border-gray-200">
      disabled
    </span>
  )
}

function ProviderCard({ provider }: { provider: TopologyProvider }) {
  const metrics = provider.metrics
  const healthScore = Number(provider.health_score)
  const healthColor =
    healthScore >= 80 ? 'text-green-600' : healthScore >= 50 ? 'text-yellow-600' : 'text-red-600'

  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-center gap-2 min-w-0">
            <div className="font-medium truncate">
              {provider.channel_name || '（未命名渠道）'}
            </div>
            <span className="text-xs text-muted-foreground">#{provider.id}</span>
          </div>
          <div className="mt-1 text-sm text-muted-foreground truncate">
            上游模型：{provider.upstream_model_name || '-'}
          </div>
        </div>
        <div className="flex flex-col items-end gap-1 flex-shrink-0">
          <ProviderStateBadge status={provider.status} circuitState={provider.circuit_state} />
          <div className={cn('text-sm font-medium', healthColor)}>
            健康 {Number.isFinite(healthScore) ? healthScore.toFixed(1) : '-'}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3 mt-4">
        <div className="space-y-1">
          <div className="text-xs text-muted-foreground">请求数</div>
          <div className="text-sm font-medium">
            {metrics ? Number(metrics.request_count).toLocaleString() : '-'}
          </div>
        </div>
        <div className="space-y-1">
          <div className="text-xs text-muted-foreground">成功率</div>
          <div className="text-sm font-medium">
            {metrics ? `${Number(metrics.success_rate ?? 0).toFixed(1)}%` : '-'}
          </div>
        </div>
        <div className="space-y-1">
          <div className="text-xs text-muted-foreground">平均延迟</div>
          <div className="text-sm font-medium">
            {metrics ? `${Math.round(Number(metrics.avg_latency_ms ?? 0))}ms` : '-'}
          </div>
        </div>
      </div>

      <div className="mt-4">
        <div className="flex items-center justify-between mb-2">
          <div className="text-sm font-medium">实例</div>
          <div className="text-xs text-muted-foreground">{provider.instances.length} 个</div>
        </div>
        {provider.instances.length === 0 ? (
          <div className="text-sm text-muted-foreground">暂无实例</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">名称</th>
                  <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">类型</th>
                  <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">状态</th>
                  <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">权重</th>
                  <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">并发</th>
                  <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">RPM/TPM</th>
                  <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">资源账户</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {provider.instances.map((inst) => (
                  <tr key={inst.id} className="hover:bg-gray-50">
                    <td className="px-2 py-2 font-medium whitespace-nowrap">{inst.name}</td>
                    <td className="px-2 py-2 text-muted-foreground whitespace-nowrap">{inst.instance_type}</td>
                    <td className="px-2 py-2 whitespace-nowrap">
                      <InstanceStatusBadge status={inst.status} />
                    </td>
                    <td className="px-2 py-2 whitespace-nowrap">{inst.weight}</td>
                    <td className="px-2 py-2 whitespace-nowrap">{inst.max_concurrency || '-'}</td>
                    <td className="px-2 py-2 whitespace-nowrap">
                      {(inst.rpm_limit || '-') + '/' + (inst.tpm_limit || '-')}
                    </td>
                    <td className="px-2 py-2 text-muted-foreground whitespace-nowrap">
                      {inst.resource_account_name || '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

export default function TopologyPage() {
  const { toast } = useToast()

  const inFlightRef = useRef(false)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const [models, setModels] = useState<Model[]>([])
  const [selectedModelId, setSelectedModelId] = useState<string>('all')
  const [selectedOperation, setSelectedOperation] = useState<string>('all')
  const [metricsWindowSeconds, setMetricsWindowSeconds] = useState<number>(300)
  const [autoRefresh, setAutoRefresh] = useState<boolean>(true)

  const [topology, setTopology] = useState<ModelProviderTopologyResponse | null>(null)

  const operationOptions = useMemo(() => {
    const known = [
      'chat.completions',
      'embeddings',
      'images.generations',
      'videos.generations',
      'audio.transcriptions',
      'audio.speech',
      'audio.translations',
      'completions',
    ]
    const fromData = new Set<string>()
    for (const m of topology?.models ?? []) {
      for (const op of m.operations ?? []) {
        if (op?.operation) fromData.add(op.operation)
      }
    }
    return Array.from(new Set([...known, ...Array.from(fromData)])).sort()
  }, [topology])

  const flatLinks = useMemo<FlatLinkRow[]>(() => {
    if (!topology) return []
    const rows: FlatLinkRow[] = []
    for (const m of topology.models ?? []) {
      for (const op of m.operations ?? []) {
        for (const p of op.providers ?? []) {
          rows.push({
            model_id: m.id,
            model_name: m.name,
            operation: op.operation,
            provider_id: p.id,
            channel_name: p.channel_name,
            upstream_model_name: p.upstream_model_name,
            status: p.status,
            circuit_state: p.circuit_state,
            health_score: p.health_score,
            request_count: p.metrics?.request_count,
            success_rate: p.metrics?.success_rate,
            avg_latency_ms: p.metrics?.avg_latency_ms,
            instance_count: p.instances.length,
          })
        }
      }
    }
    return rows
  }, [topology])

  const fetchModels = useCallback(async () => {
    try {
      const res = await modelApi.list({ page: 1, page_size: 200 })
      setModels(res.data?.list ?? [])
    } catch (err) {
      toast({
        title: '加载模型列表失败',
        description: getErrorMessage(err),
        variant: 'destructive',
      })
    }
  }, [toast])

  const fetchTopology = useCallback(
    async ({ silent }: { silent?: boolean } = {}) => {
      if (inFlightRef.current) return
      inFlightRef.current = true

      const shouldSilent = Boolean(silent)
      if (!shouldSilent && topology) setRefreshing(true)

      try {
        const params: { model_id?: string; operation?: string; metrics_window_seconds?: number } = {
          metrics_window_seconds: metricsWindowSeconds,
        }
        if (selectedModelId !== 'all') params.model_id = selectedModelId
        if (selectedOperation !== 'all') params.operation = selectedOperation

        const res = await topologyApi.modelProviders(params)
        setTopology(res.data)
      } catch (err) {
        toast({
          title: '加载拓扑失败',
          description: getErrorMessage(err),
          variant: 'destructive',
        })
      } finally {
        inFlightRef.current = false
        setLoading(false)
        setRefreshing(false)
      }
    },
    [metricsWindowSeconds, selectedModelId, selectedOperation, toast, topology]
  )

  useEffect(() => {
    fetchModels()
  }, [fetchModels])

  useEffect(() => {
    fetchTopology({ silent: true })
  }, [fetchTopology])

  useEffect(() => {
    if (!autoRefresh) return
    const id = window.setInterval(() => {
      fetchTopology({ silent: true })
    }, 5000)
    return () => window.clearInterval(id)
  }, [autoRefresh, fetchTopology])

  const lastUpdated = useMemo(() => {
    if (!topology?.generated_at) return '-'
    const d = new Date(topology.generated_at)
    if (Number.isNaN(d.getTime())) return topology.generated_at
    return d.toLocaleString()
  }, [topology?.generated_at])

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">模型-源头链路拓扑</h1>
          <p className="text-sm text-muted-foreground mt-1">更新时间：{lastUpdated}</p>
        </div>
        <Button
          variant="outline"
          onClick={() => fetchTopology()}
          disabled={refreshing || loading}
        >
          <RefreshCw className={cn('w-4 h-4 mr-2', (refreshing || loading) && 'animate-spin')} />
          刷新
        </Button>
      </div>

      <Card className="p-4">
        <div className="flex flex-col lg:flex-row lg:items-end gap-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 flex-1">
            <div className="space-y-2">
              <Label>模型</Label>
              <Select value={selectedModelId} onValueChange={setSelectedModelId}>
                <SelectTrigger>
                  <SelectValue placeholder="选择模型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  {models.map((m) => (
                    <SelectItem key={m.id} value={m.id}>
                      {m.display_name || m.name || m.id}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Operation</Label>
              <Select value={selectedOperation} onValueChange={setSelectedOperation}>
                <SelectTrigger>
                  <SelectValue placeholder="选择 Operation" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  {operationOptions.map((op) => (
                    <SelectItem key={op} value={op}>
                      {op}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>指标窗口</Label>
              <Select
                value={String(metricsWindowSeconds)}
                onValueChange={(v) => setMetricsWindowSeconds(Number(v))}
              >
                <SelectTrigger>
                  <SelectValue placeholder="窗口" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="60">60s</SelectItem>
                  <SelectItem value="300">300s</SelectItem>
                  <SelectItem value="900">900s</SelectItem>
                  <SelectItem value="1800">1800s</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>刷新</Label>
              <div className="flex items-center justify-between rounded-md border px-3 py-2">
                <Label htmlFor="auto-refresh" className="text-sm text-muted-foreground cursor-pointer">
                  自动刷新（5s）
                </Label>
                <Switch
                  id="auto-refresh"
                  checked={autoRefresh}
                  onCheckedChange={setAutoRefresh}
                />
              </div>
            </div>
          </div>
        </div>
      </Card>

      {loading ? (
        <Loading />
      ) : !topology || flatLinks.length === 0 ? (
        <Card className="p-6">
          <EmptyState
            icon={Share2}
            title="暂无链路数据"
            description="请先在“模型源头（ProviderGroup）/实例（ProviderInstance）”完成配置，或调整筛选条件。"
            action={{
              label: '重新加载',
              onClick: () => fetchTopology(),
            }}
          />
        </Card>
      ) : (
        <>
          <Card className="p-4">
            <div className="flex items-center gap-2 mb-3">
              <Share2 className="w-4 h-4 text-orange-600" />
              <h2 className="text-lg font-semibold">模型层 → 源头层 关系清单</h2>
              <span className="text-sm text-muted-foreground">（{flatLinks.length} 条）</span>
            </div>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">模型</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">Operation</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">源头组</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">渠道</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">上游模型</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">状态</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">健康</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">请求</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">成功率</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">延迟</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">实例</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {flatLinks.map((row) => (
                    <tr key={`${row.provider_id}`} className="hover:bg-gray-50">
                      <td className="px-3 py-2 whitespace-nowrap">
                        <div className="font-medium">{row.model_name || row.model_id}</div>
                        <div className="text-xs text-muted-foreground">{row.model_id}</div>
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">{row.operation}</td>
                      <td className="px-3 py-2 whitespace-nowrap">#{row.provider_id}</td>
                      <td className="px-3 py-2 whitespace-nowrap">{row.channel_name}</td>
                      <td className="px-3 py-2 whitespace-nowrap">{row.upstream_model_name || '-'}</td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        <ProviderStateBadge status={row.status} circuitState={row.circuit_state} />
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        {Number.isFinite(Number(row.health_score))
                          ? Number(row.health_score).toFixed(1)
                          : '-'}
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        {typeof row.request_count === 'number' ? row.request_count.toLocaleString() : '-'}
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        {typeof row.success_rate === 'number' ? `${row.success_rate.toFixed(1)}%` : '-'}
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        {typeof row.avg_latency_ms === 'number' ? `${Math.round(row.avg_latency_ms)}ms` : '-'}
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">{row.instance_count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>

          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <GitBranch className="w-4 h-4 text-orange-600" />
              <h2 className="text-lg font-semibold">实时拓扑视图</h2>
            </div>

            <div className="space-y-4">
              {topology.models.map((m) => (
                <Card key={m.id} className="p-4">
                  <div className="flex items-center gap-2">
                    <div className="p-2 bg-orange-100 rounded-full">
                      <GitBranch className="w-4 h-4 text-orange-600" />
                    </div>
                    <div className="min-w-0">
                      <div className="font-semibold truncate">{m.name || m.id}</div>
                      <div className="text-sm text-muted-foreground truncate">{m.id}</div>
                    </div>
                  </div>

                  <div className="mt-4 space-y-5">
                    {m.operations.map((op) => (
                      <div key={op.operation} className="border-l-2 border-orange-200 pl-4 ml-2">
                        <div className="flex items-center gap-2 mb-3">
                          <div className="p-2 bg-orange-50 rounded-full">
                            <Activity className="w-4 h-4 text-orange-600" />
                          </div>
                          <div className="font-medium">{op.operation}</div>
                          <div className="text-sm text-muted-foreground">
                            {op.providers.length} 个源头组
                          </div>
                        </div>

                        <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
                          {op.providers.map((p) => (
                            <ProviderCard key={p.id} provider={p} />
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  )
}

