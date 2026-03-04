'use client'

import { useCallback, useEffect, useState } from 'react'
import { userApi, User, getErrorMessage } from '@/lib/api'
import { useToast } from '@/components/ui/use-toast'
import { useRouter } from 'next/navigation'
import {
  Search,
  MoreVertical,
  Ban,
  CheckCircle,
  DollarSign,
  Eye,
  RefreshCw,
} from 'lucide-react'
import { formatDate, formatCurrency, getStatusColor, getStatusText } from '@/lib/utils'

export default function UsersPage() {
  const { toast } = useToast()
  const router = useRouter()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(10)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [showBalanceModal, setShowBalanceModal] = useState(false)
  const [balanceAmount, setBalanceAmount] = useState('')
  const [balanceReason, setBalanceReason] = useState('')
  const [actionMenuId, setActionMenuId] = useState<string | null>(null)

  const loadUsers = useCallback(async () => {
    setLoading(true)
    try {
      const res = await userApi.list({
        page,
        page_size: pageSize,
        keyword: search || undefined,
        status: statusFilter || undefined,
      })
      if (res.data) {
        setUsers(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (error) {
      toast({
        title: '加载用户失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, search, statusFilter, toast])

  useEffect(() => {
    loadUsers()
  }, [loadUsers])

  const handleStatusChange = async (userId: string, newStatus: string) => {
    try {
      await userApi.updateStatus(userId, newStatus)
      toast({ title: '更新成功', description: '用户状态已更新' })
      loadUsers()
    } catch (error) {
      toast({
        title: '更新状态失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
    setActionMenuId(null)
  }

  const handleBalanceUpdate = async () => {
    if (!selectedUser || !balanceAmount) return
    const amount = Number(balanceAmount)
    if (!Number.isFinite(amount) || amount === 0) {
      toast({ title: '金额不合法', description: '请输入有效的调整金额（可为正或负）', variant: 'destructive' })
      return
    }
    try {
      await userApi.updateBalance(
        selectedUser.id,
        amount,
        balanceReason || '管理员调整'
      )
      setShowBalanceModal(false)
      setBalanceAmount('')
      setBalanceReason('')
      setSelectedUser(null)
      toast({ title: '调整成功', description: '余额已更新' })
      loadUsers()
    } catch (error) {
      toast({
        title: '调整失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">用户管理</h1>
          <p className="text-muted-foreground">管理系统用户账户</p>
        </div>
        <button
          onClick={loadUsers}
          className="flex items-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
        >
          <RefreshCw className="w-4 h-4" />
          刷新
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="搜索用户名或邮箱..."
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
          <option value="active">正常</option>
          <option value="disabled">禁用</option>
        </select>
      </div>

      {/* Users Table */}
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-muted/50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  用户
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  邮箱
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  余额
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  角色
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  注册时间
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
                      <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-orange-500"></div>
                    </div>
                  </td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-12 text-center text-muted-foreground">
                    暂无用户数据
                  </td>
                </tr>
              ) : (
                users.map((user) => (
                  <tr key={user.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-orange-500 rounded-full flex items-center justify-center text-white font-medium">
                          {user.username.charAt(0).toUpperCase()}
                        </div>
                        <div>
                          <div className="font-medium">{user.username}</div>
                          <div className="text-sm text-muted-foreground">ID: {user.id.slice(0, 8)}...</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {user.email}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="font-medium text-green-500">
                        {formatCurrency(user.balance)}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 text-xs rounded-full ${
                        user.role === 'admin' 
                          ? 'bg-purple-500/10 text-purple-500' 
                          : 'bg-blue-500/10 text-blue-500'
                      }`}>
                        {user.role === 'admin' ? '管理员' : '普通用户'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(user.status)}`}>
                        {getStatusText(user.status)}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground">
                      {formatDate(user.created_at)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right">
                      <div className="relative">
                        <button
                          onClick={() => setActionMenuId(actionMenuId === user.id ? null : user.id)}
                          className="p-2 hover:bg-accent rounded-lg transition-colors"
                        >
                          <MoreVertical className="w-4 h-4" />
                        </button>
                        {actionMenuId === user.id && (
                          <div className="absolute right-0 top-full mt-1 w-48 bg-card border border-border rounded-lg shadow-lg py-1 z-10">
                            <button
                              onClick={() => {
                                setActionMenuId(null)
                                router.push(`/dashboard/users/${user.id}`)
                              }}
                              className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                            >
                              <Eye className="w-4 h-4" />
                              查看详情
                            </button>
                            <button
                              onClick={() => {
                                setSelectedUser(user)
                                setShowBalanceModal(true)
                                setActionMenuId(null)
                              }}
                              className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                            >
                              <DollarSign className="w-4 h-4" />
                              调整余额
                            </button>
                            {user.status === 'active' ? (
                              <button
                                onClick={() => handleStatusChange(user.id, 'disabled')}
                                className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-500 hover:bg-accent"
                              >
                                <Ban className="w-4 h-4" />
                                禁用账户
                              </button>
                            ) : (
                              <button
                                onClick={() => handleStatusChange(user.id, 'active')}
                                className="w-full flex items-center gap-2 px-4 py-2 text-sm text-green-500 hover:bg-accent"
                              >
                                <CheckCircle className="w-4 h-4" />
                                启用账户
                              </button>
                            )}
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

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-6 py-4 border-t border-border">
            <div className="text-sm text-muted-foreground">
              共 {total} 条记录，第 {page} / {totalPages} 页
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

      {/* Balance Modal */}
      {showBalanceModal && selectedUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-xl p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">调整用户余额</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">用户</label>
                <div className="px-4 py-2 bg-muted rounded-lg">
                  {selectedUser.username} ({selectedUser.email})
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">当前余额</label>
                <div className="px-4 py-2 bg-muted rounded-lg text-green-500 font-medium">
                  {formatCurrency(selectedUser.balance)}
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">调整金额</label>
                <input
                  type="number"
                  step="0.01"
                  value={balanceAmount}
                  onChange={(e) => setBalanceAmount(e.target.value)}
                  placeholder="正数增加，负数减少"
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">调整原因</label>
                <input
                  type="text"
                  value={balanceReason}
                  onChange={(e) => setBalanceReason(e.target.value)}
                  placeholder="可选"
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => {
                  setShowBalanceModal(false)
                  setSelectedUser(null)
                  setBalanceAmount('')
                  setBalanceReason('')
                }}
                className="px-4 py-2 border border-border rounded-lg hover:bg-accent"
              >
                取消
              </button>
              <button
                onClick={handleBalanceUpdate}
                disabled={!balanceAmount}
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-500/50 text-white rounded-lg"
              >
                确认调整
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
