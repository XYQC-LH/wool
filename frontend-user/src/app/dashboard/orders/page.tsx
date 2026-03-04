'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useToast } from '@/components/ui/use-toast'
import { useAuthStore } from '@/store/auth'
import { useOrders, useBillingOverview, useConsumptionDetails, useCreateOrder, useCancelOrder } from '@/lib/query'
import { formatCurrency, formatDate } from '@/lib/utils'
import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  CreditCard,
  Plus,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  Wallet,
  DollarSign,
  Activity,
  Loader2,
} from 'lucide-react'

// 订单状态配置
const statusConfig: Record<string, { label: string; color: string; icon: React.ComponentType<{ className?: string }> }> = {
  pending: { label: '待支付', color: 'text-yellow-500', icon: Clock },
  paid: { label: '已支付', color: 'text-green-500', icon: CheckCircle },
  failed: { label: '支付失败', color: 'text-red-500', icon: XCircle },
  cancelled: { label: '已取消', color: 'text-gray-500', icon: XCircle },
  refunded: { label: '已退款', color: 'text-blue-500', icon: AlertCircle },
}

export default function OrdersPage() {
  const router = useRouter()
  const { isAuthenticated } = useAuthStore()
  const { toast } = useToast()
  
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState('')
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [amount, setAmount] = useState('')
  const [paymentMethod, setPaymentMethod] = useState<'stripe' | 'crypto'>('stripe')
  const [cancelTarget, setCancelTarget] = useState<{ id: string; order_no: string } | null>(null)

  // 使用 TanStack Query
  const { data: ordersData, isLoading: ordersLoading, refetch: refetchOrders } = useOrders({
    page,
    page_size: 10,
    status: statusFilter || undefined,
  })
  
  const { data: billingOverview, isLoading: billingLoading } = useBillingOverview()
  const { data: consumptionData, isLoading: consumptionLoading } = useConsumptionDetails({ page: 1, page_size: 10 })
  
  const createOrder = useCreateOrder()
  const cancelOrder = useCancelOrder()

  const orders = ordersData?.list || []
  const totalPages = Math.max(1, Math.ceil((ordersData?.total || 0) / 10))
  const consumptionDetails = consumptionData?.list || []

  const defaultBilling = {
    balance: 0,
    today_cost: 0,
    month_cost: 0,
    total_recharge: 0,
    today_requests: 0,
    month_requests: 0,
  }

  const billing = billingOverview || defaultBilling

  const handleCreateOrder = async () => {
    const parsedAmount = Number(amount)
    if (!Number.isFinite(parsedAmount) || parsedAmount <= 0) {
      toast({
        title: '金额不合法',
        description: '请输入有效的充值金额',
        variant: 'destructive',
      })
      return
    }

    try {
      const result = await createOrder.mutateAsync({
        amount: parsedAmount,
        currency: 'CNY',
        payment_method: paymentMethod,
      })
      
      toast({
        title: '订单已创建',
        description: '即将跳转到支付页面…',
      })
      setShowCreateModal(false)
      setAmount('')
      
      if (result.payment_url) {
        router.push(result.payment_url)
      }
    } catch (error) {
      toast({
        title: '创建订单失败',
        description: error instanceof Error ? error.message : '请稍后重试',
        variant: 'destructive',
      })
    }
  }

  const handleCancelOrder = async () => {
    if (!cancelTarget) return
    
    try {
      await cancelOrder.mutateAsync(cancelTarget.id)
      toast({
        title: '订单已取消',
        description: `订单号：${cancelTarget.order_no}`,
      })
      setCancelTarget(null)
    } catch (error) {
      toast({
        title: '取消订单失败',
        description: error instanceof Error ? error.message : '请稍后重试',
        variant: 'destructive',
      })
    }
  }

  const getStatusInfo = (status: string) => {
    return statusConfig[status] || { label: status, color: 'text-gray-500', icon: AlertCircle }
  }

  // 未登录状态
  if (!isAuthenticated) {
    return (
      <div className="space-y-6 animate-fade-in">
        <div>
          <h1 className="text-2xl font-bold">充值账单</h1>
          <p className="text-muted-foreground">管理您的充值订单和账单记录</p>
        </div>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center py-12">
              <CreditCard className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">需要登录</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                请先登录以查看您的充值账单
              </p>
              <div className="flex justify-center gap-2 mt-4">
                <Link href="/login">
                  <Button>登录</Button>
                </Link>
                <Link href="/register">
                  <Button variant="outline">注册</Button>
                </Link>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">充值账单</h1>
          <p className="text-muted-foreground">管理您的充值订单和账单记录</p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="mr-2 h-4 w-4" />
          充值
        </Button>
      </div>

      {/* 账单概览卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="bg-gradient-to-br from-primary/10 to-primary/5 border-primary/20">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">当前余额</CardTitle>
            <Wallet className="h-4 w-4 text-primary" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {billingLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                formatCurrency(billing.balance)
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-1">可用于模型调用和生成</p>
          </CardContent>
        </Card>
        
        <Card className="bg-gradient-to-br from-amber-500/10 to-amber-500/5 border-amber-500/20">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">今日消费</CardTitle>
            <DollarSign className="h-4 w-4 text-amber-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {billingLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                formatCurrency(billing.today_cost)
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-1">{billing.today_requests.toLocaleString()} 次请求</p>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-blue-500/10 to-blue-500/5 border-blue-500/20">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">本月消费</CardTitle>
            <Activity className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {billingLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                formatCurrency(billing.month_cost)
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-1">{billing.month_requests.toLocaleString()} 次请求</p>
          </CardContent>
        </Card>
      </div>

      {/* 筛选器 */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-4">
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value)
                setPage(1)
              }}
              className="px-3 py-2 rounded-md border border-input bg-background text-sm"
            >
              <option value="">全部状态</option>
              <option value="pending">待支付</option>
              <option value="paid">已支付</option>
              <option value="failed">支付失败</option>
              <option value="cancelled">已取消</option>
              <option value="refunded">已退款</option>
            </select>
            <Button variant="outline" size="icon" onClick={() => refetchOrders()}>
              <RefreshCw className="h-4 w-4" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* 订单列表 */}
      <Card>
        <CardHeader>
          <CardTitle>订单记录</CardTitle>
          <CardDescription>您的所有充值订单</CardDescription>
        </CardHeader>
        <CardContent>
          {ordersLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : orders.length === 0 ? (
            <div className="text-center py-12">
              <CreditCard className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">暂无订单</h3>
              <p className="mt-2 text-muted-foreground">点击上方“充值”按钮创建您的第一笔订单</p>
            </div>
          ) : (
            <div className="space-y-4">
              {orders.map((order) => {
                const statusInfo = getStatusInfo(order.status)
                const StatusIcon = statusInfo.icon
                return (
                  <div
                    key={order.id}
                    className="flex items-center justify-between p-4 border rounded-lg hover:bg-accent/50 transition-colors"
                  >
                    <div className="flex items-center gap-4">
                      <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                        <CreditCard className="h-5 w-5 text-primary" />
                      </div>
                      <div>
                        <div className="font-medium">订单号: {order.order_no}</div>
                        <div className="text-sm text-muted-foreground">
                          {formatDate(order.created_at)}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-6">
                      <div className="text-right">
                        <div className="font-bold text-lg">
                          {formatCurrency(order.amount, order.currency || 'CNY')}
                        </div>
                        <div className={`flex items-center gap-1 text-sm ${statusInfo.color}`}>
                          <StatusIcon className="h-3 w-3" />
                          {statusInfo.label}
                        </div>
                      </div>
                      {order.status === 'pending' && (
                        <div className="flex gap-2">
                          <Button size="sm" variant="outline" onClick={() => setCancelTarget(order)}>
                            取消
                          </Button>
                          <Button
                            size="sm"
                            onClick={() => {
                              const orderNo = encodeURIComponent(order.order_no)
                              const url =
                                order.payment_method === 'stripe'
                                  ? `/pay/stripe?order_no=${orderNo}`
                                  : order.payment_method === 'crypto'
                                    ? `/pay/crypto?order_no=${orderNo}`
                                    : `/pay?order_no=${orderNo}`
                              router.push(url)
                            }}
                          >
                            支付
                          </Button>
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}

          {/* 分页 */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-6">
              <Button
                variant="outline"
                size="icon"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <span className="text-sm text-muted-foreground">
                第 {page} 页，共 {totalPages} 页
              </span>
              <Button
                variant="outline"
                size="icon"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 消费明细 */}
      <Card>
        <CardHeader>
          <CardTitle>消费明细</CardTitle>
          <CardDescription>最近的模型调用和生成消费记录</CardDescription>
        </CardHeader>
        <CardContent>
          {consumptionLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : consumptionDetails.length === 0 ? (
            <div className="text-center py-12">
              <AlertCircle className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">暂无消费记录</h3>
              <p className="mt-2 text-muted-foreground">开始使用模型后，消费记录将显示在这里</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">时间</th>
                    <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">模型</th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">Token数</th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">费用</th>
                  </tr>
                </thead>
                <tbody>
                  {consumptionDetails.map((detail: { id: string; created_at: string; model: string; total_tokens: number; cost: number }) => (
                    <tr key={detail.id} className="border-b hover:bg-accent/50 transition-colors">
                      <td className="py-3 px-4 text-sm">{formatDate(detail.created_at)}</td>
                      <td className="py-3 px-4 text-sm font-medium">{detail.model}</td>
                      <td className="py-3 px-4 text-sm text-right font-mono">
                        {detail.total_tokens.toLocaleString()}
                      </td>
                      <td className="py-3 px-4 text-sm text-right font-medium">
                        {formatCurrency(detail.cost)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <ConfirmDialog
        open={!!cancelTarget}
        title="取消订单"
        description={cancelTarget ? `确定要取消订单号 ${cancelTarget.order_no} 吗？` : undefined}
        confirmText="确认取消"
        cancelText="返回"
        destructive
        loading={cancelOrder.isPending}
        onCancel={() => setCancelTarget(null)}
        onConfirm={handleCancelOrder}
      />

      {/* 创建订单弹窗 */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <Card className="w-full max-w-md mx-4">
            <CardHeader>
              <CardTitle>充值</CardTitle>
              <CardDescription>选择充值金额和支付方式</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">充值金额 (CNY)</label>
                <Input
                  type="number"
                  placeholder="请输入充值金额"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  min="1"
                  step="0.01"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">快捷金额</label>
                <div className="grid grid-cols-4 gap-2">
                  {[10, 50, 100, 500].map((value) => (
                    <Button
                      key={value}
                      variant={amount === String(value) ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setAmount(String(value))}
                    >
                      ${value}
                    </Button>
                  ))}
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">支付方式</label>
                <div className="grid grid-cols-2 gap-2">
                  <Button
                    variant={paymentMethod === 'stripe' ? 'default' : 'outline'}
                    onClick={() => setPaymentMethod('stripe')}
                    className="justify-start"
                  >
                    <CreditCard className="mr-2 h-4 w-4" />
                    信用卡
                  </Button>
                  <Button
                    variant={paymentMethod === 'crypto' ? 'default' : 'outline'}
                    onClick={() => setPaymentMethod('crypto')}
                    className="justify-start"
                  >
                    <svg className="mr-2 h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z" />
                    </svg>
                    加密货币
                  </Button>
                </div>
              </div>

              <div className="flex gap-2 pt-4">
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => {
                    setShowCreateModal(false)
                    setAmount('')
                  }}
                >
                  取消
                </Button>
                <Button
                  className="flex-1"
                  onClick={handleCreateOrder}
                  disabled={createOrder.isPending || !amount}
                >
                  {createOrder.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      处理中...
                    </>
                  ) : '确认充值'}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}