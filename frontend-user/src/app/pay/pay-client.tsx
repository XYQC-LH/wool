'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import { useToast } from '@/components/ui/use-toast'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { useAuthStore } from '@/store/auth'
import { orderApi, Order } from '@/lib/api'
import { formatCurrency, formatDate } from '@/lib/utils'
import { handleApiError } from '@/lib/error-handler'
import { AlertCircle, CheckCircle2, Clock, Loader2, XCircle } from 'lucide-react'

const orderStatusConfig: Record<
  string,
  { label: string; icon: React.ComponentType<{ className?: string }>; color: string }
> = {
  pending: { label: '待支付', icon: Clock, color: 'text-yellow-500' },
  paid: { label: '已支付', icon: CheckCircle2, color: 'text-green-500' },
  failed: { label: '支付失败', icon: XCircle, color: 'text-red-500' },
  cancelled: { label: '已取消', icon: XCircle, color: 'text-gray-500' },
  refunded: { label: '已退款', icon: AlertCircle, color: 'text-blue-500' },
}

export default function PayClient(props: {
  title: string
  methodLabel: string
  icon: React.ReactNode
  orderNo: string
}) {
  const { title, methodLabel, icon, orderNo } = props
  const { toast } = useToast()
  const { isAuthenticated, isLoading: authLoading, fetchProfile } = useAuthStore()

  const [order, setOrder] = useState<Order | null>(null)
  const [loading, setLoading] = useState(false)
  const [paying, setPaying] = useState(false)
  const [cancelLoading, setCancelLoading] = useState(false)
  const [showPayConfirm, setShowPayConfirm] = useState(false)
  const [showCancelConfirm, setShowCancelConfirm] = useState(false)

  const statusInfo = useMemo(() => {
    if (!order?.status) return { label: '未知', icon: AlertCircle, color: 'text-muted-foreground' }
    return orderStatusConfig[order.status] || { label: order.status, icon: AlertCircle, color: 'text-muted-foreground' }
  }, [order?.status])
  const StatusIcon = statusInfo.icon

  const loadOrder = useCallback(async () => {
    setLoading(true)
    try {
      const res = await orderApi.getByOrderNo(orderNo)
      if (res.code === 0) {
        setOrder(res.data)
        return
      }
      toast({
        title: '加载订单失败',
        description: res.message || '请求失败',
        variant: 'destructive',
      })
      setOrder(null)
    } catch (error) {
      handleApiError(error, { customMessage: '加载订单失败，请稍后重试' })
      setOrder(null)
    } finally {
      setLoading(false)
    }
  }, [orderNo, toast])

  const handlePay = useCallback(async () => {
    setPaying(true)
    try {
      const res = await orderApi.payByOrderNo(orderNo)
      if (res.code === 0) {
        toast({
          title: '支付成功',
          description: '订单已完成支付，余额已更新',
        })
        await fetchProfile()
        await loadOrder()
        setShowPayConfirm(false)
        return
      }

      toast({
        title: '支付失败',
        description: res.message || '请求失败',
        variant: 'destructive',
      })
    } catch (error) {
      handleApiError(error, { customMessage: '支付失败，请稍后重试' })
    } finally {
      setPaying(false)
    }
  }, [fetchProfile, loadOrder, orderNo, toast])

  const handleCancelOrder = useCallback(async () => {
    if (!order) return
    setCancelLoading(true)
    try {
      const res = await orderApi.cancel(order.id)
      if (res.code === 0) {
        toast({ title: '订单已取消', description: `订单号：${order.order_no}` })
        await loadOrder()
        setShowCancelConfirm(false)
        return
      }
      toast({
        title: '取消失败',
        description: res.message || '请求失败',
        variant: 'destructive',
      })
    } catch (error) {
      handleApiError(error, { customMessage: '取消订单失败，请稍后重试' })
    } finally {
      setCancelLoading(false)
    }
  }, [order, loadOrder, toast])

  useEffect(() => {
    fetchProfile()
  }, [fetchProfile])

  useEffect(() => {
    if (!orderNo) return
    if (!isAuthenticated) return
    loadOrder()
  }, [isAuthenticated, loadOrder, orderNo])

  if (!orderNo) {
    return (
      <div className="min-h-screen bg-background p-4 md:p-8">
        <div className="max-w-xl mx-auto space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertCircle className="w-5 h-5 text-destructive" />
                参数缺失
              </CardTitle>
              <CardDescription>缺少订单号（order_no），无法继续支付。</CardDescription>
            </CardHeader>
            <CardContent className="flex justify-end">
              <Link href="/dashboard/orders">
                <Button>返回充值账单</Button>
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  if (authLoading && !isAuthenticated) {
    return (
      <div className="min-h-screen bg-background p-4 md:p-8">
        <div className="max-w-xl mx-auto space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
                正在加载用户信息…
              </CardTitle>
              <CardDescription>请稍候</CardDescription>
            </CardHeader>
            <CardContent className="flex justify-end">
              <Link href="/dashboard/orders">
                <Button variant="outline">返回充值账单</Button>
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen bg-background p-4 md:p-8">
        <div className="max-w-xl mx-auto space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertCircle className="w-5 h-5 text-destructive" />
                需要登录
              </CardTitle>
              <CardDescription>请先登录后再继续支付。</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col sm:flex-row gap-2 justify-end">
              <Link href="/login" className="sm:order-last">
                <Button className="w-full sm:w-auto">去登录</Button>
              </Link>
              <Link href="/dashboard/orders" className="sm:order-first">
                <Button variant="outline" className="w-full sm:w-auto">返回充值账单</Button>
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background p-4 md:p-8">
      <div className="max-w-xl mx-auto space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <span className="inline-flex items-center justify-center w-9 h-9 rounded-lg bg-primary/10 text-primary">
                {icon}
              </span>
              {title}
            </CardTitle>
            <CardDescription>订单号：{orderNo}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-lg border p-4 space-y-3">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">支付方式</span>
                <span className="font-medium">{methodLabel}</span>
              </div>

              {loading ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="w-4 h-4 animate-spin" />
                  正在加载订单信息…
                </div>
              ) : order ? (
                <>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">订单金额</span>
                    <span className="font-medium">{formatCurrency(Number(order.amount), order.currency || 'CNY')}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">创建时间</span>
                    <span className="font-medium">{formatDate(order.created_at)}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">订单状态</span>
                    <span className={`inline-flex items-center gap-1 font-medium ${statusInfo.color}`}>
                      <StatusIcon className="w-4 h-4" />
                      {statusInfo.label}
                    </span>
                  </div>
                </>
              ) : (
                <div className="text-sm text-muted-foreground">
                  未获取到订单信息，请稍后重试。
                </div>
              )}
            </div>

            <div className="flex flex-col sm:flex-row gap-2 justify-end">
              {order?.status === 'pending' ? (
                <>
                  <Button
                    variant="outline"
                    className="w-full sm:w-auto"
                    onClick={() => setShowCancelConfirm(true)}
                    disabled={loading || cancelLoading}
                  >
                    取消订单
                  </Button>
                  <Button
                    className="w-full sm:w-auto"
                    onClick={() => setShowPayConfirm(true)}
                    disabled={loading || paying}
                  >
                    立即支付
                  </Button>
                </>
              ) : null}

              <Link href="/dashboard/orders" className="sm:order-first">
                <Button variant={order?.status === 'pending' ? 'outline' : 'default'} className="w-full sm:w-auto">
                  返回充值账单
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>

      <ConfirmDialog
        open={showPayConfirm}
        title="确认支付"
        description="将为该订单执行支付确认（测试/手动模式）。确认后订单会标记为已支付并增加余额。"
        confirmText="确认支付"
        cancelText="返回"
        loading={paying}
        onCancel={() => setShowPayConfirm(false)}
        onConfirm={handlePay}
      />

      <ConfirmDialog
        open={showCancelConfirm}
        title="取消订单"
        description="确认要取消该订单吗？仅待支付订单可取消。"
        confirmText="确认取消"
        cancelText="返回"
        destructive
        loading={cancelLoading}
        onCancel={() => setShowCancelConfirm(false)}
        onConfirm={handleCancelOrder}
      />
    </div>
  )
}
