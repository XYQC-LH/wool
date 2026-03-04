'use client'

import { useCallback, useEffect, useState } from 'react'
import { userApi, User, UserStats, getErrorMessage } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useToast } from '@/components/ui/use-toast'
import { formatDate, formatCurrency, formatNumber } from '@/lib/utils'
import {
  ArrowLeft,
  User as UserIcon,
  Mail,
  DollarSign,
  Calendar,
  Activity,
} from 'lucide-react'
import Link from 'next/link'
import { useParams } from 'next/navigation'

export default function UserDetailPage() {
  const { toast } = useToast()
  const params = useParams()
  const id = Array.isArray(params.id) ? params.id[0] : params.id
  const [user, setUser] = useState<User | null>(null)
  const [stats, setStats] = useState<UserStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [statsLoading, setStatsLoading] = useState(false)
  const [showBalanceModal, setShowBalanceModal] = useState(false)
  const [balanceAmount, setBalanceAmount] = useState('')
  const [updating, setUpdating] = useState(false)

  const loadUser = useCallback(async (userID: string) => {
    setLoading(true)
    try {
      const res = await userApi.get(userID)
      if (res.data) {
        setUser(res.data)
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
  }, [toast])

  const loadUserStats = useCallback(async (userID: string) => {
    setStatsLoading(true)
    try {
      const res = await userApi.stats(userID)
      if (res.data) {
        setStats(res.data)
      } else {
        setStats(null)
      }
    } catch (error) {
      toast({
        title: '加载统计失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
      setStats(null)
    } finally {
      setStatsLoading(false)
    }
  }, [toast])

  useEffect(() => {
    if (!id) return
    loadUser(id)
    loadUserStats(id)
  }, [id, loadUser, loadUserStats])

  const handleAdjustBalance = async () => {
    if (!id) return
    const amount = Number(balanceAmount)
    if (!Number.isFinite(amount) || amount === 0) {
      toast({ title: '金额不合法', description: '请输入有效的调整金额（可为正或负）', variant: 'destructive' })
      return
    }
    setUpdating(true)
    try {
      await userApi.updateBalance(id, amount, '管理员调整')
      setShowBalanceModal(false)
      setBalanceAmount('')
      toast({ title: '调整成功', description: '余额已更新' })
      loadUser(id)
    } catch (error) {
      toast({
        title: '调整失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setUpdating(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-orange-500"></div>
      </div>
    )
  }

  if (!user) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">用户不存在</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center gap-4">
        <Link href="/dashboard/users">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="w-4 h-4" />
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold">用户详情</h1>
          <p className="text-muted-foreground">查看和管理用户信息</p>
        </div>
      </div>

      {/* User Info */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <UserIcon className="w-5 h-5" />
              基本信息
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-16 h-16 bg-orange-500 rounded-full flex items-center justify-center text-white text-2xl font-bold">
                {user.username.charAt(0).toUpperCase()}
              </div>
              <div>
                <h3 className="text-lg font-semibold">{user.username}</h3>
                <p className="text-sm text-muted-foreground">{user.email}</p>
              </div>
            </div>
            <div className="space-y-2 pt-4 border-t">
              <div className="flex items-center gap-2 text-sm">
                <Mail className="w-4 h-4 text-muted-foreground" />
                <span className="text-muted-foreground">用户ID:</span>
                <span className="font-mono">{user.id}</span>
              </div>
              <div className="flex items-center gap-2 text-sm">
                <Calendar className="w-4 h-4 text-muted-foreground" />
                <span className="text-muted-foreground">注册时间:</span>
                <span>{formatDate(user.created_at)}</span>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <DollarSign className="w-5 h-5" />
              账户信息
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm text-muted-foreground">当前余额</p>
              <p className="text-3xl font-bold text-green-500">{formatCurrency(user.balance)}</p>
            </div>
            <div className="pt-4 border-t">
              <Button onClick={() => setShowBalanceModal(true)} className="w-full">
                调整余额
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Usage Stats */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="w-5 h-5" />
            使用统计
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div className="p-4 bg-muted rounded-lg">
              <p className="text-sm text-muted-foreground">总请求数</p>
              <p className="text-2xl font-bold">
                {statsLoading ? '-' : stats ? Number(stats.total_requests || 0).toLocaleString() : '-'}
              </p>
            </div>
            <div className="p-4 bg-muted rounded-lg">
              <p className="text-sm text-muted-foreground">Token消耗</p>
              <p className="text-2xl font-bold">
                {statsLoading ? '-' : stats ? formatNumber(Number(stats.total_tokens || 0)) : '-'}
              </p>
            </div>
            <div className="p-4 bg-muted rounded-lg">
              <p className="text-sm text-muted-foreground">总消费</p>
              <p className="text-2xl font-bold">
                {statsLoading ? '-' : stats ? formatCurrency(Number(stats.total_cost || 0)) : '-'}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Balance Modal */}
      {showBalanceModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
          <Card className="w-full max-w-md">
            <CardHeader>
              <CardTitle>调整余额</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">调整金额（正数增加，负数减少）</label>
                <input
                  type="number"
                  placeholder="例如：100 或 -50"
                  value={balanceAmount}
                  onChange={(e) => setBalanceAmount(e.target.value)}
                  className="w-full px-3 py-2 border rounded-md"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowBalanceModal(false)
                    setBalanceAmount('')
                  }}
                  className="flex-1"
                >
                  取消
                </Button>
                <Button
                  onClick={handleAdjustBalance}
                  disabled={updating || !balanceAmount}
                  className="flex-1"
                >
                  {updating ? '处理中...' : '确认'}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}
