'use client'

import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Loading } from '@/components/common/loading'
import { EmptyState } from '@/components/common/empty-state'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { Modal } from '@/components/ui/modal'
import { Switch } from '@/components/ui/switch'
import { useToast } from '@/components/ui/use-toast'
import { cn, formatDate } from '@/lib/utils'
import {
  CreateProviderPricingRuleRequest,
  UpdateProviderPricingRuleRequest,
  getErrorMessage,
  providerPricingRuleApi,
  topologyApi,
  type ModelProviderTopologyResponse,
  type TopologyProvider,
  type TopologyProviderPricingRule,
} from '@/lib/api'
import { OPERATION_OPTIONS } from '@/lib/operations'
import { ArrowLeft, CreditCard, Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react'

function formatDecimal(value: unknown, fractionDigits = 6): string {
  const num = Number(value)
  if (!Number.isFinite(num)) return String(value ?? '')
  return num.toFixed(fractionDigits)
}

function getLatestUpdatedAt(rules: TopologyProviderPricingRule[] | undefined): string | null {
  if (!rules || rules.length === 0) return null
  let latest = rules[0].updated_at
  for (const r of rules) {
    if (new Date(r.updated_at).getTime() > new Date(latest).getTime()) {
      latest = r.updated_at
    }
  }
  return latest
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; className: string }> = {
    active: { label: '可用', className: 'bg-green-100 text-green-800' },
    disabled: { label: '禁用', className: 'bg-gray-100 text-gray-800' },
    cooling: { label: '冷却', className: 'bg-yellow-100 text-yellow-800' },
  }
  const item = map[status] || { label: status || '-', className: 'bg-gray-100 text-gray-800' }
  return <span className={cn('px-2 py-1 rounded-full text-xs font-medium', item.className)}>{item.label}</span>
}

function CircuitBadge({ state }: { state: string }) {
  const map: Record<string, { label: string; className: string }> = {
    closed: { label: '正常', className: 'bg-green-100 text-green-800' },
    open: { label: '熔断', className: 'bg-red-100 text-red-800' },
    half_open: { label: '半开', className: 'bg-yellow-100 text-yellow-800' },
  }
  const item = map[state] || { label: state || '-', className: 'bg-gray-100 text-gray-800' }
  return <span className={cn('px-2 py-1 rounded-full text-xs font-medium', item.className)}>{item.label}</span>
}

type PricingRuleFormState = {
  operation: string
  unit: string
  cost_per_unit: string
  price_per_unit: string
  enabled: boolean
}

const DEFAULT_RULE_FORM: PricingRuleFormState = {
  operation: 'chat.completions',
  unit: 'request',
  cost_per_unit: '0',
  price_per_unit: '0',
  enabled: true,
}

export default function PricingSourcesModelDetailPage() {
  const { toast } = useToast()
  const router = useRouter()
  const routeParams = useParams()
  const rawModelId = Array.isArray(routeParams.modelId) ? routeParams.modelId[0] : routeParams.modelId
  const modelId = useMemo(() => decodeURIComponent(rawModelId || ''), [rawModelId])

  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const [operationFilter, setOperationFilter] = useState('all')
  const [windowSeconds, setWindowSeconds] = useState(300)

  const [topology, setTopology] = useState<ModelProviderTopologyResponse | null>(null)

  const [showPricingModal, setShowPricingModal] = useState(false)
  const [selectedProviderId, setSelectedProviderId] = useState<number | null>(null)
  const [editingRuleId, setEditingRuleId] = useState<number | null>(null)
  const [deletingRuleId, setDeletingRuleId] = useState<number | null>(null)
  const [savingRule, setSavingRule] = useState(false)
  const [deletingRule, setDeletingRule] = useState(false)
  const [togglingRuleId, setTogglingRuleId] = useState<number | null>(null)
  const [ruleForm, setRuleForm] = useState<PricingRuleFormState>(DEFAULT_RULE_FORM)

  const load = useCallback(async ({ silent }: { silent?: boolean } = {}) => {
    if (!silent) setRefreshing(true)
    try {
      if (!modelId) {
        setTopology(null)
        return
      }
      const res = await topologyApi.modelProviders({
        model_id: modelId,
        metrics_window_seconds: windowSeconds,
        include_instances: false,
        include_pricing_rules: true,
      })
      setTopology(res.data || null)
    } catch (error) {
      toast({
        title: '加载模型源头失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [modelId, toast, windowSeconds])

  useEffect(() => {
    load({ silent: true })
  }, [load])

  const model = useMemo(() => {
    const models = topology?.models ?? []
    return models.find((m) => m.id === modelId) || models[0] || null
  }, [modelId, topology])

  const allOperations = useMemo(() => {
    const ops = (model?.operations ?? []).map((op) => op.operation).filter(Boolean)
    return Array.from(new Set(ops)).sort((a, b) => String(a).localeCompare(String(b)))
  }, [model])

  const providers = useMemo<TopologyProvider[]>(() => {
    const list: TopologyProvider[] = []
    for (const op of model?.operations ?? []) {
      for (const p of op.providers ?? []) {
        list.push(p)
      }
    }
    return list
  }, [model])

  const filteredProviders = useMemo(() => {
    if (operationFilter === 'all') return providers
    return providers.filter((p) => p.operation === operationFilter)
  }, [providers, operationFilter])

  const summary = useMemo(() => {
    const total = providers.length
    const circuitOpen = providers.filter((p) => p.circuit_state === 'open').length
    const unavailable = providers.filter((p) => p.status !== 'active' && p.circuit_state !== 'open').length
    const available = providers.filter((p) => p.status === 'active' && p.circuit_state !== 'open').length
    return { total, available, circuitOpen, unavailable }
  }, [providers])

  const selectedProvider = useMemo<TopologyProvider | null>(() => {
    if (!selectedProviderId) return null
    return providers.find((p) => p.id === selectedProviderId) || null
  }, [providers, selectedProviderId])

  const openPricing = useCallback((provider: TopologyProvider) => {
    setSelectedProviderId(provider.id)
    setEditingRuleId(null)
    setDeletingRuleId(null)
    setRuleForm({
      ...DEFAULT_RULE_FORM,
      operation: provider.operation || DEFAULT_RULE_FORM.operation,
    })
    setShowPricingModal(true)
  }, [])

  const startEditRule = useCallback((rule: TopologyProviderPricingRule) => {
    setEditingRuleId(rule.id)
    setRuleForm({
      operation: rule.operation || DEFAULT_RULE_FORM.operation,
      unit: rule.unit || DEFAULT_RULE_FORM.unit,
      cost_per_unit: String(rule.cost_per_unit ?? '0'),
      price_per_unit: String(rule.price_per_unit ?? '0'),
      enabled: Boolean(rule.enabled),
    })
  }, [])

  const resetRuleForm = useCallback(() => {
    setEditingRuleId(null)
    setRuleForm({
      ...DEFAULT_RULE_FORM,
      operation: selectedProvider?.operation || DEFAULT_RULE_FORM.operation,
    })
  }, [selectedProvider?.operation])

  const submitRule = useCallback(async () => {
    if (!selectedProvider) return

    const operation = ruleForm.operation.trim()
    const unit = ruleForm.unit.trim()
    if (!operation) {
      toast({ title: 'operation 不能为空', variant: 'destructive' })
      return
    }
    if (!unit) {
      toast({ title: 'unit 不能为空', variant: 'destructive' })
      return
    }

    const cost = Number(ruleForm.cost_per_unit)
    const price = Number(ruleForm.price_per_unit)
    if (!Number.isFinite(cost) || cost < 0) {
      toast({ title: 'cost_per_unit 必须是非负数', variant: 'destructive' })
      return
    }
    if (!Number.isFinite(price) || price < 0) {
      toast({ title: 'price_per_unit 必须是非负数', variant: 'destructive' })
      return
    }

    setSavingRule(true)
    try {
      if (editingRuleId) {
        const payload: UpdateProviderPricingRuleRequest = {
          operation,
          unit,
          cost_per_unit: cost,
          price_per_unit: price,
          enabled: ruleForm.enabled,
        }
        await providerPricingRuleApi.update(editingRuleId, payload)
        toast({ title: '已更新计费规则' })
      } else {
        const payload: CreateProviderPricingRuleRequest = {
          provider_id: selectedProvider.id,
          operation,
          unit,
          cost_per_unit: cost,
          price_per_unit: price,
          enabled: ruleForm.enabled,
        }
        await providerPricingRuleApi.create(payload)
        toast({ title: '已创建计费规则' })
      }

      await load({ silent: true })
      resetRuleForm()
    } catch (error) {
      toast({
        title: '保存失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setSavingRule(false)
    }
  }, [editingRuleId, load, resetRuleForm, ruleForm, selectedProvider, toast])

  const confirmDeleteRule = useCallback(async () => {
    if (!deletingRuleId) return

    setDeletingRule(true)
    try {
      await providerPricingRuleApi.delete(deletingRuleId)
      toast({ title: '已删除计费规则' })
      await load({ silent: true })
      setDeletingRuleId(null)
      if (editingRuleId === deletingRuleId) {
        resetRuleForm()
      }
    } catch (error) {
      toast({
        title: '删除失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setDeletingRule(false)
    }
  }, [deletingRuleId, editingRuleId, load, resetRuleForm, toast])

  const toggleRuleEnabled = useCallback(async (ruleId: number, enabled: boolean) => {
    setTogglingRuleId(ruleId)
    try {
      const payload: UpdateProviderPricingRuleRequest = { enabled }
      await providerPricingRuleApi.update(ruleId, payload)
      await load({ silent: true })
      toast({ title: enabled ? '已启用规则' : '已禁用规则' })
    } catch (error) {
      toast({
        title: '更新失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setTogglingRuleId(null)
    }
  }, [load, toast])

  const pricingSummaryText = useCallback((provider: TopologyProvider) => {
    const total = provider.pricing_rules?.length || 0
    const enabled = provider.pricing_rules?.filter((r) => r.enabled).length || 0
    if (total === 0) return '未配置'
    return `已配置 ${enabled}/${total}`
  }, [])

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3">
        <Link href="/dashboard/pricing/sources" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="w-4 h-4" />
          返回模型列表
        </Link>

        <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold">源头定价 · {model?.name || modelId}</h1>
            <p className="text-sm text-muted-foreground mt-1">
              展示模型的源头状态、调用数据，并按源头配置计费规则（cost/price per unit）。
            </p>
          </div>
          <Button variant="outline" onClick={() => load()} disabled={loading || refreshing}>
            <RefreshCw className={cn('w-4 h-4 mr-2', (loading || refreshing) && 'animate-spin')} />
            刷新
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="p-4">
          <div className="text-sm text-muted-foreground">源头数</div>
          <div className="text-2xl font-bold mt-1">{summary.total}</div>
        </Card>
        <Card className="p-4">
          <div className="text-sm text-muted-foreground">可用</div>
          <div className="text-2xl font-bold mt-1 text-green-600">{summary.available}</div>
        </Card>
        <Card className="p-4">
          <div className="text-sm text-muted-foreground">熔断</div>
          <div className="text-2xl font-bold mt-1 text-red-600">{summary.circuitOpen}</div>
        </Card>
        <Card className="p-4">
          <div className="text-sm text-muted-foreground">不可用</div>
          <div className="text-2xl font-bold mt-1 text-gray-600">{summary.unavailable}</div>
        </Card>
      </div>

      <Card className="p-4">
        <div className="flex flex-col md:flex-row md:items-center gap-3">
          <div className="flex-1">
            <label className="block text-xs text-muted-foreground mb-1">operation</label>
            <select
              value={operationFilter}
              onChange={(e) => setOperationFilter(e.target.value)}
              className="w-full px-3 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            >
              <option value="all">全部</option>
              {allOperations.map((op) => (
                <option key={op} value={op}>
                  {op}
                </option>
              ))}
            </select>
          </div>
          <div className="flex-1">
            <label className="block text-xs text-muted-foreground mb-1">最近调用窗口</label>
            <select
              value={String(windowSeconds)}
              onChange={(e) => setWindowSeconds(Number(e.target.value))}
              className="w-full px-3 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            >
              <option value="300">近 5 分钟</option>
              <option value="900">近 15 分钟</option>
              <option value="3600">近 60 分钟</option>
            </select>
          </div>
          <div className="md:self-end">
            <Button variant="outline" onClick={() => load()} disabled={loading || refreshing}>
              <RefreshCw className={cn('w-4 h-4 mr-2', (loading || refreshing) && 'animate-spin')} />
              应用
            </Button>
          </div>
        </div>
      </Card>

      {loading ? (
        <Loading />
      ) : !model ? (
        <Card className="p-6">
          <EmptyState
            icon={CreditCard}
            title="未找到模型"
            description="拓扑数据中未找到该模型，请检查 model_providers 配置。"
            action={{ label: '返回列表', onClick: () => router.push('/dashboard/pricing/sources') }}
          />
        </Card>
      ) : filteredProviders.length === 0 ? (
        <Card className="p-6">
          <EmptyState
            icon={CreditCard}
            title="暂无源头"
            description="该模型下暂无源头（model_providers），或被筛选条件过滤。"
            action={{ label: '刷新', onClick: () => load() }}
          />
        </Card>
      ) : (
        <Card className="p-4">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">源头</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">状态</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">累计调用</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">最近调用</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">价格规则</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {filteredProviders.map((p) => (
                  <tr key={p.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <div className="font-medium">{p.channel_name}</div>
                      <div className="text-xs text-muted-foreground flex items-center gap-2">
                        <span className="font-mono">#{p.id}</span>
                        <span className="font-mono">{p.operation}</span>
                      </div>
                      {p.upstream_model_name ? (
                        <div className="text-xs text-muted-foreground mt-1">
                          upstream: <span className="font-mono">{p.upstream_model_name}</span>
                        </div>
                      ) : null}
                    </td>
                    <td className="px-4 py-3 space-y-2">
                      <div className="flex items-center gap-2">
                        <StatusBadge status={p.status} />
                        <CircuitBadge state={p.circuit_state} />
                      </div>
                      <div className="text-xs text-muted-foreground">
                        健康分: <span className="font-medium">{formatDecimal(p.health_score, 2)}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span className="font-medium">{p.total_requests ?? 0}</span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="font-medium">{p.metrics?.request_count ?? 0}</div>
                      <div className="text-xs text-muted-foreground">
                        成功率 {formatDecimal(p.metrics?.success_rate ?? 0, 2)}% · 延迟 {formatDecimal(p.metrics?.avg_latency_ms ?? 0, 0)}ms
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="font-medium">{pricingSummaryText(p)}</div>
                      {p.pricing_rules && p.pricing_rules.length > 0 ? (
                        <div className="text-xs text-muted-foreground">
                          最近更新: {formatDate(getLatestUpdatedAt(p.pricing_rules) || p.pricing_rules[0].updated_at)}
                        </div>
                      ) : (
                        <div className="text-xs text-muted-foreground">—</div>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => openPricing(p)}
                        className="inline-flex items-center gap-2 px-3 py-1.5 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
                      >
                        配置价格
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <Modal
        isOpen={showPricingModal}
        onClose={() => {
          setShowPricingModal(false)
          setSelectedProviderId(null)
          setDeletingRuleId(null)
          resetRuleForm()
        }}
        title={selectedProvider ? `源头计费规则 · ${selectedProvider.channel_name} (#${selectedProvider.id})` : '源头计费规则'}
        size="xl"
      >
        {!selectedProvider ? (
          <EmptyState icon={CreditCard} title="请选择源头" description="未找到对应源头数据，请刷新后重试。" />
        ) : (
          <div className="space-y-6">
            <Card className="p-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                <div>
                  <div className="text-xs text-muted-foreground">operation</div>
                  <div className="font-mono mt-1">{selectedProvider.operation}</div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground">状态</div>
                  <div className="flex items-center gap-2 mt-1">
                    <StatusBadge status={selectedProvider.status} />
                    <CircuitBadge state={selectedProvider.circuit_state} />
                  </div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground">累计/最近调用</div>
                  <div className="mt-1">
                    <span className="font-medium">{selectedProvider.total_requests ?? 0}</span>
                    <span className="text-muted-foreground"> / </span>
                    <span className="font-medium">{selectedProvider.metrics?.request_count ?? 0}</span>
                    <span className="text-muted-foreground">（近 {selectedProvider.metrics?.window_seconds ?? windowSeconds}s）</span>
                  </div>
                </div>
              </div>
            </Card>

            <Card className="p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Plus className="w-4 h-4 text-orange-600" />
                  <div className="font-medium">{editingRuleId ? `编辑规则 #${editingRuleId}` : '新增规则'}</div>
                </div>
                {editingRuleId ? (
                  <Button variant="outline" onClick={resetRuleForm} disabled={savingRule}>
                    取消编辑
                  </Button>
                ) : null}
              </div>

              <div className="mt-4 grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">operation</label>
                  <select
                    value={ruleForm.operation}
                    onChange={(e) => setRuleForm((prev) => ({ ...prev, operation: e.target.value }))}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  >
                    {OPERATION_OPTIONS.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">unit</label>
                  <input
                    value={ruleForm.unit}
                    onChange={(e) => setRuleForm((prev) => ({ ...prev, unit: e.target.value }))}
                    placeholder="request / image / second / 1k_tokens ..."
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">cost_per_unit</label>
                  <input
                    type="number"
                    min="0"
                    step="0.000001"
                    value={ruleForm.cost_per_unit}
                    onChange={(e) => setRuleForm((prev) => ({ ...prev, cost_per_unit: e.target.value }))}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">price_per_unit</label>
                  <input
                    type="number"
                    min="0"
                    step="0.000001"
                    value={ruleForm.price_per_unit}
                    onChange={(e) => setRuleForm((prev) => ({ ...prev, price_per_unit: e.target.value }))}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  />
                </div>
              </div>

              <div className="mt-4 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <Switch
                    checked={ruleForm.enabled}
                    onCheckedChange={(checked) => setRuleForm((prev) => ({ ...prev, enabled: checked }))}
                  />
                  <span className="text-sm text-muted-foreground">启用规则</span>
                </div>

                <div className="flex items-center gap-3">
                  <Button variant="outline" onClick={() => load()} disabled={savingRule}>
                    <RefreshCw className={cn('w-4 h-4 mr-2', savingRule && 'animate-spin')} />
                    刷新
                  </Button>
                  <Button onClick={submitRule} disabled={savingRule}>
                    {savingRule ? '保存中...' : editingRuleId ? '保存修改' : '创建规则'}
                  </Button>
                </div>
              </div>
            </Card>

            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <CreditCard className="w-4 h-4 text-orange-600" />
                <div className="font-medium">当前规则</div>
              </div>
              <Link
                href={`/dashboard/pricing-rules?provider_id=${encodeURIComponent(String(selectedProvider.id))}`}
                className="text-sm text-orange-600 hover:text-orange-700"
              >
                去“计费规则”查看更多
              </Link>
            </div>

            {selectedProvider.pricing_rules && selectedProvider.pricing_rules.length > 0 ? (
              <Card className="p-0 overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 text-sm">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">operation / unit</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">cost</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">price</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">启用</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">更新时间</th>
                        <th className="px-4 py-3 text-right text-xs font-medium text-gray-500">操作</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200">
                      {selectedProvider.pricing_rules.map((r) => (
                        <tr key={r.id} className="hover:bg-gray-50">
                          <td className="px-4 py-3">
                            <div className="font-mono">{r.operation}</div>
                            <div className="text-xs text-muted-foreground font-mono">{r.unit}</div>
                            <div className="text-xs text-muted-foreground">#{r.id}</div>
                          </td>
                          <td className="px-4 py-3 font-mono">{formatDecimal(r.cost_per_unit)}</td>
                          <td className="px-4 py-3 font-mono">{formatDecimal(r.price_per_unit)}</td>
                          <td className="px-4 py-3">
                            <Switch
                              checked={Boolean(r.enabled)}
                              onCheckedChange={(checked) => toggleRuleEnabled(r.id, checked)}
                              disabled={togglingRuleId === r.id}
                            />
                          </td>
                          <td className="px-4 py-3 text-muted-foreground">{formatDate(r.updated_at)}</td>
                          <td className="px-4 py-3 text-right">
                            <div className="inline-flex items-center gap-2">
                              <button
                                onClick={() => startEditRule(r)}
                                className="p-2 hover:bg-orange-500/10 rounded-lg transition-colors text-orange-600"
                                aria-label="编辑"
                              >
                                <Pencil className="w-4 h-4" />
                              </button>
                              <button
                                onClick={() => setDeletingRuleId(r.id)}
                                className="p-2 hover:bg-red-500/10 rounded-lg transition-colors text-red-500"
                                aria-label="删除"
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </Card>
            ) : (
              <Card className="p-6">
                <EmptyState icon={CreditCard} title="暂无规则" description="你可以先新增一条规则，或跳转到“计费规则”页面批量配置。" />
              </Card>
            )}
          </div>
        )}
      </Modal>

      <ConfirmDialog
        open={Boolean(deletingRuleId)}
        onOpenChange={(open) => {
          if (!open) setDeletingRuleId(null)
        }}
        title="确认删除计费规则？"
        description={`将删除规则 #${deletingRuleId || ''}。此操作不可撤销。`}
        confirmText="删除"
        variant="destructive"
        loading={deletingRule}
        onConfirm={confirmDeleteRule}
      />
    </div>
  )
}
