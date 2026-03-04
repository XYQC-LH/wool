'use client'

import Link from 'next/link'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Loading } from '@/components/common/loading'
import { EmptyState } from '@/components/common/empty-state'
import { useToast } from '@/components/ui/use-toast'
import { cn } from '@/lib/utils'
import { getErrorMessage, modelApi, type Model } from '@/lib/api'
import { CreditCard, RefreshCw, Search } from 'lucide-react'

function normalizeText(v: string): string {
  return v.trim().toLowerCase()
}

function formatMoney(value: unknown): string {
  const num = Number(value)
  if (!Number.isFinite(num)) return String(value ?? '')
  return num.toFixed(6)
}

export default function PricingModelsListPage() {
  const { toast } = useToast()

  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [keyword, setKeyword] = useState('')

  const [models, setModels] = useState<Model[]>([])

  const load = useCallback(async ({ silent }: { silent?: boolean } = {}) => {
    if (!silent) setRefreshing(true)
    try {
      const res = await modelApi.list({ page: 1, page_size: 200 })
      setModels(res.data?.list ?? [])
    } catch (error) {
      toast({
        title: '加载模型失败',
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

  const filtered = useMemo(() => {
    const k = normalizeText(keyword)
    if (!k) return models
    return models.filter((m) =>
      normalizeText(`${m.id} ${m.name} ${m.display_name}`).includes(k)
    )
  }, [models, keyword])

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">模型定价</h1>
          <p className="text-sm text-muted-foreground mt-1">
            配置模型基础价格与可用功能（operation 开关）。
          </p>
        </div>
        <Button variant="outline" onClick={() => load()} disabled={loading || refreshing}>
          <RefreshCw className={cn('w-4 h-4 mr-2', (loading || refreshing) && 'animate-spin')} />
          刷新
        </Button>
      </div>

      <Card className="p-4">
        <div className="relative">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索模型（ID/名称）"
            className="w-full pl-9 pr-3 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
          />
        </div>
      </Card>

      {loading ? (
        <Loading />
      ) : filtered.length === 0 ? (
        <Card className="p-6">
          <EmptyState
            icon={CreditCard}
            title="暂无模型"
            description="请先在“模型管理”创建模型。"
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
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">Input Price</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">Output Price</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">状态</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {filtered.map((m) => (
                  <tr key={m.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <div className="font-medium">{m.display_name || m.name || m.id}</div>
                      <div className="text-xs text-muted-foreground">{m.id}</div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">{formatMoney(m.input_price)}</td>
                    <td className="px-4 py-3 whitespace-nowrap">{formatMoney(m.output_price)}</td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      {m.enabled ? (
                        <span className="px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                          enabled
                        </span>
                      ) : (
                        <span className="px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                          disabled
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Link
                        href={`/dashboard/pricing/models/${encodeURIComponent(m.id)}`}
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

