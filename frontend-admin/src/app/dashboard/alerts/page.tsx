'use client'

import { useCallback, useEffect, useState } from 'react'
import { alertApi, Alert, getErrorMessage } from '@/lib/api'
import { RefreshCw, ShieldAlert, CheckCircle2 } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Loading } from '@/components/common/loading'
import { EmptyState } from '@/components/common/empty-state'
import { Pagination } from '@/components/common/pagination'
import { useToast } from '@/components/ui/use-toast'
import { cn, formatRelativeTime } from '@/lib/utils'

function SeverityBadge({ severity }: { severity: Alert['severity'] }) {
  const color =
    severity === 'critical' || severity === 'error'
      ? 'bg-red-500/10 text-red-500'
      : severity === 'warning'
        ? 'bg-yellow-500/10 text-yellow-500'
        : 'bg-muted text-muted-foreground'

  return <span className={cn('px-2 py-1 rounded-full text-xs font-medium', color)}>{severity}</span>
}

function StatusBadge({ status }: { status: Alert['status'] }) {
  const color =
    status === 'active'
      ? 'bg-orange-500/10 text-orange-500'
      : status === 'resolved'
        ? 'bg-green-500/10 text-green-500'
        : 'bg-muted text-muted-foreground'

  const text = status === 'active' ? '未处理' : status === 'resolved' ? '已解决' : status
  return <span className={cn('px-2 py-1 rounded-full text-xs font-medium', color)}>{text}</span>
}

export default function AlertsPage() {
  const { toast } = useToast()
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [filters, setFilters] = useState({
    severity: '',
    status: 'active',
    type: '',
  })
  const [resolvingId, setResolvingId] = useState<string | null>(null)

  const loadAlerts = useCallback(async () => {
    setLoading(true)
    try {
      const res = await alertApi.list({
        page,
        page_size: pageSize,
        severity: filters.severity || undefined,
        status: filters.status || undefined,
        type: filters.type || undefined,
      })
      setAlerts(res.data?.list || [])
      setTotal(res.data?.total || 0)
    } catch (error) {
      toast({
        title: '加载告警失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
      setAlerts([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [filters.severity, filters.status, filters.type, page, pageSize, toast])

  useEffect(() => {
    loadAlerts()
  }, [loadAlerts])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const handleResolve = async (id: string) => {
    setResolvingId(id)
    try {
      await alertApi.resolve(id)
      toast({ title: '已处理', description: '告警已标记为已解决' })
      await loadAlerts()
    } catch (error) {
      toast({
        title: '处理失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setResolvingId(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">告警中心</h1>
          <p className="text-muted-foreground">查看与处理系统告警</p>
        </div>
        <Button onClick={loadAlerts} variant="outline" className="gap-2">
          <RefreshCw className="w-4 h-4" />
          刷新
        </Button>
      </div>

      <Card className="p-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 md:items-end">
          <div>
            <label className="block text-sm font-medium mb-2">严重级别</label>
            <select
              value={filters.severity}
              onChange={(e) => {
                setFilters((v) => ({ ...v, severity: e.target.value }))
                setPage(1)
              }}
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            >
              <option value="">全部</option>
              <option value="info">info</option>
              <option value="warning">warning</option>
              <option value="error">error</option>
              <option value="critical">critical</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">状态</label>
            <select
              value={filters.status}
              onChange={(e) => {
                setFilters((v) => ({ ...v, status: e.target.value }))
                setPage(1)
              }}
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            >
              <option value="">全部</option>
              <option value="active">未处理</option>
              <option value="resolved">已解决</option>
              <option value="ignored">已忽略</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">类型（可选）</label>
            <input
              type="text"
              value={filters.type}
              onChange={(e) => {
                setFilters((v) => ({ ...v, type: e.target.value }))
                setPage(1)
              }}
              placeholder="例如：channel_down"
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              className="flex-1"
              onClick={() => {
                setFilters({ severity: '', status: 'active', type: '' })
                setPage(1)
              }}
            >
              重置
            </Button>
            <Button className="flex-1" onClick={loadAlerts}>
              查询
            </Button>
          </div>
        </div>
      </Card>

      <Card className="p-6">
        {loading ? (
          <Loading />
        ) : alerts.length === 0 ? (
          <EmptyState icon={ShieldAlert} title="暂无告警" description="当前筛选条件下没有告警记录" />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">时间</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">级别</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">类型</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">标题/内容</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">状态</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {alerts.map((a) => (
                  <tr key={a.id} className="hover:bg-muted/30">
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      <div>{new Date(a.created_at).toLocaleString('zh-CN')}</div>
                      <div className="text-xs">{formatRelativeTime(a.created_at)}</div>
                    </td>
                    <td className="px-4 py-3">
                      <SeverityBadge severity={a.severity} />
                    </td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">{a.type}</td>
                    <td className="px-4 py-3">
                      <div className="text-sm font-medium">{a.title}</div>
                      <div className="text-xs text-muted-foreground line-clamp-2">{a.message}</div>
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={a.status} />
                    </td>
                    <td className="px-4 py-3 text-right">
                      {a.status === 'active' ? (
                        <Button
                          size="sm"
                          onClick={() => handleResolve(a.id)}
                          disabled={resolvingId === a.id}
                          className="gap-2"
                        >
                          <CheckCircle2 className="w-4 h-4" />
                          {resolvingId === a.id ? '处理中...' : '解决'}
                        </Button>
                      ) : (
                        <span className="text-xs text-muted-foreground">-</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <Pagination
          currentPage={page}
          totalPages={totalPages}
          total={total}
          pageSize={pageSize}
          onPageChange={(p) => setPage(p)}
          className="mt-6"
        />
      </Card>
    </div>
  )
}
