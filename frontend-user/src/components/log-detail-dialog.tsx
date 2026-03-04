'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { X, Copy, ChevronDown, ChevronUp, CheckCircle, XCircle, Clock } from 'lucide-react'
import { copyToClipboard, formatCurrency, formatDate, formatNumber } from '@/lib/utils'
import { useToast } from '@/components/ui/use-toast'
import type { Log } from '@/lib/api'

interface LogDetailDialogProps {
  log: Log | null
  open: boolean
  onClose: () => void
}

export function LogDetailDialog({ log, open, onClose }: LogDetailDialogProps) {
  const { toast } = useToast()
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set())

  const toggleSection = (section: string) => {
    const newExpanded = new Set(expandedSections)
    if (newExpanded.has(section)) {
      newExpanded.delete(section)
    } else {
      newExpanded.add(section)
    }
    setExpandedSections(newExpanded)
  }

  const handleCopy = async (text: string) => {
    await copyToClipboard(text)
    toast({
      title: '已复制',
      description: '内容已复制到剪贴板',
    })
  }

  if (!log || !open) return null

  const isSuccess = log.status === 'success'
  const isPending = log.status === 'pending'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
      <Card className="w-full max-w-4xl max-h-[90vh] overflow-hidden">
        <CardHeader className="flex items-center justify-between border-b">
          <CardTitle className="flex items-center gap-2">
            {isSuccess ? (
              <CheckCircle className="w-5 h-5 text-emerald-500" />
            ) : (
              <XCircle className="w-5 h-5 text-red-500" />
            )}
            请求详情
          </CardTitle>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </CardHeader>
        <CardContent className="p-0 overflow-y-auto max-h-[calc(90vh-80px)]">
          <div className="p-6 space-y-6">
            {/* 基本信息 */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="p-3 bg-muted rounded-lg">
                <p className="text-xs text-muted-foreground">请求时间</p>
                <p className="text-sm font-medium mt-1">
                  {log.created_at ? formatDate(log.created_at) : '-'}
                </p>
              </div>
              <div className="p-3 bg-muted rounded-lg">
                <p className="text-xs text-muted-foreground">模型</p>
                <p className="text-sm font-medium mt-1">{log.model}</p>
              </div>
              <div className="p-3 bg-muted rounded-lg">
                <p className="text-xs text-muted-foreground">响应时间</p>
                <p className="text-sm font-medium mt-1">
                  {Number.isFinite(log.duration_ms) ? `${log.duration_ms}ms` : `${log.duration}ms`}
                </p>
              </div>
              <div className="p-3 bg-muted rounded-lg">
                <p className="text-xs text-muted-foreground">费用</p>
                <p className="text-sm font-medium mt-1">{formatCurrency(Number(log.total_cost))}</p>
              </div>
            </div>

            {/* 统计信息 */}
            <div>
              <button onClick={() => toggleSection('stats')} className="flex items-center justify-between w-full text-left">
                <h3 className="font-semibold">统计信息</h3>
                {expandedSections.has('stats') ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
              </button>
              {expandedSections.has('stats') && (
                <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="p-3 border rounded-lg">
                    <p className="text-xs text-muted-foreground">Token</p>
                    <p className="text-sm font-medium mt-1">{formatNumber(log.total_tokens)}</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      输入 {formatNumber(log.prompt_tokens)} / 输出 {formatNumber(log.completion_tokens)}
                    </p>
                  </div>
                  <div className="p-3 border rounded-lg">
                    <p className="text-xs text-muted-foreground">状态</p>
                    <p className="text-sm font-medium mt-1 flex items-center gap-2">
                      {isSuccess ? (
                        <CheckCircle className="w-4 h-4 text-emerald-500" />
                      ) : isPending ? (
                        <Clock className="w-4 h-4 text-amber-500" />
                      ) : (
                        <XCircle className="w-4 h-4 text-red-500" />
                      )}
                      {log.status}
                    </p>
                  </div>
                  <div className="p-3 border rounded-lg">
                    <p className="text-xs text-muted-foreground">状态码</p>
                    <p className="text-sm font-medium mt-1">{log.status_code ?? '-'}</p>
                  </div>
                  <div className="p-3 border rounded-lg">
                    <p className="text-xs text-muted-foreground">流式</p>
                    <p className="text-sm font-medium mt-1">{log.is_stream ? '是' : '否'}</p>
                  </div>
                  {log.token_key && (
                    <div className="p-3 border rounded-lg md:col-span-2">
                      <p className="text-xs text-muted-foreground">API Key（脱敏）</p>
                      <p className="text-sm font-mono mt-1">{log.token_key}</p>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* 错误信息 */}
            {log.error_message && (
              <div>
                <button
                  onClick={() => toggleSection('error')}
                  className="flex items-center justify-between w-full text-left"
                >
                  <h3 className="font-semibold text-red-500">错误信息</h3>
                  {expandedSections.has('error') ? (
                    <ChevronUp className="w-4 h-4" />
                  ) : (
                    <ChevronDown className="w-4 h-4" />
                  )}
                </button>
                {expandedSections.has('error') && (
                  <div className="mt-4">
                    <pre className="bg-red-500/10 border border-red-500/20 p-3 rounded-lg text-xs text-red-500 overflow-x-auto">
                      {log.error_message}
                    </pre>
                  </div>
                )}
              </div>
            )}

            {/* 原始数据 */}
            <div>
              <button onClick={() => toggleSection('raw')} className="flex items-center justify-between w-full text-left">
                <h3 className="font-semibold">原始数据</h3>
                {expandedSections.has('raw') ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
              </button>
              {expandedSections.has('raw') && (
                <div className="mt-4 space-y-2">
                  <div className="flex justify-end">
                    <Button variant="outline" size="sm" onClick={() => handleCopy(JSON.stringify(log, null, 2))}>
                      <Copy className="w-3 h-3 mr-1" />
                      复制 JSON
                    </Button>
                  </div>
                  <pre className="bg-muted p-3 rounded-lg text-xs overflow-x-auto">
                    {JSON.stringify(log, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
