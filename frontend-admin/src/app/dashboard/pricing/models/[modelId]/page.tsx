'use client'

import Link from 'next/link'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Loading } from '@/components/common/loading'
import { EmptyState } from '@/components/common/empty-state'
import { useToast } from '@/components/ui/use-toast'
import { cn, formatDate } from '@/lib/utils'
import {
  CreateModelCapabilityRequest,
  UpdateModelCapabilityRequest,
  UpdateModelRequest,
  getErrorMessage,
  modelApi,
  modelCapabilityApi,
  type Model,
  type ModelCapability,
} from '@/lib/api'
import { OPERATION_OPTIONS } from '@/lib/operations'
import { ArrowLeft, RefreshCw, Save, SlidersHorizontal } from 'lucide-react'

function toNumber(value: string): number | null {
  const num = Number(value)
  if (!Number.isFinite(num)) return null
  return num
}

function formatMoney(value: unknown): string {
  const num = Number(value)
  if (!Number.isFinite(num)) return String(value ?? '')
  return num.toFixed(6)
}

export default function PricingModelDetailPage() {
  const { toast } = useToast()
  const router = useRouter()
  const routeParams = useParams()
  const rawModelId = Array.isArray(routeParams.modelId) ? routeParams.modelId[0] : routeParams.modelId
  const modelId = useMemo(() => decodeURIComponent(rawModelId || ''), [rawModelId])

  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [togglingModel, setTogglingModel] = useState(false)
  const [togglingOp, setTogglingOp] = useState<string | null>(null)

  const [model, setModel] = useState<Model | null>(null)
  const [capabilities, setCapabilities] = useState<ModelCapability[]>([])

  const [form, setForm] = useState({
    input_price: '0',
    output_price: '0',
  })

  const capabilityMap = useMemo(() => {
    const map = new Map<string, ModelCapability>()
    for (const c of capabilities) {
      if (c?.operation) map.set(c.operation, c)
    }
    return map
  }, [capabilities])

  const operations = useMemo(() => {
    const set = new Set<string>(OPERATION_OPTIONS.map((o) => o.value))
    for (const c of capabilities) {
      if (c?.operation) set.add(c.operation)
    }
    return Array.from(set).sort((a, b) => a.localeCompare(b))
  }, [capabilities])

  const load = useCallback(async ({ silent }: { silent?: boolean } = {}) => {
    if (!silent) setRefreshing(true)
    try {
      if (!modelId) {
        setModel(null)
        setCapabilities([])
        return
      }
      const [mRes, cRes] = await Promise.all([
        modelApi.get(modelId),
        modelCapabilityApi.list({ model_id: modelId, page: 1, page_size: 200 }),
      ])

      const m = mRes.data || null
      setModel(m)
      setCapabilities(cRes.data?.list || [])
      if (m) {
        setForm({
          input_price: String(m.input_price ?? '0'),
          output_price: String(m.output_price ?? '0'),
        })
      }
    } catch (error) {
      toast({
        title: '加载模型定价失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [modelId, toast])

  useEffect(() => {
    load({ silent: true })
  }, [load])

  const savePricing = useCallback(async () => {
    if (!modelId) return

    const input = toNumber(form.input_price)
    const output = toNumber(form.output_price)
    if (input === null || input < 0) {
      toast({ title: 'input_price 必须是非负数', variant: 'destructive' })
      return
    }
    if (output === null || output < 0) {
      toast({ title: 'output_price 必须是非负数', variant: 'destructive' })
      return
    }

    setSaving(true)
    try {
      const payload: UpdateModelRequest = {
        input_price: input,
        output_price: output,
      }
      await modelApi.update(modelId, payload)
      toast({ title: '已保存模型价格' })
      await load({ silent: true })
    } catch (error) {
      toast({
        title: '保存失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setSaving(false)
    }
  }, [form.input_price, form.output_price, load, modelId, toast])

  const toggleModelEnabled = useCallback(async (enabled: boolean) => {
    if (!model) return
    setTogglingModel(true)
    try {
      await modelApi.updateStatus(model.id, enabled)
      toast({ title: enabled ? '已启用模型' : '已禁用模型' })
      await load({ silent: true })
    } catch (error) {
      toast({
        title: '更新失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setTogglingModel(false)
    }
  }, [load, model, toast])

  const setOperationEnabled = useCallback(async (operation: string, enabled: boolean) => {
    if (!modelId || !operation) return

    setTogglingOp(operation)
    try {
      const existing = capabilityMap.get(operation)
      if (existing) {
        const payload: UpdateModelCapabilityRequest = { enabled }
        await modelCapabilityApi.update(existing.id, payload)
      } else {
        if (enabled) {
          toast({ title: '该 operation 默认允许（未配置能力记录）' })
          return
        }
        const payload: CreateModelCapabilityRequest = { model_id: modelId, operation, enabled }
        await modelCapabilityApi.create(payload)
      }

      toast({ title: enabled ? '已启用 operation' : '已禁用 operation' })
      await load({ silent: true })
    } catch (error) {
      toast({
        title: '更新失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setTogglingOp(null)
    }
  }, [capabilityMap, load, modelId, toast])

  const operationEnabled = useCallback((operation: string): boolean => {
    const cap = capabilityMap.get(operation)
    if (!cap) return true
    return Boolean(cap.enabled)
  }, [capabilityMap])

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3">
        <Link href="/dashboard/pricing/models" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="w-4 h-4" />
          返回模型列表
        </Link>

        <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold">模型定价 · {model?.display_name || model?.name || modelId}</h1>
            <p className="text-sm text-muted-foreground mt-1">
              配置模型基础价格（input/output），并管理该模型支持的 operation（开关）。
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Button variant="outline" onClick={() => load()} disabled={loading || refreshing}>
              <RefreshCw className={cn('w-4 h-4 mr-2', (loading || refreshing) && 'animate-spin')} />
              刷新
            </Button>
          </div>
        </div>
      </div>

      {loading ? (
        <Loading />
      ) : !model ? (
        <Card className="p-6">
          <EmptyState
            icon={SlidersHorizontal}
            title="未找到模型"
            description="请确认该模型已存在，或返回列表重新选择。"
            action={{ label: '返回列表', onClick: () => router.push('/dashboard/pricing/models') }}
          />
        </Card>
      ) : (
        <>
          <Card className="p-6 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-muted-foreground">模型 ID</div>
                <div className="font-mono mt-1">{model.id}</div>
              </div>
              <div className="flex items-center gap-3">
                <Switch
                  checked={Boolean(model.enabled)}
                  onCheckedChange={(checked) => toggleModelEnabled(checked)}
                  disabled={togglingModel}
                />
                <span className="text-sm text-muted-foreground">{model.enabled ? '已启用' : '已禁用'}</span>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2">Input Price</label>
                <input
                  type="number"
                  min="0"
                  step="0.000001"
                  value={form.input_price}
                  onChange={(e) => setForm((prev) => ({ ...prev, input_price: e.target.value }))}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
                <div className="text-xs text-muted-foreground mt-1">
                  当前：<span className="font-mono">{formatMoney(model.input_price)}</span>
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Output Price</label>
                <input
                  type="number"
                  min="0"
                  step="0.000001"
                  value={form.output_price}
                  onChange={(e) => setForm((prev) => ({ ...prev, output_price: e.target.value }))}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
                <div className="text-xs text-muted-foreground mt-1">
                  当前：<span className="font-mono">{formatMoney(model.output_price)}</span>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end gap-3">
              <Button variant="outline" onClick={() => load()} disabled={saving}>
                取消
              </Button>
              <Button onClick={savePricing} disabled={saving}>
                <Save className="w-4 h-4 mr-2" />
                {saving ? '保存中...' : '保存价格'}
              </Button>
            </div>
          </Card>

          <Card className="p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <SlidersHorizontal className="w-4 h-4 text-orange-600" />
                <h2 className="text-lg font-semibold">operation 开关</h2>
              </div>
              <Link href="/dashboard/model-capabilities" className="text-sm text-orange-600 hover:text-orange-700">
                去“模型能力”查看更多
              </Link>
            </div>

            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">operation</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">来源</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">状态</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">更新时间</th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-gray-500">开关</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {operations.map((op) => {
                    const cap = capabilityMap.get(op)
                    const enabled = operationEnabled(op)
                    return (
                      <tr key={op} className="hover:bg-gray-50">
                        <td className="px-4 py-3 font-mono">{op}</td>
                        <td className="px-4 py-3 text-muted-foreground">
                          {cap ? `已配置(#${cap.id})` : '默认允许(未配置)'}
                        </td>
                        <td className="px-4 py-3">
                          {enabled ? (
                            <span className="px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">启用</span>
                          ) : (
                            <span className="px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">禁用</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">
                          {cap?.updated_at ? formatDate(cap.updated_at) : cap?.created_at ? formatDate(cap.created_at) : '—'}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <Switch
                            checked={enabled}
                            disabled={togglingOp === op}
                            onCheckedChange={(checked) => setOperationEnabled(op, checked)}
                          />
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </Card>
        </>
      )}
    </div>
  )
}
