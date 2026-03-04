'use client'

import { useCallback, useEffect, useState } from 'react'
import { logApi, Log, getErrorMessage } from '@/lib/api'
import { useToast } from '@/components/ui/use-toast'
import {
  Search,
  RefreshCw,
  Download,
  Clock,
  CheckCircle,
  XCircle,
} from 'lucide-react'
import { formatDate, formatCurrency, getStatusColor, getStatusText } from '@/lib/utils'
import { exportData, formatDateForExport, formatCurrencyForExport, formatStatusForExport } from '@/lib/export'

export default function LogsPage() {
  const { toast } = useToast()
  const [logs, setLogs] = useState<Log[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [filters, setFilters] = useState({
    user_id: '',
    model: '',
    channel_id: '',
    status: '',
    start_date: '',
    end_date: '',
  })

  const loadLogs = useCallback(async (targetPage: number) => {
    setLoading(true)
    try {
      const res = await logApi.list({
        page: targetPage,
        page_size: pageSize,
        user_id: filters.user_id || undefined,
        model: filters.model || undefined,
        channel_id: filters.channel_id ? parseInt(filters.channel_id) : undefined,
        status: filters.status || undefined,
        start_date: filters.start_date || undefined,
        end_date: filters.end_date || undefined,
      })
      if (res.data) {
        setLogs(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (error) {
      toast({
        title: '加载日志失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [filters.channel_id, filters.end_date, filters.model, filters.start_date, filters.status, filters.user_id, pageSize, toast])

  useEffect(() => {
    loadLogs(page)
  }, [loadLogs, page])

  const handleSearch = () => {
    if (page === 1) {
      loadLogs(1)
      return
    }
    setPage(1)
  }

  const handleExport = async () => {
    try {
      // 获取所有数据（不分页）
      const res = await logApi.list({
        page: 1,
        page_size: 10000, // 获取大量数据
        user_id: filters.user_id || undefined,
        model: filters.model || undefined,
        channel_id: filters.channel_id ? parseInt(filters.channel_id) : undefined,
        status: filters.status || undefined,
        start_date: filters.start_date || undefined,
        end_date: filters.end_date || undefined,
      })
      
      if (res.data && res.data.list) {
        const exportRows = res.data.list.map(log => ({
          '时间': formatDateForExport(log.created_at),
          '用户ID': log.user_id,
          '用户名': log.username || '-',
          '模型': log.model,
          '渠道ID': log.channel_id,
          '渠道名称': log.channel_name || '-',
          '输入Token': log.prompt_tokens,
          '输出Token': log.completion_tokens,
          '总Token': log.total_tokens,
          '费用': formatCurrencyForExport(log.total_cost),
          '上游成本': formatCurrencyForExport(log.upstream_cost),
          '利润': formatCurrencyForExport(log.profit),
          '耗时(ms)': log.duration,
          '状态': formatStatusForExport(log.status),
          '错误信息': log.error_message || '-',
        }))
        
        exportData({
          filename: `logs_${new Date().toISOString().split('T')[0]}`,
          data: exportRows,
        })

        toast({ title: '导出成功', description: `已导出 ${exportRows.length} 条日志` })
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
      user_id: '',
      model: '',
      channel_id: '',
      status: '',
      start_date: '',
      end_date: '',
    })
    setPage(1)
    loadLogs(1)
  }

  const handleQuickDate = (days: number) => {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - days)
    
    setFilters({
      ...filters,
      start_date: start.toISOString().split('T')[0],
      end_date: end.toISOString().split('T')[0],
    })
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">日志查询</h1>
          <p className="text-muted-foreground">查看 API 请求日志和统计</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={handleExport}
            disabled={loading || logs.length === 0}
            className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <Download className="w-4 h-4" />
            导出
          </button>
          <button
            onClick={() => loadLogs(page)}
            className="flex items-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
            刷新
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-card border border-border rounded-xl p-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
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
            <label className="block text-sm font-medium mb-2">模型</label>
            <input
              type="text"
              value={filters.model}
              onChange={(e) => setFilters({ ...filters, model: e.target.value })}
              placeholder="例如：gpt-4"
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">渠道 ID</label>
            <input
              type="text"
              value={filters.channel_id}
              onChange={(e) => setFilters({ ...filters, channel_id: e.target.value })}
              placeholder="输入渠道 ID"
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">状态</label>
            <select
              value={filters.status}
              onChange={(e) => setFilters({ ...filters, status: e.target.value })}
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            >
              <option value="">全部状态</option>
              <option value="success">成功</option>
              <option value="error">失败</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">开始日期</label>
            <input
              type="date"
              value={filters.start_date}
              onChange={(e) => setFilters({ ...filters, start_date: e.target.value })}
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">结束日期</label>
            <input
              type="date"
              value={filters.end_date}
              onChange={(e) => setFilters({ ...filters, end_date: e.target.value })}
              className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
          </div>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3 mt-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">快速选择：</span>
            <button
              onClick={() => handleQuickDate(1)}
              className="px-3 py-1 text-sm border border-border rounded-lg hover:bg-accent"
            >
              今天
            </button>
            <button
              onClick={() => handleQuickDate(7)}
              className="px-3 py-1 text-sm border border-border rounded-lg hover:bg-accent"
            >
              最近7天
            </button>
            <button
              onClick={() => handleQuickDate(30)}
              className="px-3 py-1 text-sm border border-border rounded-lg hover:bg-accent"
            >
              最近30天
            </button>
          </div>
          <div className="flex gap-3">
            <button
              onClick={handleReset}
              className="px-4 py-2 border border-border rounded-lg hover:bg-accent"
            >
              重置
            </button>
            <button
              onClick={handleSearch}
              className="flex items-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg"
            >
              <Search className="w-4 h-4" />
              搜索
            </button>
          </div>
        </div>
      </div>

      {/* Logs Table */}
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-muted/50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  时间
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  用户
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  模型
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  渠道
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Token
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  费用
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  耗时
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  状态
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading ? (
                <tr>
                  <td colSpan={8} className="px-4 py-12 text-center">
                    <div className="flex items-center justify-center">
                      <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-orange-500"></div>
                    </div>
                  </td>
                </tr>
              ) : logs.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-4 py-12 text-center text-muted-foreground">
                    暂无日志数据
                  </td>
                </tr>
              ) : (
                logs.map((log) => (
                  <tr key={log.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3 whitespace-nowrap text-sm">
                      <div className="flex items-center gap-2">
                        <Clock className="w-4 h-4 text-muted-foreground" />
                        {formatDate(log.created_at)}
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm">
                      <div>
                        <div className="font-medium">{log.username || '-'}</div>
                        <div className="text-xs text-muted-foreground">{log.user_id.slice(0, 8)}...</div>
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span className="px-2 py-1 text-xs bg-blue-500/10 text-blue-500 rounded-md">
                        {log.model}
                      </span>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm">
                      <div>
                        <div className="font-medium">{log.channel_name || '-'}</div>
                        <div className="text-xs text-muted-foreground">ID: {log.channel_id}</div>
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm">
                      <div className="space-y-1">
                        <div className="text-xs">
                          <span className="text-muted-foreground">输入:</span> {log.prompt_tokens.toLocaleString()}
                        </div>
                        <div className="text-xs">
                          <span className="text-muted-foreground">输出:</span> {log.completion_tokens.toLocaleString()}
                        </div>
                        <div className="text-xs font-medium">
                          <span className="text-muted-foreground">总计:</span> {log.total_tokens.toLocaleString()}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm">
                      <div className="space-y-1">
                        <div className="text-green-500 font-medium">
                          {formatCurrency(log.total_cost)}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          成本: {formatCurrency(log.upstream_cost)}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm">
                      <span className={`${log.duration > 5000 ? 'text-red-500' : log.duration > 2000 ? 'text-yellow-500' : 'text-green-500'}`}>
                        {log.duration}ms
                      </span>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <div className="flex items-center gap-2">
                        {log.status === 'success' ? (
                          <CheckCircle className="w-4 h-4 text-green-500" />
                        ) : (
                          <XCircle className="w-4 h-4 text-red-500" />
                        )}
                        <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(log.status)}`}>
                          {getStatusText(log.status)}
                        </span>
                      </div>
                      {log.error_message && (
                        <div className="text-xs text-red-500 mt-1 max-w-xs truncate" title={log.error_message}>
                          {log.error_message}
                        </div>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-4 border-t border-border">
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
    </div>
  )
}
