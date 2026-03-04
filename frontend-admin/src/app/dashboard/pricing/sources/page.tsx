'use client'

import Link from 'next/link'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card } from '@/components/ui/card'
import { Loading } from '@/components/common/loading'
import { EmptyState } from '@/components/common/empty-state'
import { useToast } from '@/components/ui/use-toast'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { getErrorMessage, topologyApi, type ModelProviderTopologyResponse } from '@/lib/api'
import { RefreshCw, Search, GitBranch } from 'lucide-react'

type ModelSourceSummaryRow = {
  model_id: string
  model_name: string
  total_sources: number
  available_sources: number
  circuit_open_sources: number
  unavailable_sources: number
}

function normalizeText(v: string): string {
  return v.trim().toLowerCase()
}

export default function PricingSourcesModelListPage() {
  const { toast } = useToast()

  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [keyword, setKeyword] = useState('')

  const [topology, setTopology] = useState<ModelProviderTopologyResponse | null>(null)

  const load = useCallback(async ({ silent }: { silent?: boolean } = {}) => {
    if (!silent) setRefreshing(true)
    try {
      const res = await topologyApi.modelProviders({
        metrics_window_seconds: 300,
        include_instances: false,
        include_pricing_rules: false,
      })
      setTopology(res.data || null)
    } catch (error) {
      toast({
        title: '加载模型源头统计失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [toast])

  useEffect(() => {
    load({ silent: true })
  }, [load])

  const rows = useMemo<ModelSourceSummaryRow[]>(() => {
    const result: ModelSourceSummaryRow[] = []
    for (const m of topology?.models ?? []) {
      const providers = (m.operations ?? []).flatMap((op) => op.providers ?? [])
      const total = providers.length
      const circuitOpen = providers.filter((p) => p.circuit_state === 'open').length
      const unavailable = providers.filter((p) => p.status !== 'active' && p.circuit_state !== 'open').length
      const available = providers.filter((p) => p.status === 'active' && p.circuit_state !== 'open').length
      result.push({
        model_id: m.id,
        model_name: m.name || m.id,
        total_sources: total,
        available_sources: available,
        circuit_open_sources: circuitOpen,
        unavailable_sources: unavailable,
      })
    }
    return result.sort((a, b) => a.model_id.localeCompare(b.model_id))
  }, [topology])

  const filteredRows = useMemo(() => {
    const k = normalizeText(keyword)
    if (!k) return rows
    return rows.filter((r) => normalizeText(`${r.model_id} ${r.model_name}`).includes(k))
  }, [rows, keyword])

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">源头定价（按模型）</h1>
          <p className="text-sm text-muted-foreground mt-1">
            模型列表汇总源头数量与状态，进入模型后配置源头价格与查看调用数据。
          </p>
        </div>
        <Button variant="outline" onClick={() => load()} disabled={loading || refreshing}>
          <RefreshCw className={cn('w-4 h-4 mr-2', (loading || refreshing) && 'animate-spin')} />
          刷新
        </Button>
      </div>

      <Card className="p-4">
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="搜索模型（ID/名称）"
              className="w-full pl-9 pr-3 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
          </div>
        </div>
      </Card>

      {loading ? (
        <Loading />
      ) : filteredRows.length === 0 ? (
        <Card className="p-6">
          <EmptyState
            icon={GitBranch}
            title="暂无数据"
            description="未找到模型源头关系数据，请先配置 model_providers（模型源头）。"
            action={{ label: '重新加载', onClick: () => load() }}
          />
        </Card>
      ) : (
        <Card className="p-4">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">模型</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">源头数</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">可用</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">熔断</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">不可用</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {filteredRows.map((r) => (
                  <tr key={r.model_id} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <div className="font-medium">{r.model_name}</div>
                      <div className="text-xs text-muted-foreground">{r.model_id}</div>
                    </td>
                    <td className="px-4 py-3">{r.total_sources}</td>
                    <td className="px-4 py-3">
                      <span className="font-medium text-green-600">{r.available_sources}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="font-medium text-red-600">{r.circuit_open_sources}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="font-medium text-gray-600">{r.unavailable_sources}</span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Link
                        href={`/dashboard/pricing/sources/${encodeURIComponent(r.model_id)}`}
                        className="inline-flex items-center gap-2 px-3 py-1.5 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
                      >
                        配置
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  )
}
