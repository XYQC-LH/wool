'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'next/navigation'
import {
  CreateProviderPricingRuleRequest,
  ModelProviderResponse,
  ProviderPricingRule,
  UpdateProviderPricingRuleRequest,
  getErrorMessage,
  modelProviderApi,
  providerPricingRuleApi,
} from '@/lib/api'
import { Pagination } from '@/components/common/pagination'
import { Loading } from '@/components/common/loading'
import { EmptyState } from '@/components/common/empty-state'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { Modal } from '@/components/ui/modal'
import { Switch } from '@/components/ui/switch'
import { useToast } from '@/components/ui/use-toast'
import { formatDate } from '@/lib/utils'
import { CreditCard, Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { OPERATION_OPTIONS } from '@/lib/operations'

function toNumber(value: unknown, fallback: number): number {
  const num = Number(value)
  if (!Number.isFinite(num)) return fallback
  return num
}

function formatDecimal(value: unknown, fractionDigits = 6): string {
  const num = Number(value)
  if (!Number.isFinite(num)) return String(value ?? '')
  return num.toFixed(fractionDigits)
}

export default function ProviderPricingRulesPage() {
  const { toast } = useToast()
  const searchParams = useSearchParams()

  const [providers, setProviders] = useState<ModelProviderResponse[]>([])
  const [rules, setRules] = useState<ProviderPricingRule[]>([])

  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)

  const [filterProviderId, setFilterProviderId] = useState('all')
  const [filterOperation, setFilterOperation] = useState('all')

  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const [selectedRule, setSelectedRule] = useState<ProviderPricingRule | null>(null)

  const providerOptions = useMemo(() => {
    return providers.map((p) => {
      const modelLabel = p.model?.name || p.model_name || p.model_id
      const channelLabel = p.channel?.name || p.channel_name || `channel#${p.channel_id}`
      return { value: String(p.id), label: `${p.id} · ${modelLabel} · ${channelLabel} · ${p.operation}` }
    })
  }, [providers])

  const loadProviders = useCallback(async () => {
    try {
      const res = await modelProviderApi.list({ page: 1, page_size: 200 })
      if (res.data?.list) {
        setProviders(res.data.list)
      }
    } catch (error) {
      toast({
        title: '加载源头失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }, [toast])

  const loadRules = useCallback(async () => {
    setLoading(true)
    try {
      const params: { page: number; page_size: number; provider_id?: number; operation?: string } = { page, page_size: pageSize }
      if (filterProviderId !== 'all') params.provider_id = Number(filterProviderId)
      if (filterOperation !== 'all') params.operation = filterOperation

      const res = await providerPricingRuleApi.list(params)
      setRules(res.data?.list || [])
      setTotal(res.data?.total || 0)
    } catch (error) {
      toast({
        title: '加载计费规则失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [filterOperation, filterProviderId, page, pageSize, toast])

  useEffect(() => {
    loadProviders()
  }, [loadProviders])

  useEffect(() => {
    const providerId = searchParams.get('provider_id')
    const operation = searchParams.get('operation')

    let changed = false
    if (providerId && providerId !== filterProviderId) {
      setFilterProviderId(providerId)
      changed = true
    }
    if (operation && operation !== filterOperation) {
      setFilterOperation(operation)
      changed = true
    }
    if (changed) {
      setPage(1)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams])

  useEffect(() => {
    loadRules()
  }, [loadRules])

  const totalPages = Math.ceil(total / pageSize)

  const [formData, setFormData] = useState({
    provider_id: '',
    operation: 'images.generations',
    unit: 'request',
    cost_per_unit: '0',
    price_per_unit: '0',
    meta: '',
    enabled: true,
  })

  const resetForm = useCallback(() => {
    setFormData({
      provider_id: '',
      operation: 'images.generations',
      unit: 'request',
      cost_per_unit: '0',
      price_per_unit: '0',
      meta: '',
      enabled: true,
    })
  }, [])

  const openCreate = useCallback(() => {
    resetForm()
    setSelectedRule(null)
    setShowCreateModal(true)
  }, [resetForm])

  const openEdit = useCallback((rule: ProviderPricingRule) => {
    setSelectedRule(rule)
    setFormData({
      provider_id: String(rule.provider_id),
      operation: rule.operation || 'images.generations',
      unit: rule.unit || 'request',
      cost_per_unit: String(rule.cost_per_unit ?? '0'),
      price_per_unit: String(rule.price_per_unit ?? '0'),
      meta: rule.meta ? JSON.stringify(rule.meta, null, 2) : '',
      enabled: Boolean(rule.enabled),
    })
    setShowEditModal(true)
  }, [])

  const onSubmit = useCallback(async () => {
    const providerIdNum = toNumber(formData.provider_id, 0)
    if (providerIdNum <= 0) {
      toast({ title: '请先选择源头（provider_id）', variant: 'destructive' })
      return
    }
    if (!formData.operation.trim()) {
      toast({ title: 'operation 不能为空', variant: 'destructive' })
      return
    }
    if (!formData.unit.trim()) {
      toast({ title: 'unit 不能为空', variant: 'destructive' })
      return
    }

    const cost = toNumber(formData.cost_per_unit, 0)
    const price = toNumber(formData.price_per_unit, 0)
    if (cost < 0 || price < 0) {
      toast({ title: 'cost/price 不能为负数', variant: 'destructive' })
      return
    }

    let meta: Record<string, unknown> | undefined
    const metaRaw = formData.meta.trim()
    if (metaRaw) {
      try {
        const parsed = JSON.parse(metaRaw)
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          meta = parsed as Record<string, unknown>
        } else {
          toast({ title: 'meta JSON 无效', description: '请提供对象类型 JSON 或留空', variant: 'destructive' })
          return
        }
      } catch {
        toast({ title: 'meta JSON 无效', description: '请提供合法 JSON 或留空', variant: 'destructive' })
        return
      }
    }

    setSaving(true)
    try {
      if (selectedRule) {
        const payload: UpdateProviderPricingRuleRequest = {
          operation: formData.operation.trim(),
          unit: formData.unit.trim(),
          cost_per_unit: cost,
          price_per_unit: price,
          meta,
          enabled: formData.enabled,
        }
        await providerPricingRuleApi.update(selectedRule.id, payload)
        toast({ title: '更新成功', description: '计费规则已更新' })
      } else {
        const payload: CreateProviderPricingRuleRequest = {
          provider_id: providerIdNum,
          operation: formData.operation.trim(),
          unit: formData.unit.trim(),
          cost_per_unit: cost,
          price_per_unit: price,
          meta,
          enabled: formData.enabled,
        }
        await providerPricingRuleApi.create(payload)
        toast({ title: '创建成功', description: '计费规则已创建' })
      }

      setShowCreateModal(false)
      setShowEditModal(false)
      resetForm()
      setSelectedRule(null)
      loadRules()
    } catch (error) {
      toast({
        title: selectedRule ? '更新失败' : '创建失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setSaving(false)
    }
  }, [formData, loadRules, resetForm, selectedRule, toast])

  const onDelete = useCallback((rule: ProviderPricingRule) => {
    setSelectedRule(rule)
    setShowDeleteConfirm(true)
  }, [])

  const confirmDelete = useCallback(async () => {
    if (!selectedRule) return
    setDeleting(true)
    try {
      await providerPricingRuleApi.delete(selectedRule.id)
      toast({ title: '删除成功', description: '计费规则已删除' })
      setShowDeleteConfirm(false)
      setSelectedRule(null)
      loadRules()
    } catch (error) {
      toast({
        title: '删除失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setDeleting(false)
    }
  }, [loadRules, selectedRule, toast])

  const toggleEnabled = useCallback(
    async (rule: ProviderPricingRule, enabled: boolean) => {
      if (!rule) return

      setRules((prev) => prev.map((item) => (item.id === rule.id ? { ...item, enabled } : item)))

      try {
        const payload: UpdateProviderPricingRuleRequest = { enabled }
        await providerPricingRuleApi.update(rule.id, payload)
        toast({ title: '更新成功', description: enabled ? '已启用' : '已禁用' })
      } catch (error) {
        setRules((prev) => prev.map((item) => (item.id === rule.id ? { ...item, enabled: rule.enabled } : item)))
        toast({
          title: '更新失败',
          description: getErrorMessage(error),
          variant: 'destructive',
        })
      }
    },
    [toast]
  )

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">多模态计费规则</h1>
          <p className="text-muted-foreground">按 operation + unit 配置成本与售价（用于调度排序 / 扣费）</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={loadRules}
            className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg hover:bg-accent transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
            刷新
          </button>
          <button
            onClick={openCreate}
            className="flex items-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
            新建规则
          </button>
        </div>
      </div>

      <div className="flex flex-col sm:flex-row gap-4">
        <select
          value={filterProviderId}
          onChange={(e) => {
            setFilterProviderId(e.target.value)
            setPage(1)
          }}
          className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
        >
          <option value="all">全部源头</option>
          {providerOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>

        <select
          value={filterOperation}
          onChange={(e) => {
            setFilterOperation(e.target.value)
            setPage(1)
          }}
          className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
        >
          <option value="all">全部操作</option>
          {OPERATION_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-muted/50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  源头
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  操作
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  单位
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  成本
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  售价
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading ? (
                <tr>
                  <td colSpan={7} className="px-6 py-12 text-center">
                    <div className="flex items-center justify-center">
                      <Loading size="md" />
                    </div>
                  </td>
                </tr>
              ) : rules.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-12">
                    <EmptyState
                      icon={CreditCard}
                      title="暂无计费规则"
                      description="你可以为多模态 operation 配置按 unit 计费的成本与售价。"
                      action={{ label: '新建规则', onClick: openCreate }}
                    />
                  </td>
                </tr>
              ) : (
                rules.map((rule) => {
                  const providerLabel = rule.provider
                    ? `${rule.provider.model?.name || rule.provider.model_id || rule.provider_id} · ${rule.provider.channel?.name || ''}`
                    : String(rule.provider_id)

                  return (
                    <tr key={rule.id} className="hover:bg-muted/30 transition-colors">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-orange-500/10 rounded-lg">
                            <CreditCard className="w-5 h-5 text-orange-500" />
                          </div>
                          <div>
                            <div className="font-medium">{providerLabel}</div>
                            <div className="text-sm text-muted-foreground">
                              #{rule.provider_id} · {formatDate(rule.updated_at || rule.created_at)}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="font-mono text-sm">{rule.operation}</span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="font-mono text-sm">{rule.unit}</span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="font-mono text-sm">{formatDecimal(rule.cost_per_unit)}</span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="font-mono text-sm">{formatDecimal(rule.price_per_unit)}</span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-3">
                          <Switch checked={rule.enabled} onCheckedChange={(checked) => toggleEnabled(rule, checked)} />
                          <span className="text-sm text-muted-foreground">{rule.enabled ? '启用' : '禁用'}</span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => openEdit(rule)}
                            className="p-2 hover:bg-accent rounded-lg transition-colors"
                            aria-label="编辑"
                          >
                            <Pencil className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => onDelete(rule)}
                            className="p-2 hover:bg-red-500/10 rounded-lg transition-colors text-red-500"
                            aria-label="删除"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>

        {!loading && rules.length > 0 && (
          <div className="px-6 py-4 border-t border-border">
            <Pagination
              currentPage={page}
              totalPages={totalPages}
              total={total}
              pageSize={pageSize}
              onPageChange={setPage}
            />
          </div>
        )}
      </div>

      <Modal
        isOpen={showCreateModal || showEditModal}
        onClose={() => {
          setShowCreateModal(false)
          setShowEditModal(false)
          setSelectedRule(null)
          resetForm()
        }}
        title={selectedRule ? '编辑计费规则' : '新建计费规则'}
        size="lg"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">源头（provider_id）</label>
            <select
              value={formData.provider_id}
              onChange={(e) => setFormData((prev) => ({ ...prev, provider_id: e.target.value }))}
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            >
              <option value="">请选择源头</option>
              {providerOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-2">operation</label>
              <select
                value={formData.operation}
                onChange={(e) => setFormData((prev) => ({ ...prev, operation: e.target.value }))}
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
                value={formData.unit}
                onChange={(e) => setFormData((prev) => ({ ...prev, unit: e.target.value }))}
                placeholder="request / image / video_second ..."
                className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-2">cost_per_unit</label>
              <input
                type="number"
                min="0"
                step="0.000001"
                value={formData.cost_per_unit}
                onChange={(e) => setFormData((prev) => ({ ...prev, cost_per_unit: e.target.value }))}
                className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">price_per_unit</label>
              <input
                type="number"
                min="0"
                step="0.000001"
                value={formData.price_per_unit}
                onChange={(e) => setFormData((prev) => ({ ...prev, price_per_unit: e.target.value }))}
                className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">meta（JSON，可选）</label>
            <textarea
              value={formData.meta}
              onChange={(e) => setFormData((prev) => ({ ...prev, meta: e.target.value }))}
              placeholder='{"currency":"USD","note":"optional"}'
              rows={4}
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500 font-mono text-sm"
            />
          </div>

          <div className="flex items-center justify-between pt-2">
            <div className="flex items-center gap-3">
              <Switch
                checked={formData.enabled}
                onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, enabled: checked }))}
              />
              <span className="text-sm text-muted-foreground">启用规则</span>
            </div>

            <div className="flex items-center gap-3">
              <button
                onClick={() => {
                  setShowCreateModal(false)
                  setShowEditModal(false)
                  setSelectedRule(null)
                  resetForm()
                }}
                className="px-4 py-2 border border-border rounded-lg hover:bg-accent transition-colors"
                disabled={saving}
              >
                取消
              </button>
              <button
                onClick={onSubmit}
                disabled={saving}
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors disabled:opacity-50"
              >
                {saving ? '保存中...' : '保存'}
              </button>
            </div>
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title="确认删除计费规则？"
        description={`将删除规则 #${selectedRule?.id || ''}（provider_id=${selectedRule?.provider_id || ''}，${selectedRule?.operation || ''} / ${selectedRule?.unit || ''}）。此操作不可撤销。`}
        confirmText="删除"
        variant="destructive"
        loading={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  )
}
