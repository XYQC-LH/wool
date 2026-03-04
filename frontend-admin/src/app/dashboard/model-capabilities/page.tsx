'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  CreateModelCapabilityRequest,
  Model,
  ModelCapability,
  UpdateModelCapabilityRequest,
  getErrorMessage,
  modelApi,
  modelCapabilityApi,
} from '@/lib/api'
import { OPERATION_OPTIONS } from '@/lib/operations'
import { Pagination } from '@/components/common/pagination'
import { Loading } from '@/components/common/loading'
import { EmptyState } from '@/components/common/empty-state'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { Modal } from '@/components/ui/modal'
import { Switch } from '@/components/ui/switch'
import { useToast } from '@/components/ui/use-toast'
import { formatDate } from '@/lib/utils'
import { Plus, RefreshCw, SlidersHorizontal, Trash2 } from 'lucide-react'

export default function ModelCapabilitiesPage() {
  const { toast } = useToast()

  const [models, setModels] = useState<Model[]>([])
  const [capabilities, setCapabilities] = useState<ModelCapability[]>([])

  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)

  const [filterModelId, setFilterModelId] = useState('')
  const [filterOperation, setFilterOperation] = useState('all')

  const [showCreateModal, setShowCreateModal] = useState(false)
  const [saving, setSaving] = useState(false)

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [selected, setSelected] = useState<ModelCapability | null>(null)

  const modelSuggestions = useMemo(() => models.map((m) => ({ id: m.id, name: m.name })), [models])

  const loadModels = useCallback(async () => {
    try {
      const res = await modelApi.list({ page: 1, page_size: 200 })
      setModels(res.data?.list || [])
    } catch (error) {
      toast({
        title: '加载模型列表失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }, [toast])

  const loadCapabilities = useCallback(async () => {
    setLoading(true)
    try {
      const params: { model_id?: string; operation?: string; page: number; page_size: number } = {
        page,
        page_size: pageSize,
      }
      if (filterModelId.trim()) params.model_id = filterModelId.trim()
      if (filterOperation !== 'all') params.operation = filterOperation

      const res = await modelCapabilityApi.list(params)
      setCapabilities(res.data?.list || [])
      setTotal(res.data?.total || 0)
    } catch (error) {
      toast({
        title: '加载模型能力失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [filterModelId, filterOperation, page, pageSize, toast])

  useEffect(() => {
    loadModels()
  }, [loadModels])

  useEffect(() => {
    loadCapabilities()
  }, [loadCapabilities])

  const totalPages = Math.ceil(total / pageSize)

  const [formData, setFormData] = useState({
    model_id: '',
    operation: 'chat.completions',
    enabled: true,
  })

  const resetForm = useCallback(() => {
    setFormData({
      model_id: '',
      operation: 'chat.completions',
      enabled: true,
    })
  }, [])

  const openCreate = useCallback(() => {
    resetForm()
    setShowCreateModal(true)
  }, [resetForm])

  const submitCreate = useCallback(async () => {
    if (!formData.model_id.trim()) {
      toast({ title: 'model_id 不能为空', variant: 'destructive' })
      return
    }
    if (!formData.operation.trim()) {
      toast({ title: 'operation 不能为空', variant: 'destructive' })
      return
    }

    const payload: CreateModelCapabilityRequest = {
      model_id: formData.model_id.trim(),
      operation: formData.operation.trim(),
      enabled: formData.enabled,
    }

    setSaving(true)
    try {
      await modelCapabilityApi.create(payload)
      toast({ title: '创建成功', description: '模型能力已创建' })
      setShowCreateModal(false)
      resetForm()
      loadCapabilities()
    } catch (error) {
      toast({
        title: '创建失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setSaving(false)
    }
  }, [formData, loadCapabilities, resetForm, toast])

  const toggleEnabled = useCallback(async (capability: ModelCapability, enabled: boolean) => {
    if (!capability) return

    setCapabilities((prev) => prev.map((item) => (item.id === capability.id ? { ...item, enabled } : item)))

    const payload: UpdateModelCapabilityRequest = { enabled }
    try {
      await modelCapabilityApi.update(capability.id, payload)
      toast({ title: '更新成功', description: enabled ? '已启用' : '已禁用' })
    } catch (error) {
      setCapabilities((prev) => prev.map((item) => (item.id === capability.id ? { ...item, enabled: capability.enabled } : item)))
      toast({
        title: '更新失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }, [toast])

  const askDelete = useCallback((capability: ModelCapability) => {
    setSelected(capability)
    setShowDeleteConfirm(true)
  }, [])

  const confirmDelete = useCallback(async () => {
    if (!selected) return
    setDeleting(true)
    try {
      await modelCapabilityApi.delete(selected.id)
      toast({ title: '删除成功', description: '模型能力已删除' })
      setShowDeleteConfirm(false)
      setSelected(null)
      loadCapabilities()
    } catch (error) {
      toast({
        title: '删除失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setDeleting(false)
    }
  }, [loadCapabilities, selected, toast])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">模型能力</h1>
          <p className="text-muted-foreground">为模型定义支持的 operation，并在调度链路进行能力校验</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={loadCapabilities}
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
            新建能力
          </button>
        </div>
      </div>

      <div className="flex flex-col sm:flex-row gap-4">
        <div className="flex-1">
          <input
            value={filterModelId}
            onChange={(e) => {
              setFilterModelId(e.target.value)
              setPage(1)
            }}
            placeholder="按 model_id 筛选（支持模糊由后端决定）"
            className="w-full px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
          />
        </div>

        <select
          value={filterOperation}
          onChange={(e) => {
            setFilterOperation(e.target.value)
            setPage(1)
          }}
          className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
        >
          <option value="all">全部 operation</option>
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
                  模型
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Operation
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  启用
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  更新时间
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading ? (
                <tr>
                  <td colSpan={5} className="px-6 py-12 text-center">
                    <div className="flex items-center justify-center">
                      <Loading size="md" />
                    </div>
                  </td>
                </tr>
              ) : capabilities.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-6 py-12">
                    <EmptyState
                      icon={SlidersHorizontal}
                      title="暂无模型能力配置"
                      description="未配置时默认不拦截；配置后可对模型 operation 进行启用/禁用。"
                      action={{ label: '新建能力', onClick: openCreate }}
                    />
                  </td>
                </tr>
              ) : (
                capabilities.map((cap) => {
                  const modelLabel = cap.model?.name || cap.model_id
                  return (
                    <tr key={cap.id} className="hover:bg-muted/30 transition-colors">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="font-medium">{modelLabel}</div>
                        <div className="text-sm text-muted-foreground font-mono">{cap.model_id}</div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="font-mono text-sm">{cap.operation}</span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Switch checked={cap.enabled} onCheckedChange={(checked) => toggleEnabled(cap, checked)} />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground">
                        {formatDate(cap.updated_at || cap.created_at)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right">
                        <button
                          onClick={() => askDelete(cap)}
                          className="p-2 hover:bg-red-500/10 rounded-lg transition-colors text-red-500"
                          aria-label="删除"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>

        {!loading && capabilities.length > 0 && (
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
        isOpen={showCreateModal}
        onClose={() => {
          setShowCreateModal(false)
          resetForm()
        }}
        title="新建模型能力"
        size="lg"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">model_id</label>
            <input
              list="model-options"
              value={formData.model_id}
              onChange={(e) => setFormData((prev) => ({ ...prev, model_id: e.target.value }))}
              placeholder="例如：gpt-4o / sora2 / nano-banana ..."
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
            <datalist id="model-options">
              {modelSuggestions.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
            </datalist>
          </div>

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

          <div className="flex items-center gap-3">
            <Switch checked={formData.enabled} onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, enabled: checked }))} />
            <span className="text-sm text-muted-foreground">启用</span>
          </div>

          <div className="flex items-center justify-end gap-3 pt-2">
            <button
              onClick={() => {
                setShowCreateModal(false)
                resetForm()
              }}
              className="px-4 py-2 border border-border rounded-lg hover:bg-accent transition-colors"
              disabled={saving}
            >
              取消
            </button>
            <button
              onClick={submitCreate}
              disabled={saving}
              className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors disabled:opacity-50"
            >
              {saving ? '保存中...' : '保存'}
            </button>
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title="确认删除模型能力？"
        description={`将删除能力 #${selected?.id || ''}（${selected?.model_id || ''} / ${selected?.operation || ''}）。此操作不可撤销。`}
        confirmText="删除"
        variant="destructive"
        loading={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  )
}

