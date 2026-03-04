'use client'

import { useCallback, useEffect, useState } from 'react'
import { channelApi, Channel, ChannelTestResult, getErrorMessage, type CreateChannelRequest } from '@/lib/api'
import { useToast } from '@/components/ui/use-toast'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { useRouter, useSearchParams } from 'next/navigation'
import {
  Plus,
  MoreVertical,
  Eye,
  Edit,
  Trash2,
  Play,
  Pause,
  TestTube,
  RefreshCw,
  Server,
} from 'lucide-react'
import { getStatusColor, getStatusText } from '@/lib/utils'

export default function ChannelsPage() {
  const { toast } = useToast()
  const router = useRouter()
  const searchParams = useSearchParams()
  const editId = searchParams.get('edit')
  const [channels, setChannels] = useState<Channel[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(10)
  const [statusFilter, setStatusFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [actionMenuId, setActionMenuId] = useState<number | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null)
  const [deletingChannel, setDeletingChannel] = useState<Channel | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [testingId, setTestingId] = useState<number | null>(null)
  const [testResult, setTestResult] = useState<(ChannelTestResult & { id: number }) | null>(null)

  // Form state
  const [formData, setFormData] = useState({
    name: '',
    type: 'official',
    base_url: '',
    api_key: '',
    models: '',
    weight: 1,
    priority: 0,
    retry_count: 3,
    timeout_seconds: 30,
    rate_limit: 100,
    max_concurrent: 100,
  })

  const loadChannels = useCallback(async () => {
    setLoading(true)
    try {
      const res = await channelApi.list({
        page,
        page_size: pageSize,
        status: statusFilter || undefined,
        type: typeFilter || undefined,
      })
      if (res.data) {
        setChannels(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (error) {
      toast({
        title: '加载渠道列表失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, statusFilter, toast, typeFilter])

  const handleCreate = useCallback(() => {
    setEditingChannel(null)
    setFormData({
      name: '',
      type: 'official',
      base_url: '',
      api_key: '',
      models: '',
      weight: 1,
      priority: 0,
      retry_count: 3,
      timeout_seconds: 30,
      rate_limit: 100,
      max_concurrent: 100,
    })
    setShowModal(true)
  }, [])

  const handleEdit = useCallback((channel: Channel) => {
    setEditingChannel(channel)
    setFormData({
      name: channel.name,
      type: channel.type,
      base_url: channel.base_url,
      api_key: '',
      models: channel.models.join(', '),
      weight: channel.weight,
      priority: channel.priority,
      retry_count: channel.retry_count ?? 3,
      timeout_seconds: Math.max(1, Math.round((channel.timeout_ms ?? 30000) / 1000)),
      rate_limit: channel.rate_limit,
      max_concurrent: channel.max_concurrent ?? 100,
    })
    setShowModal(true)
    setActionMenuId(null)
  }, [])

  useEffect(() => {
    loadChannels()
  }, [loadChannels])

  useEffect(() => {
    if (!editId) return

    const channelId = Number(editId)
    if (!Number.isFinite(channelId) || channelId <= 0) {
      router.replace('/dashboard/channels')
      return
    }

    const open = async () => {
      try {
        const res = await channelApi.get(channelId)
        if (res.data) {
          handleEdit(res.data)
        }
      } catch (error) {
        toast({
          title: '加载渠道失败',
          description: getErrorMessage(error),
          variant: 'destructive',
        })
      } finally {
        router.replace('/dashboard/channels')
      }
    }

    open()
  }, [editId, handleEdit, router, toast])

  const handleSubmit = async () => {
    try {
      const models = formData.models.split(',').map((m) => m.trim()).filter(Boolean)
      const apiKey = formData.api_key.trim()
      const payload: CreateChannelRequest = {
        name: formData.name.trim(),
        type: formData.type,
        base_url: formData.base_url.trim(),
        ...(apiKey ? { api_key: apiKey } : {}),
        models,
        weight: formData.weight,
        priority: formData.priority,
        retry_count: Math.max(0, Math.min(10, formData.retry_count)),
        timeout_ms: Math.max(1000, Math.round(formData.timeout_seconds * 1000)),
        rate_limit: Math.max(1, formData.rate_limit),
        max_concurrent: Math.max(1, formData.max_concurrent),
      }

      if (editingChannel) {
        await channelApi.update(editingChannel.id, payload)
      } else {
        await channelApi.create(payload)
      }
      setShowModal(false)
      toast({ title: '保存成功', description: editingChannel ? '渠道已更新' : '渠道已创建' })
      loadChannels()
    } catch (error) {
      toast({
        title: '保存失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }

  const requestDelete = (channel: Channel) => {
    setDeletingChannel(channel)
    setShowDeleteConfirm(true)
    setActionMenuId(null)
  }

  const confirmDelete = useCallback(async () => {
    if (!deletingChannel) return
    setDeleting(true)
    try {
      await channelApi.delete(deletingChannel.id)
      toast({ title: '删除成功', description: '渠道已删除' })
      setShowDeleteConfirm(false)
      setDeletingChannel(null)
      loadChannels()
    } catch (error) {
      toast({
        title: '删除失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setDeleting(false)
    }
  }, [deletingChannel, loadChannels, toast])

  const handleStatusChange = async (id: number, newStatus: string) => {
    try {
      await channelApi.updateStatus(id, newStatus)
      toast({ title: '更新成功', description: '渠道状态已更新' })
      loadChannels()
    } catch (error) {
      toast({
        title: '更新状态失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
    setActionMenuId(null)
  }

  const handleTest = async (id: number) => {
    setTestingId(id)
    setTestResult(null)
    try {
      const res = await channelApi.test(id)
      if (res.data) {
        setTestResult({ id, ...res.data })
      }
    } catch (error) {
      toast({
        title: '测试失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
      setTestResult({ id, success: false, status: 'error', latency: 0, message: '测试失败' })
    } finally {
      setTestingId(null)
    }
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">渠道管理</h1>
          <p className="text-muted-foreground">管理上游 API 渠道</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={loadChannels}
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
            添加渠道
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <select
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value)
            setPage(1)
          }}
          className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
        >
          <option value="">全部状态</option>
          <option value="healthy">健康</option>
          <option value="down">故障</option>
          <option value="disabled">禁用</option>
        </select>
        <select
          value={typeFilter}
          onChange={(e) => {
            setTypeFilter(e.target.value)
            setPage(1)
          }}
          className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
        >
          <option value="">全部类型</option>
          <option value="official">官方 API</option>
          <option value="reverse_engineered">逆向接口</option>
          <option value="proxy">代理</option>
        </select>
      </div>

      {/* Channels Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {loading ? (
          <div className="col-span-full flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-orange-500"></div>
          </div>
        ) : channels.length === 0 ? (
          <div className="col-span-full text-center py-12 text-muted-foreground">
            暂无渠道数据
          </div>
        ) : (
          channels.map((channel) => (
            <div
              key={channel.id}
              className="bg-card border border-border rounded-xl p-6 hover:shadow-lg transition-shadow"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg ${
                    channel.status === 'healthy' ? 'bg-green-500/10' : 
                    channel.status === 'down' ? 'bg-red-500/10' : 'bg-gray-500/10'
                  }`}>
                    <Server className={`w-5 h-5 ${
                      channel.status === 'healthy' ? 'text-green-500' : 
                      channel.status === 'down' ? 'text-red-500' : 'text-gray-500'
                    }`} />
                  </div>
                  <div>
                    <h3 className="font-semibold">{channel.name}</h3>
                    <p className="text-sm text-muted-foreground">
                      {channel.type === 'official' ? '官方 API' : channel.type === 'reverse_engineered' ? '逆向接口' : channel.type === 'proxy' ? '代理' : channel.type}
                    </p>
                  </div>
                </div>
                <div className="relative">
                  <button
                    onClick={() => setActionMenuId(actionMenuId === channel.id ? null : channel.id)}
                    className="p-2 hover:bg-accent rounded-lg transition-colors"
                  >
                    <MoreVertical className="w-4 h-4" />
                  </button>
                  {actionMenuId === channel.id && (
                    <div className="absolute right-0 top-full mt-1 w-40 bg-card border border-border rounded-lg shadow-lg py-1 z-10">
                      <button
                        onClick={() => {
                          setActionMenuId(null)
                          router.push(`/dashboard/channels/${channel.id}`)
                        }}
                        className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                      >
                        <Eye className="w-4 h-4" />
                        查看详情
                      </button>
                      <button
                        onClick={() => handleEdit(channel)}
                        className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                      >
                        <Edit className="w-4 h-4" />
                        编辑
                      </button>
                      <button
                        onClick={() => handleTest(channel.id)}
                        className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                      >
                        <TestTube className="w-4 h-4" />
                        测试
                      </button>
                      {channel.status === 'healthy' ? (
                        <button
                          onClick={() => handleStatusChange(channel.id, 'disabled')}
                          className="w-full flex items-center gap-2 px-4 py-2 text-sm text-yellow-500 hover:bg-accent"
                        >
                          <Pause className="w-4 h-4" />
                          禁用
                        </button>
                      ) : (
                        <button
                          onClick={() => handleStatusChange(channel.id, 'healthy')}
                          className="w-full flex items-center gap-2 px-4 py-2 text-sm text-green-500 hover:bg-accent"
                        >
                          <Play className="w-4 h-4" />
                          启用
                        </button>
                      )}
                      <button
                        onClick={() => requestDelete(channel)}
                        className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-500 hover:bg-accent"
                      >
                        <Trash2 className="w-4 h-4" />
                        删除
                      </button>
                    </div>
                  )}
                </div>
              </div>

              {/* Status */}
              <div className="flex items-center gap-2 mb-4">
                <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(channel.status)}`}>
                  {getStatusText(channel.status)}
                </span>
                {testResult?.id === channel.id && (
                  <span className={`px-2 py-1 text-xs rounded-full ${
                    testResult.status === 'success' ? 'bg-green-500/10 text-green-500' : 'bg-red-500/10 text-red-500'
                  }`}>
                    {testResult.status === 'success' ? `${testResult.latency}ms` : '测试失败'}
                  </span>
                )}
                {testingId === channel.id && (
                  <span className="px-2 py-1 text-xs rounded-full bg-blue-500/10 text-blue-500">
                    测试中...
                  </span>
                )}
              </div>

              {/* Models */}
              <div className="mb-4">
                <p className="text-sm text-muted-foreground mb-2">支持模型</p>
                <div className="flex flex-wrap gap-1">
                  {channel.models.slice(0, 3).map((model) => (
                    <span
                      key={model}
                      className="px-2 py-1 text-xs bg-muted rounded-md"
                    >
                      {model}
                    </span>
                  ))}
                  {channel.models.length > 3 && (
                    <span className="px-2 py-1 text-xs bg-muted rounded-md">
                      +{channel.models.length - 3}
                    </span>
                  )}
                </div>
              </div>

              {/* Stats */}
              <div className="grid grid-cols-3 gap-4 pt-4 border-t border-border">
                <div className="text-center">
                  <p className="text-lg font-semibold">{channel.weight}</p>
                  <p className="text-xs text-muted-foreground">权重</p>
                </div>
                <div className="text-center">
                  <p className="text-lg font-semibold">{channel.priority}</p>
                  <p className="text-xs text-muted-foreground">优先级</p>
                </div>
                <div className="text-center">
                  <p className="text-lg font-semibold">{channel.rate_limit}</p>
                  <p className="text-xs text-muted-foreground">限速/分</p>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            共 {total} 个渠道，第 {page} / {totalPages} 页
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

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={(open) => {
          setShowDeleteConfirm(open)
          if (!open) setDeletingChannel(null)
        }}
        title="确认删除渠道？"
        description={
          deletingChannel
            ? `将删除渠道「${deletingChannel.name}」（ID=${deletingChannel.id}）。此操作不可撤销。`
            : undefined
        }
        confirmText="删除"
        variant="destructive"
        loading={deleting}
        onConfirm={confirmDelete}
      />

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <h3 className="text-lg font-semibold mb-6">
              {editingChannel ? '编辑渠道' : '添加渠道'}
            </h3>
            <div className="grid grid-cols-2 gap-4">
              <div className="col-span-2">
                <label className="block text-sm font-medium mb-2">渠道名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  placeholder="例如：Azure OpenAI 1"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">渠道类型</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                >
                  <option value="official">官方 API</option>
                  <option value="reverse_engineered">逆向接口</option>
                  <option value="proxy">代理</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">权重</label>
                <input
                  type="number"
                  value={formData.weight}
                  onChange={(e) => {
                    const parsed = parseInt(e.target.value, 10)
                    setFormData({ ...formData, weight: Number.isFinite(parsed) ? parsed : 1 })
                  }}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  min="0"
                />
              </div>
              <div className="col-span-2">
                <label className="block text-sm font-medium mb-2">Base URL</label>
                <input
                  type="text"
                  value={formData.base_url}
                  onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  placeholder="https://api.openai.com/v1"
                />
              </div>
              <div className="col-span-2">
                <label className="block text-sm font-medium mb-2">API Key</label>
                <input
                  type="password"
                  value={formData.api_key}
                  onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  placeholder="sk-..."
                />
              </div>
              <div className="col-span-2">
                <label className="block text-sm font-medium mb-2">支持模型（逗号分隔）</label>
                <input
                  type="text"
                  value={formData.models}
                  onChange={(e) => setFormData({ ...formData, models: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  placeholder="gpt-4, gpt-3.5-turbo, claude-3-opus"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">优先级</label>
                <input
                  type="number"
                  value={formData.priority}
                  onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 0 })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">最大重试次数</label>
                <input
                  type="number"
                  value={formData.retry_count}
                  onChange={(e) => {
                    const parsed = parseInt(e.target.value, 10)
                    setFormData({ ...formData, retry_count: Number.isFinite(parsed) ? parsed : 3 })
                  }}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  min="0"
                  max="10"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">超时时间（秒）</label>
                <input
                  type="number"
                  value={formData.timeout_seconds}
                  onChange={(e) => setFormData({ ...formData, timeout_seconds: parseInt(e.target.value) || 30 })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  min="1"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">速率限制（次/分）</label>
                <input
                  type="number"
                  value={formData.rate_limit}
                  onChange={(e) => setFormData({ ...formData, rate_limit: parseInt(e.target.value) || 100 })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  min="1"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">最大并发</label>
                <input
                  type="number"
                  value={formData.max_concurrent}
                  onChange={(e) => setFormData({ ...formData, max_concurrent: parseInt(e.target.value) || 100 })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  min="1"
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
                disabled={!formData.name || !formData.base_url}
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-500/50 text-white rounded-lg"
              >
                {editingChannel ? '保存' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
