'use client'

import { useCallback, useEffect, useState } from 'react'
import { CreateModelRequest, modelApi, Model, UpdateModelRequest, getErrorMessage } from '@/lib/api'
import { useToast } from '@/components/ui/use-toast'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import {
  Plus,
  Search,
  MoreVertical,
  Edit,
  Trash2,
  RefreshCw,
  Cpu,
  Play,
  Pause,
} from 'lucide-react'
import { getStatusColor, getStatusText } from '@/lib/utils'

export default function ModelsPage() {
  const { toast } = useToast()
  const [models, setModels] = useState<Model[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [actionMenuId, setActionMenuId] = useState<string | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [editingModel, setEditingModel] = useState<Model | null>(null)
  const [deletingModel, setDeletingModel] = useState<Model | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const [formData, setFormData] = useState({
    name: '',
    display_name: '',
    provider: '',
    input_price: 0,
    output_price: 0,
    max_context: 4096,
    max_tokens: 4096,
    description: '',
  })

  const loadModels = useCallback(async () => {
    setLoading(true)
    try {
      const res = await modelApi.list({
        page,
        page_size: pageSize,
        enabled: statusFilter === 'enabled' ? true : statusFilter === 'disabled' ? false : undefined,
        keyword: search.trim() || undefined,
      })
      if (res.data) {
        setModels(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (error) {
      toast({
        title: '加载模型失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, search, statusFilter, toast])

  useEffect(() => {
    loadModels()
  }, [loadModels])

  const handleCreate = () => {
    setEditingModel(null)
    setFormData({
      name: '',
      display_name: '',
      provider: '',
      input_price: 0,
      output_price: 0,
      max_context: 4096,
      max_tokens: 4096,
      description: '',
    })
    setShowModal(true)
  }

  const handleEdit = (model: Model) => {
    setEditingModel(model)
    setFormData({
      name: model.name,
      display_name: model.display_name,
      provider: model.provider,
      input_price: model.input_price,
      output_price: model.output_price,
      max_context: model.max_context,
      max_tokens: model.max_tokens,
      description: model.description || '',
    })
    setShowModal(true)
    setActionMenuId(null)
  }

  const handleSubmit = async () => {
    const name = formData.name.trim()
    const displayName = formData.display_name.trim()
    const provider = formData.provider.trim()

    if (!displayName) {
      toast({ title: '显示名称不能为空', variant: 'destructive' })
      return
    }

    if (!editingModel) {
      if (!name) {
        toast({ title: '模型标识符不能为空', variant: 'destructive' })
        return
      }
      if (!provider) {
        toast({ title: '提供商不能为空', variant: 'destructive' })
        return
      }
    }

    try {
      if (editingModel) {
        const payload: UpdateModelRequest = {
          display_name: displayName,
          input_price: Number(formData.input_price),
          output_price: Number(formData.output_price),
          max_context: formData.max_context,
          max_tokens: formData.max_tokens,
          description: formData.description,
        }
        await modelApi.update(editingModel.id, payload)
      } else {
        const payload: CreateModelRequest = {
          name,
          display_name: displayName,
          provider,
          input_price: Number(formData.input_price),
          output_price: Number(formData.output_price),
          max_context: formData.max_context,
          max_tokens: formData.max_tokens,
          description: formData.description,
        }
        await modelApi.create(payload)
      }
      setShowModal(false)
      toast({ title: '保存成功', description: editingModel ? '模型已更新' : '模型已创建' })
      loadModels()
    } catch (error) {
      toast({
        title: '保存失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }

  const requestDelete = (model: Model) => {
    setDeletingModel(model)
    setShowDeleteConfirm(true)
    setActionMenuId(null)
  }

  const confirmDelete = useCallback(async () => {
    if (!deletingModel) return
    setDeleting(true)
    try {
      await modelApi.delete(deletingModel.id)
      toast({ title: '删除成功', description: '模型已删除' })
      setShowDeleteConfirm(false)
      setDeletingModel(null)
      loadModels()
    } catch (error) {
      toast({
        title: '删除失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setDeleting(false)
    }
  }, [deletingModel, loadModels, toast])

  const handleToggleEnabled = async (model: Model) => {
    try {
      await modelApi.updateStatus(model.id, !model.enabled)
      toast({ title: '更新成功', description: model.enabled ? '已禁用' : '已启用' })
      loadModels()
    } catch (error) {
      toast({
        title: '更新状态失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
    setActionMenuId(null)
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">模型管理</h1>
          <p className="text-muted-foreground">管理可用的 AI 模型和定价</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={loadModels}
            className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg hover:bg-accent transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
            刷新
          </button>
          <button
            onClick={handleCreate}
            className="flex items-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
            添加模型
          </button>
        </div>
      </div>

      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="搜索模型名称 / ID / 提供商..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(1)
            }}
            className="w-full pl-10 pr-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value)
            setPage(1)
          }}
          className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
        >
          <option value="">全部状态</option>
          <option value="enabled">已启用</option>
          <option value="disabled">已禁用</option>
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
                  提供商
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  输入价格
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  输出价格
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  上下文长度
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  功能
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
                  <td colSpan={8} className="px-6 py-12 text-center">
                    <div className="flex items-center justify-center">
                      <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-orange-500"></div>
                    </div>
                  </td>
                </tr>
              ) : models.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-6 py-12 text-center text-muted-foreground">
                    暂无模型数据
                  </td>
                </tr>
              ) : (
                models.map((model) => (
                  <tr key={model.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-blue-500/10 rounded-lg">
                          <Cpu className="w-5 h-5 text-blue-500" />
                        </div>
                        <div>
                          <div className="font-medium">{model.display_name}</div>
                          <div className="text-sm text-muted-foreground">{model.name}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="px-2 py-1 text-xs bg-purple-500/10 text-purple-500 rounded-md">
                        {model.provider}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      ¥{Number(model.input_price).toFixed(4)}/1K
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      ¥{Number(model.output_price).toFixed(4)}/1K
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {Number(model.max_context / 1000).toFixed(0)}K
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center gap-2">
                        {model.enabled ? (
                          <span className="px-2 py-1 text-xs bg-green-500/10 text-green-500 rounded-md">
                            已启用
                          </span>
                        ) : (
                          <span className="px-2 py-1 text-xs bg-muted text-muted-foreground rounded-md">
                            已禁用
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(model.status)}`}>
                        {getStatusText(model.status)}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right">
                      <div className="relative">
                        <button
                          onClick={() => setActionMenuId(actionMenuId === model.id ? null : model.id)}
                          className="p-2 hover:bg-accent rounded-lg transition-colors"
                        >
                          <MoreVertical className="w-4 h-4" />
                        </button>
                        {actionMenuId === model.id && (
                          <div className="absolute right-0 top-full mt-1 w-32 bg-card border border-border rounded-lg shadow-lg py-1 z-10">
                            <button
                              onClick={() => handleToggleEnabled(model)}
                              className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                            >
                              {model.enabled ? (
                                <>
                                  <Pause className="w-4 h-4" />
                                  禁用
                                </>
                              ) : (
                                <>
                                  <Play className="w-4 h-4" />
                                  启用
                                </>
                              )}
                            </button>
                            <button
                              onClick={() => handleEdit(model)}
                              className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                            >
                              <Edit className="w-4 h-4" />
                              编辑
                            </button>
                            <button
                              onClick={() => requestDelete(model)}
                              className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-500 hover:bg-accent"
                            >
                              <Trash2 className="w-4 h-4" />
                              删除
                            </button>
                          </div>
                        )}
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {totalPages > 1 && (
          <div className="flex items-center justify-between px-6 py-4 border-t border-border">
            <div className="text-sm text-muted-foreground">
              共 {total} 个模型，第 {page} / {totalPages} 页
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
                className="px-3 py-1 border border-border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent"
              >
                上一页
              </button>
              <button
                onClick={() => setPage(Math.min(totalPages, page + 1))}
                disabled={page === totalPages}
                className="px-3 py-1 border border-border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent"
              >
                下一页
              </button>
            </div>
          </div>
        )}
      </div>

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={(open) => {
          setShowDeleteConfirm(open)
          if (!open) setDeletingModel(null)
        }}
        title="确认删除模型？"
        description={
          deletingModel
            ? `将删除模型「${deletingModel.display_name}」（${deletingModel.name}）。此操作不可撤销。`
            : undefined
        }
        confirmText="删除"
        variant="destructive"
        loading={deleting}
        onConfirm={confirmDelete}
      />

      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <h3 className="text-lg font-semibold mb-6">
              {editingModel ? '编辑模型' : '添加模型'}
            </h3>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2">模型标识符</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  placeholder="gpt-4"
                  disabled={!!editingModel}
                />
                {editingModel && (
                  <p className="mt-1 text-xs text-muted-foreground">模型标识符创建后不可修改</p>
                )}
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">显示名称</label>
                <input
                  type="text"
                  value={formData.display_name}
                  onChange={(e) => setFormData({ ...formData, display_name: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  placeholder="GPT-4"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">提供商</label>
                <input
                  type="text"
                  value={formData.provider}
                  onChange={(e) => setFormData({ ...formData, provider: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  placeholder="OpenAI"
                  disabled={!!editingModel}
                />
                {editingModel && (
                  <p className="mt-1 text-xs text-muted-foreground">提供商创建后不可修改</p>
                )}
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">最大上下文</label>
                <input
                  type="number"
                  min="1"
                  value={formData.max_context}
                  onChange={(e) => setFormData({ ...formData, max_context: parseInt(e.target.value) || 4096 })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">输入价格 (¥/1K tokens)</label>
                <input
                  type="number"
                  min="0"
                  step="0.0001"
                  value={formData.input_price}
                  onChange={(e) => setFormData({ ...formData, input_price: parseFloat(e.target.value) || 0 })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">输出价格 (¥/1K tokens)</label>
                <input
                  type="number"
                  min="0"
                  step="0.0001"
                  value={formData.output_price}
                  onChange={(e) => setFormData({ ...formData, output_price: parseFloat(e.target.value) || 0 })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">最大输出 Token</label>
                <input
                  type="number"
                  min="1"
                  value={formData.max_tokens}
                  onChange={(e) => setFormData({ ...formData, max_tokens: parseInt(e.target.value) || 4096 })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
              </div>
              <div className="col-span-2">
                <label className="block text-sm font-medium mb-2">描述</label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  rows={3}
                  placeholder="模型描述..."
                />
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowModal(false)}
                className="px-4 py-2 border border-border rounded-lg hover:bg-accent"
              >
                取消
              </button>
              <button
                onClick={handleSubmit}
                disabled={
                  !formData.display_name.trim() ||
                  (!editingModel && (!formData.name.trim() || !formData.provider.trim()))
                }
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-500/50 text-white rounded-lg"
              >
                {editingModel ? '保存' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
