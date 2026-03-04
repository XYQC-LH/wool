'use client'

import { useCallback, useEffect, useState } from 'react'
import { RefreshCw, Filter, AlertTriangle } from 'lucide-react'
import api from '@/lib/api'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Loading } from '@/components/common/loading'
import { EmptyState } from '@/components/common/empty-state'
import { Pagination } from '@/components/common/pagination'

interface CircuitEvent {
  id: number
  provider_id: number
  provider_name: string
  channel_name: string
  event_type: 'open' | 'close' | 'half_open' | string
  reason: string
  created_at: string
}

function EventTypeBadge({ type }: { type: CircuitEvent['event_type'] }) {
  const color =
    type === 'open'
      ? 'bg-red-500/10 text-red-500'
      : type === 'half_open'
        ? 'bg-yellow-500/10 text-yellow-500'
        : type === 'close'
          ? 'bg-green-500/10 text-green-500'
          : 'bg-muted text-muted-foreground'

  const text = type === 'open' ? '熔断' : type === 'half_open' ? '半开' : type === 'close' ? '恢复' : type

  return <span className={`px-2 py-1 rounded-full text-xs font-medium ${color}`}>{text}</span>
}

export default function DispatchEventsPage() {
  const [loading, setLoading] = useState(true)
  const [events, setEvents] = useState<CircuitEvent[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [providerId, setProviderId] = useState('')
  const [inputProviderId, setInputProviderId] = useState('')

  const loadEvents = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = {
        page: String(page),
        page_size: String(pageSize),
      }
      if (providerId.trim()) {
        params.provider_id = providerId.trim()
      }

      const response = await api.get('/api/admin/dispatch/events', { params })

      if (!response || typeof response !== 'object') {
        setEvents([])
        setTotal(0)
        return
      }

      if ('success' in response && response.success === false) {
        setEvents([])
        setTotal(0)
        return
      }

      const data =
        'data' in response && response.data && typeof response.data === 'object'
          ? (response.data as Record<string, unknown>)
          : null
      const list: CircuitEvent[] = Array.isArray(data?.list) ? (data.list as CircuitEvent[]) : []
      setEvents(list)
      setTotal(typeof data?.total === 'number' ? data.total : 0)
    } catch {
      setEvents([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, providerId])

  useEffect(() => {
    loadEvents()
  }, [loadEvents])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">熔断事件</h1>
          <p className="text-muted-foreground">查看源头熔断/恢复事件记录</p>
        </div>
        <Button onClick={loadEvents} variant="outline" className="gap-2">
          <RefreshCw className="w-4 h-4" />
          刷新
        </Button>
      </div>

      <Card className="p-6">
        <div className="flex flex-col md:flex-row gap-4 md:items-end">
          <div className="flex-1">
            <label className="block text-sm font-medium mb-2">源头 ID（可选）</label>
            <Input
              value={inputProviderId}
              onChange={(e) => setInputProviderId(e.target.value)}
              placeholder="例如：123"
            />
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              className="gap-2"
              onClick={() => {
                setPage(1)
                setProviderId(inputProviderId)
              }}
            >
              <Filter className="w-4 h-4" />
              筛选
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                setInputProviderId('')
                setProviderId('')
                setPage(1)
              }}
            >
              重置
            </Button>
          </div>
        </div>
      </Card>

      <Card className="p-6">
        {loading ? (
          <Loading />
        ) : events.length === 0 ? (
          <EmptyState title="暂无事件" description="当前筛选条件下没有熔断事件记录" />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">时间</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">模型/源头</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">渠道</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">类型</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">原因</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {events.map((e) => (
                  <tr key={e.id} className="hover:bg-muted/30">
                    <td className="px-4 py-3 text-sm text-muted-foreground">
                      {new Date(e.created_at).toLocaleString('zh-CN')}
                    </td>
                    <td className="px-4 py-3 text-sm font-medium">{e.provider_name || `#${e.provider_id}`}</td>
                    <td className="px-4 py-3 text-sm text-muted-foreground">{e.channel_name || '-'}</td>
                    <td className="px-4 py-3">
                      <EventTypeBadge type={e.event_type} />
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {e.reason ? (
                        <span className="text-muted-foreground">{e.reason}</span>
                      ) : (
                        <span className="inline-flex items-center gap-1 text-muted-foreground">
                          <AlertTriangle className="w-4 h-4" />
                          -
                        </span>
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
