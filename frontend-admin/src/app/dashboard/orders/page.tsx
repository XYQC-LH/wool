'use client'

import { useCallback, useEffect, useState } from 'react'
import { orderApi, Order, getErrorMessage } from '@/lib/api'
import { useToast } from '@/components/ui/use-toast'
import {
  Search,
  RefreshCw,
  CreditCard,
  DollarSign,
  CheckCircle,
  XCircle,
  Clock,
  MoreVertical,
  Eye,
  Download,
} from 'lucide-react'
import { formatDate, formatCurrency, getStatusColor, getStatusText } from '@/lib/utils'
import { exportData, formatDateForExport, formatCurrencyForExport, formatStatusForExport } from '@/lib/export'

export default function OrdersPage() {
  const { toast } = useToast()
  const [orders, setOrders] = useState<Order[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [filters, setFilters] = useState({
    status: '',
    user_id: '',
    payment_method: '',
  })
  const [actionMenuId, setActionMenuId] = useState<string | null>(null)
  const [showDetailModal, setShowDetailModal] = useState(false)
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null)

  const loadOrders = useCallback(async () => {
    setLoading(true)
    try {
      const res = await orderApi.list({
        page,
        page_size: pageSize,
        status: filters.status || undefined,
        user_id: filters.user_id || undefined,
        payment_method: filters.payment_method || undefined,
      })
      if (res.data) {
        setOrders(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (error) {
      toast({
        title: '加载订单失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [filters.payment_method, filters.status, filters.user_id, page, pageSize, toast])

  useEffect(() => {
    loadOrders()
  }, [loadOrders])

  const handleSearch = () => {
    if (page === 1) {
      loadOrders()
      return
    }
    setPage(1)
  }

  const handleExport = async () => {
    try {
      // 获取所有数据（不分页）
      const res = await orderApi.list({
        page: 1,
        page_size: 10000, // 获取大量数据
        status: filters.status || undefined,
        user_id: filters.user_id || undefined,
        payment_method: filters.payment_method || undefined,
      })
      
      if (res.data && res.data.list) {
        const exportRows = res.data.list.map(order => ({
          '订单号': order.order_no,
          '用户ID': order.user_id,
          '用户名': order.username || '-',
          '金额': formatCurrencyForExport(order.amount),
          '货币': order.currency,
          '支付方式': order.payment_method === 'stripe' ? 'Stripe' : '加密货币',
          '状态': formatStatusForExport(order.status),
          '创建时间': formatDateForExport(order.created_at),
          '支付时间': order.paid_at ? formatDateForExport(order.paid_at) : '-',
        }))
        
        exportData({
          filename: `orders_${new Date().toISOString().split('T')[0]}`,
          data: exportRows,
        })

        toast({ title: '导出成功', description: `已导出 ${exportRows.length} 条订单` })
      }
    } catch (error) {
      toast({
        title: '导出失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }

  const handleReset = () => {
    setFilters({
      status: '',
      user_id: '',
      payment_method: '',
    })
    setPage(1)
  }

  const handleViewDetail = (order: Order) => {
    setSelectedOrder(order)
    setShowDetailModal(true)
    setActionMenuId(null)
  }

  const handleStatusChange = async (orderId: string, newStatus: string) => {
    try {
      await orderApi.updateStatus(orderId, newStatus)
      toast({ title: '更新成功', description: '订单状态已更新' })
      loadOrders()
    } catch (error) {
      toast({
        title: '更新失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
    setActionMenuId(null)
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">订单管理</h1>
          <p className="text-muted-foreground">管理用户充值订单</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={handleExport}
            disabled={loading || orders.length === 0}
            className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <Download className="w-4 h-4" />
            导出
          </button>
          <button
            onClick={loadOrders}
            className="flex items-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
            刷新
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-card border border-border rounded-xl p-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div>
            <label className="block text-sm font-medium mb-2">订单状态</label>
            <select
              value={filters.status}
              onChange={(e) => setFilters({ ...filters, status: e.target.value })}
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            >
              <option value="">全部状态</option>
              <option value="pending">待支付</option>
              <option value="paid">已支付</option>
              <option value="failed">支付失败</option>
              <option value="refunded">已退款</option>
              <option value="cancelled">已取消</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">用户 ID</label>
            <input
              type="text"
              value={filters.user_id}
              onChange={(e) => setFilters({ ...filters, user_id: e.target.value })}
              placeholder="输入用户 ID"
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">支付方式</label>
            <select
              value={filters.payment_method}
              onChange={(e) => setFilters({ ...filters, payment_method: e.target.value })}
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            >
              <option value="">全部方式</option>
              <option value="stripe">Stripe</option>
              <option value="crypto">加密货币</option>
            </select>
          </div>
          <div className="flex items-end gap-2">
            <button
              onClick={handleReset}
              className="flex-1 px-4 py-2 border border-border rounded-lg hover:bg-accent"
            >
              重置
            </button>
            <button
              onClick={handleSearch}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg"
            >
              <Search className="w-4 h-4" />
              搜索
            </button>
          </div>
        </div>
      </div>

      {/* Orders Table */}
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-muted/50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  订单号
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  用户
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  金额
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  支付方式
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  创建时间
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
              ) : orders.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-12 text-center text-muted-foreground">
                    暂无订单数据
                  </td>
                </tr>
              ) : (
                orders.map((order) => (
                  <tr key={order.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center gap-2">
                        <CreditCard className="w-4 h-4 text-muted-foreground" />
                        <span className="font-mono text-sm">{order.order_no}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div>
                        <div className="font-medium">{order.username || '-'}</div>
                        <div className="text-xs text-muted-foreground">{order.user_id.slice(0, 8)}...</div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center gap-2">
                        <DollarSign className="w-4 h-4 text-green-500" />
                        <span className="font-medium text-green-500">
                          {formatCurrency(order.amount)}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 text-xs rounded-md ${
                        order.payment_method === 'stripe' 
                          ? 'bg-purple-500/10 text-purple-500' 
                          : 'bg-blue-500/10 text-blue-500'
                      }`}>
                        {order.payment_method === 'stripe' ? 'Stripe' : '加密货币'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center gap-2">
                        {order.status === 'paid' ? (
                          <CheckCircle className="w-4 h-4 text-green-500" />
                        ) : order.status === 'failed' || order.status === 'cancelled' ? (
                          <XCircle className="w-4 h-4 text-red-500" />
                        ) : (
                          <Clock className="w-4 h-4 text-yellow-500" />
                        )}
                        <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(order.status)}`}>
                          {getStatusText(order.status)}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground">
                      {formatDate(order.created_at)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right">
                      <div className="relative">
                        <button
                          onClick={() => setActionMenuId(actionMenuId === order.id ? null : order.id)}
                          className="p-2 hover:bg-accent rounded-lg transition-colors"
                        >
                          <MoreVertical className="w-4 h-4" />
                        </button>
                        {actionMenuId === order.id && (
                          <div className="absolute right-0 top-full mt-1 w-40 bg-card border border-border rounded-lg shadow-lg py-1 z-10">
                            <button
                              onClick={() => handleViewDetail(order)}
                              className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                            >
                              <Eye className="w-4 h-4" />
                              查看详情
                            </button>
                            {order.status === 'pending' && (
                              <>
                                <button
                                  onClick={() => handleStatusChange(order.id, 'paid')}
                                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-green-500 hover:bg-accent"
                                >
                                  <CheckCircle className="w-4 h-4" />
                                  标记已支付
                                </button>
                                <button
                                  onClick={() => handleStatusChange(order.id, 'cancelled')}
                                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-500 hover:bg-accent"
                                >
                                  <XCircle className="w-4 h-4" />
                                  取消订单
                                </button>
                              </>
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

      {/* Order Detail Modal */}
      {showDetailModal && selectedOrder && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-xl p-6 w-full max-w-lg">
            <h3 className="text-lg font-semibold mb-6">订单详情</h3>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">订单号</label>
                  <div className="px-4 py-2 bg-muted rounded-lg font-mono text-sm">
                    {selectedOrder.order_no}
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">状态</label>
                  <div className={`px-4 py-2 rounded-lg ${getStatusColor(selectedOrder.status)}`}>
                    {getStatusText(selectedOrder.status)}
                  </div>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">用户 ID</label>
                  <div className="px-4 py-2 bg-muted rounded-lg font-mono text-sm">
                    {selectedOrder.user_id.slice(0, 8)}...
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">用户名</label>
                  <div className="px-4 py-2 bg-muted rounded-lg">
                    {selectedOrder.username || '-'}
                  </div>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">金额</label>
                  <div className="px-4 py-2 bg-muted rounded-lg text-green-500 font-medium">
                    {formatCurrency(selectedOrder.amount)}
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">支付方式</label>
                  <div className="px-4 py-2 bg-muted rounded-lg">
                    {selectedOrder.payment_method === 'stripe' ? 'Stripe' : '加密货币'}
                  </div>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">创建时间</label>
                  <div className="px-4 py-2 bg-muted rounded-lg text-sm">
                    {formatDate(selectedOrder.created_at)}
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">支付时间</label>
                  <div className="px-4 py-2 bg-muted rounded-lg text-sm">
                    {selectedOrder.paid_at ? formatDate(selectedOrder.paid_at) : '-'}
                  </div>
                </div>
              </div>
            </div>
            <div className="flex justify-end mt-6">
              <button
                onClick={() => {
                  setShowDetailModal(false)
                  setSelectedOrder(null)
                }}
                className="px-4 py-2 border border-border rounded-lg hover:bg-accent"
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
