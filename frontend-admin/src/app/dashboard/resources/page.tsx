'use client'

import { useCallback, useEffect, useState } from 'react'
import { resourceAccountApi, ResourceAccount, getErrorMessage, type UpdateResourceAccountRequest } from '@/lib/api'
import { useToast } from '@/components/ui/use-toast'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import {
  Plus,
  MoreVertical,
  Edit,
  Trash2,
  RefreshCw,
  Database,
  CheckCircle,
  XCircle,
  Clock,
  AlertCircle,
  RotateCcw,
} from 'lucide-react'
import { formatDate, getStatusColor, getStatusText } from '@/lib/utils'

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function normalizeCredentials(input: Record<string, unknown>): Record<string, string> {
  const output: Record<string, string> = {}
  for (const [key, value] of Object.entries(input || {})) {
    if (value === undefined || value === null) continue
    if (typeof value === 'string') {
      output[key] = value
      continue
    }
    if (typeof value === 'number' || typeof value === 'boolean') {
      output[key] = String(value)
      continue
    }
    const serialized = JSON.stringify(value)
    if (typeof serialized === 'string' && serialized !== 'null') {
      output[key] = serialized
    }
  }
  return output
}

function parseCredentialsText(text: string): { ok: true; value: Record<string, unknown> } | { ok: false; error: string } {
  const trimmed = text.trim()
  if (!trimmed) return { ok: false, error: '凭证不能为空' }

  try {
    const parsed: unknown = JSON.parse(trimmed)
    if (!isRecord(parsed)) return { ok: false, error: '凭证必须是 JSON 对象' }
    return { ok: true, value: parsed }
  } catch {
    return { ok: false, error: '凭证 JSON 格式不正确' }
  }
}

export default function ResourcesPage() {
  const { toast } = useToast()
  const [accounts, setAccounts] = useState<ResourceAccount[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [statusFilter, setStatusFilter] = useState('')
  const [channelFilter, setChannelFilter] = useState('')
  const [actionMenuId, setActionMenuId] = useState<number | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [editingAccount, setEditingAccount] = useState<ResourceAccount | null>(null)
  const [refreshingId, setRefreshingId] = useState<number | null>(null)
  const [deletingAccount, setDeletingAccount] = useState<ResourceAccount | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const [formData, setFormData] = useState({
    channel_id: 0,
    account_name: '',
    status: 'active',
    expires_at: '',
  })
  const [credentialsText, setCredentialsText] = useState('{\n  \"username\": \"\",\n  \"password\": \"\"\n}')
  const [originalCredentialsText, setOriginalCredentialsText] = useState('{\n  \"username\": \"\",\n  \"password\": \"\"\n}')
  const [credentialsError, setCredentialsError] = useState<string | null>(null)
  const [credentialsKeyCount, setCredentialsKeyCount] = useState(0)

  const loadAccounts = useCallback(async () => {
    setLoading(true)
    try {
      const res = await resourceAccountApi.list({
        page,
        page_size: pageSize,
        status: statusFilter || undefined,
        channel_id: channelFilter ? parseInt(channelFilter) : undefined,
      })
      if (res.data) {
        setAccounts(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (error) {
      toast({
        title: '加载资源账户失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [channelFilter, page, pageSize, statusFilter, toast])

  useEffect(() => {
    loadAccounts()
  }, [loadAccounts])

  const handleCreate = () => {
    setEditingAccount(null)
    setFormData({
      channel_id: 0,
      account_name: '',
      status: 'active',
      expires_at: '',
    })
    const initialCredentials = '{\n  \"username\": \"\",\n  \"password\": \"\"\n}'
    setCredentialsText(initialCredentials)
    setOriginalCredentialsText(initialCredentials)
    setCredentialsError(null)
    setCredentialsKeyCount(2)
    setShowModal(true)
  }

  const handleEdit = (account: ResourceAccount) => {
    setEditingAccount(account)
    setFormData({
      channel_id: account.channel_id,
      account_name: account.account_name,
      status: account.status,
      expires_at: account.expires_at ? new Date(account.expires_at).toISOString().split('T')[0] : '',
    })
    const text = JSON.stringify(account.credentials || {}, null, 2)
    setCredentialsText(text)
    setOriginalCredentialsText(text)
    setCredentialsError(null)
    setCredentialsKeyCount(Object.keys(account.credentials || {}).length)
    setShowModal(true)
    setActionMenuId(null)
  }

  const handleSubmit = async () => {
    try {
      const credentialsChanged =
        !editingAccount || credentialsText.trim() !== originalCredentialsText.trim()

      let normalizedCredentials: Record<string, string> | undefined
      if (credentialsChanged) {
        const parsed = parseCredentialsText(credentialsText)
        if (!parsed.ok) {
          setCredentialsError(parsed.error)
          toast({ title: '凭证格式错误', description: parsed.error, variant: 'destructive' })
          return
        }

        normalizedCredentials = normalizeCredentials(parsed.value)
        if (Object.keys(normalizedCredentials).length === 0) {
          const message = '凭证不能为空'
          setCredentialsError(message)
          toast({ title: '凭证格式错误', description: message, variant: 'destructive' })
          return
        }
      }

      const base: UpdateResourceAccountRequest = {
        account_name: formData.account_name,
        status: formData.status,
        expires_at: formData.expires_at ? new Date(formData.expires_at) : undefined,
      }

      if (normalizedCredentials) {
        base.credentials = normalizedCredentials
      }

      if (editingAccount) {
        await resourceAccountApi.update(editingAccount.id, base)
      } else {
        await resourceAccountApi.create({
          account_name: formData.account_name,
          status: base.status,
          expires_at: base.expires_at,
          credentials: base.credentials ?? {},
          channel_id: formData.channel_id,
        })
      }
      setShowModal(false)
      toast({ title: '保存成功', description: editingAccount ? '资源账户已更新' : '资源账户已创建' })
      loadAccounts()
    } catch (error) {
      toast({
        title: '保存失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }

  const requestDelete = (account: ResourceAccount) => {
    setDeletingAccount(account)
    setShowDeleteConfirm(true)
    setActionMenuId(null)
  }

  const confirmDelete = useCallback(async () => {
    if (!deletingAccount) return
    setDeleting(true)
    try {
      await resourceAccountApi.delete(deletingAccount.id)
      toast({ title: '删除成功', description: '资源账户已删除' })
      setShowDeleteConfirm(false)
      setDeletingAccount(null)
      loadAccounts()
    } catch (error) {
      toast({
        title: '删除失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setDeleting(false)
    }
  }, [deletingAccount, loadAccounts, toast])

  const handleRefresh = async (id: number) => {
    setRefreshingId(id)
    try {
      await resourceAccountApi.refresh(id)
      toast({ title: '刷新成功', description: '资源账户已刷新' })
      loadAccounts()
    } catch (error) {
      toast({
        title: '刷新失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setRefreshingId(null)
    }
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">资源账户管理</h1>
          <p className="text-muted-foreground">管理逆向工程账户和资源池</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={loadAccounts}
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
            添加账户
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
          <option value="active">活跃</option>
          <option value="inactive">不活跃</option>
          <option value="expired">已过期</option>
          <option value="banned">已封禁</option>
        </select>
        <input
          type="number"
          placeholder="渠道 ID"
          value={channelFilter}
          onChange={(e) => {
            setChannelFilter(e.target.value)
            setPage(1)
          }}
          className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
        />
      </div>

      {/* Accounts Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {loading ? (
          <div className="col-span-full flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-orange-500"></div>
          </div>
        ) : accounts.length === 0 ? (
          <div className="col-span-full text-center py-12 text-muted-foreground">
            暂无资源账户数据
          </div>
        ) : (
          accounts.map((account) => (
            <div
              key={account.id}
              className="bg-card border border-border rounded-xl p-6 hover:shadow-lg transition-shadow"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg ${
                    account.status === 'active' ? 'bg-green-500/10' : 
                    account.status === 'inactive' ? 'bg-gray-500/10' :
                    account.status === 'expired' ? 'bg-yellow-500/10' : 'bg-red-500/10'
                  }`}>
                    <Database className={`w-5 h-5 ${
                      account.status === 'active' ? 'text-green-500' : 
                      account.status === 'inactive' ? 'text-gray-500' :
                      account.status === 'expired' ? 'text-yellow-500' : 'text-red-500'
                    }`} />
                  </div>
                  <div>
                    <h3 className="font-semibold">{account.account_name}</h3>
                    <p className="text-sm text-muted-foreground">渠道 ID: {account.channel_id}</p>
                  </div>
                </div>
                <div className="relative">
                  <button
                    onClick={() => setActionMenuId(actionMenuId === account.id ? null : account.id)}
                    className="p-2 hover:bg-accent rounded-lg transition-colors"
                  >
                    <MoreVertical className="w-4 h-4" />
                  </button>
                  {actionMenuId === account.id && (
                    <div className="absolute right-0 top-full mt-1 w-40 bg-card border border-border rounded-lg shadow-lg py-1 z-10">
                      <button
                        onClick={() => handleEdit(account)}
                        className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                      >
                        <Edit className="w-4 h-4" />
                        编辑
                      </button>
                      <button
                        onClick={() => handleRefresh(account.id)}
                        disabled={refreshingId === account.id}
                        className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent disabled:opacity-50"
                      >
                        <RotateCcw className={`w-4 h-4 ${refreshingId === account.id ? 'animate-spin' : ''}`} />
                        刷新
                      </button>
                      <button
                        onClick={() => requestDelete(account)}
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
                <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(account.status)}`}>
                  {getStatusText(account.status)}
                </span>
                {account.error_count > 0 && (
                  <span className="px-2 py-1 text-xs rounded-full bg-red-500/10 text-red-500 flex items-center gap-1">
                    <AlertCircle className="w-3 h-3" />
                    {account.error_count} 错误
                  </span>
                )}
              </div>

              {/* Info */}
              <div className="space-y-2 mb-4">
                <div className="flex items-center gap-2 text-sm">
                  <Clock className="w-4 h-4 text-muted-foreground" />
                  <span className="text-muted-foreground">最后活跃:</span>
                  <span>{account.last_active_at ? formatDate(account.last_active_at) : '从未'}</span>
                </div>
                {account.expires_at && (
                  <div className="flex items-center gap-2 text-sm">
                    <XCircle className="w-4 h-4 text-muted-foreground" />
                    <span className="text-muted-foreground">过期时间:</span>
                    <span>{formatDate(account.expires_at)}</span>
                  </div>
                )}
              </div>

              {/* Error */}
              {account.last_error && (
                <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
                  <p className="text-xs text-red-500 truncate" title={account.last_error}>
                    {account.last_error}
                  </p>
                </div>
              )}

              {/* Stats */}
              <div className="grid grid-cols-2 gap-4 pt-4 border-t border-border">
                <div className="text-center">
                  <p className="text-lg font-semibold">{account.error_count}</p>
                  <p className="text-xs text-muted-foreground">错误次数</p>
                </div>
                <div className="text-center">
                  <p className="text-lg font-semibold">
                    {account.status === 'active' ? (
                      <CheckCircle className="w-5 h-5 text-green-500 mx-auto" />
                    ) : (
                      <XCircle className="w-5 h-5 text-red-500 mx-auto" />
                    )}
                  </p>
                  <p className="text-xs text-muted-foreground">当前状态</p>
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
            共 {total} 个账户，第 {page} / {totalPages} 页
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
          if (!open) setDeletingAccount(null)
        }}
        title="确认删除资源账户？"
        description={
          deletingAccount
            ? `将删除资源账户「${deletingAccount.account_name}」（ID=${deletingAccount.id}）。此操作不可撤销。`
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
              {editingAccount ? '编辑资源账户' : '添加资源账户'}
            </h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">渠道 ID</label>
                <input
                  type="number"
                  value={formData.channel_id}
                  onChange={(e) => setFormData({ ...formData, channel_id: parseInt(e.target.value) || 0 })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  placeholder="输入渠道 ID"
                  disabled={!!editingAccount}
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">账户名称</label>
                <input
                  type="text"
                  value={formData.account_name}
                  onChange={(e) => setFormData({ ...formData, account_name: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  placeholder="例如：Jimeng Account 1"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">凭证 (JSON)</label>
                <textarea
                  value={credentialsText}
                  onChange={(e) => {
                    const next = e.target.value
                    setCredentialsText(next)

                    const parsed = parseCredentialsText(next)
                    if (!parsed.ok) {
                      setCredentialsError(parsed.error)
                      setCredentialsKeyCount(0)
                      return
                    }

                    const normalized = normalizeCredentials(parsed.value)
                    const keyCount = Object.keys(normalized).length
                    setCredentialsKeyCount(keyCount)
                    setCredentialsError(keyCount === 0 ? '凭证不能为空' : null)
                  }}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500 font-mono text-sm"
                  rows={6}
                  placeholder='{"username": "", "password": ""}'
                />
                {credentialsError && (
                  <p className="mt-2 text-sm text-red-500">{credentialsError}</p>
                )}
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">状态</label>
                <select
                  value={formData.status}
                  onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                >
                  <option value="active">活跃</option>
                  <option value="inactive">不活跃</option>
                  <option value="expired">已过期</option>
                  <option value="banned">已封禁</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">过期时间（可选）</label>
                <input
                  type="date"
                  value={formData.expires_at}
                  onChange={(e) => setFormData({ ...formData, expires_at: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
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
                  !formData.account_name ||
                  !formData.channel_id ||
                  !!credentialsError ||
                  (!editingAccount && credentialsKeyCount === 0)
                }
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-500/50 text-white rounded-lg"
              >
                {editingAccount ? '保存' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
