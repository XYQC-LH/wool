'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/store/auth'
import { useTokens, useCreateToken, useDeleteToken, useUpdateTokenStatus, useUpdateToken, useModels } from '@/lib/query'
import { setGatewayApiKey } from '@/lib/api'
import { formatCurrency, formatDate, formatNumber, copyToClipboard } from '@/lib/utils'
import { toast } from '@/components/ui/use-toast'
import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  Key,
  Plus,
  Copy,
  Trash2,
  Eye,
  EyeOff,
  MoreVertical,
  Loader2,
} from 'lucide-react'

function AllowedModelsPicker(props: {
  selected: string[]
  onChange: (next: string[]) => void
  availableModels: string[]
  loading: boolean
}) {
  const { selected, onChange, availableModels, loading } = props
  const [query, setQuery] = useState('')

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return availableModels
    return availableModels.filter((name) => name.toLowerCase().includes(q))
  }, [availableModels, query])

  const shown = filtered.slice(0, 20)

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <label className="text-sm font-medium">允许的模型（可选）</label>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-8"
          onClick={() => onChange([])}
        >
          不限制
        </Button>
      </div>

      <Input
        placeholder="搜索模型名称..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />

      {loading ? (
        <div className="p-3 text-sm text-muted-foreground">加载模型中...</div>
      ) : availableModels.length === 0 ? (
        <div className="p-3 text-sm text-muted-foreground">暂无可用模型</div>
      ) : (
        <div className="grid grid-cols-2 gap-2">
          {shown.map((modelName) => (
            <label
              key={modelName}
              className="flex items-center gap-2 p-2 border rounded-lg cursor-pointer hover:bg-accent"
            >
              <input
                type="checkbox"
                value={modelName}
                className="rounded"
                checked={selected.includes(modelName)}
                onChange={(e) => {
                  onChange(
                    e.target.checked
                      ? (selected.includes(modelName) ? selected : [...selected, modelName])
                      : selected.filter((m) => m !== modelName)
                  )
                }}
              />
              <span className="text-sm truncate" title={modelName}>
                {modelName}
              </span>
            </label>
          ))}
        </div>
      )}

      <p className="text-xs text-muted-foreground">
        {availableModels.length > 0 ? (
          <>
            已选择 <span className="text-foreground font-medium">{selected.length}</span> 个；展示前 20 个匹配项（共{' '}
            {filtered.length} 个）。留空表示允许所有模型。
          </>
        ) : (
          '留空表示允许所有模型'
        )}
      </p>
    </div>
  )
}

export default function TokensPage() {
  const { isAuthenticated } = useAuthStore()
  const { data: tokensData, isLoading: tokensLoading } = useTokens()
  const { data: modelsData } = useModels()
  
  // Mutations
  const createToken = useCreateToken()
  const deleteToken = useDeleteToken()
  const updateTokenStatus = useUpdateTokenStatus()
  const updateToken = useUpdateToken()

  // Local state
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [createdTokenKey, setCreatedTokenKey] = useState<string | null>(null)
  const [showAdvancedSettings, setShowAdvancedSettings] = useState(false)
  const [selectedToken, setSelectedToken] = useState<{ id: string; name: string } | null>(null)
  const [newTokenName, setNewTokenName] = useState('')
  const [tokenQuota, setTokenQuota] = useState('')
  const [tokenExpiry, setTokenExpiry] = useState('')
  const [tokenIpWhitelist, setTokenIpWhitelist] = useState('')
  const [tokenAllowedModels, setTokenAllowedModels] = useState<string[]>([])
  const [tokenRateLimit, setTokenRateLimit] = useState('')
  const [visibleKeys, setVisibleKeys] = useState<Set<string>>(new Set())
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null)

  const tokens = tokensData?.list || []
  const availableModels = useMemo(() => {
    if (!modelsData) return []
    return Array.from(new Set(modelsData.map((m) => m.name).filter(Boolean)))
  }, [modelsData])

  const handleCreate = async () => {
    if (!newTokenName.trim()) return
    
    try {
      const result = await createToken.mutateAsync({
        name: newTokenName,
        quota: tokenQuota ? parseFloat(tokenQuota) : undefined,
        expires_at: tokenExpiry ? new Date(tokenExpiry) : undefined,
        allowed_ips: tokenIpWhitelist ? tokenIpWhitelist.split(',').map(ip => ip.trim()) : undefined,
        allowed_models: tokenAllowedModels.length > 0 ? tokenAllowedModels : undefined,
        rate_limit: tokenRateLimit ? parseInt(tokenRateLimit) : undefined,
      })
      
      toast({
        title: '创建成功',
        description: '新的 API Key 已创建（仅展示一次，请立即保存）',
      })
      setCreatedTokenKey(result.key)
      setShowCreateModal(false)
      resetForm()
    } catch (error) {
      toast({
        title: '创建失败',
        description: error instanceof Error ? error.message : '请稍后重试',
        variant: 'destructive',
      })
    }
  }

  const handleToggleStatus = async (token: { id: string; status: string }) => {
    try {
      const newStatus = token.status === 'active' ? 'disabled' : 'active'
      await updateTokenStatus.mutateAsync({ id: token.id, status: newStatus })
      toast({
        title: '状态已更新',
        description: `API Key 已${newStatus === 'active' ? '启用' : '禁用'}`,
      })
    } catch (error) {
      toast({
        title: '更新失败',
        description: error instanceof Error ? error.message : '请稍后重试',
        variant: 'destructive',
      })
    }
  }

  const openAdvancedSettings = (token: { id: string; name: string; remain_quota?: number; expires_at?: string; allowed_ips?: string[]; allowed_models?: string[]; rate_limit?: number }) => {
    setSelectedToken(token)
    setTokenQuota(token.remain_quota?.toString() || '')
    setTokenExpiry(token.expires_at ? new Date(token.expires_at).toISOString().split('T')[0] : '')
    setTokenIpWhitelist(token.allowed_ips?.join(', ') || '')
    setTokenAllowedModels(token.allowed_models || [])
    setTokenRateLimit(token.rate_limit?.toString() || '')
    setShowAdvancedSettings(true)
  }

  const handleSaveAdvancedSettings = async () => {
    if (!selectedToken) return
    
    try {
      const allowedIPs = tokenIpWhitelist
        ? tokenIpWhitelist.split(',').map(ip => ip.trim()).filter(Boolean)
        : []
      
      await updateToken.mutateAsync({
        id: selectedToken.id,
        data: {
          quota: tokenQuota ? parseFloat(tokenQuota) : undefined,
          expires_at: tokenExpiry ? new Date(tokenExpiry) : undefined,
          allowed_ips: allowedIPs,
          allowed_models: tokenAllowedModels,
          rate_limit: tokenRateLimit ? parseInt(tokenRateLimit) : undefined,
        }
      })
      
      toast({ title: '设置已保存', description: 'API Key 高级设置已更新' })
      setShowAdvancedSettings(false)
      setSelectedToken(null)
    } catch (error) {
      toast({
        title: '保存失败',
        description: error instanceof Error ? error.message : '请稍后重试',
        variant: 'destructive',
      })
    }
  }

  const handleConfirmDelete = async () => {
    if (!deleteTarget) return
    
    try {
      await deleteToken.mutateAsync(deleteTarget.id)
      toast({ title: '删除成功', description: 'API Key 已删除' })
      setDeleteTarget(null)
    } catch (error) {
      toast({
        title: '删除失败',
        description: error instanceof Error ? error.message : '请稍后重试',
        variant: 'destructive',
      })
    }
  }

  const handleCopy = async (key: string) => {
    if (key.includes('...')) {
      toast({
        title: '无法复制完整 Key',
        description: '出于安全考虑，完整 API Key 仅在创建时展示一次，请在创建后立即保存。',
        variant: 'destructive',
      })
      return
    }
    await copyToClipboard(key)
    toast({ title: '已复制', description: 'API Key 已复制到剪贴板' })
  }

  const toggleKeyVisibility = (id: string) => {
    const newVisible = new Set(visibleKeys)
    if (newVisible.has(id)) {
      newVisible.delete(id)
    } else {
      newVisible.add(id)
    }
    setVisibleKeys(newVisible)
  }

  const maskKey = (key: string) => {
    if (key.length <= 10) return key
    return key.slice(0, 7) + '...' + key.slice(-4)
  }

  const resetForm = () => {
    setNewTokenName('')
    setTokenQuota('')
    setTokenExpiry('')
    setTokenIpWhitelist('')
    setTokenAllowedModels([])
    setTokenRateLimit('')
  }

  // 未登录状态
  if (!isAuthenticated) {
    return (
      <div className="space-y-6 animate-fade-in">
        <div>
          <h1 className="text-2xl font-bold">API Keys</h1>
          <p className="text-muted-foreground">管理您的 API 密钥</p>
        </div>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center py-12">
              <Key className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">需要登录</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                请先登录以管理您的 API Keys
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
          <h1 className="text-2xl font-bold">API Keys</h1>
          <p className="text-muted-foreground">管理您的 API 密钥</p>
        </div>
        <Button onClick={() => {
          setShowCreateModal(true)
          resetForm()
        }}>
          <Plus className="mr-2 h-4 w-4" />
          创建 API Key
        </Button>
      </div>

      {/* API Keys 列表 */}
      <Card>
        <CardHeader>
          <CardTitle>您的 API Keys</CardTitle>
          <CardDescription>
            请妥善保管您的 API Key，不要泄露给他人
          </CardDescription>
        </CardHeader>
        <CardContent>
          {tokensLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : tokens.length === 0 ? (
            <div className="text-center py-12">
              <Key className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">暂无 API Key</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                创建一个 API Key 开始使用
              </p>
              <Button className="mt-4" onClick={() => setShowCreateModal(true)}>
                <Plus className="mr-2 h-4 w-4" />
                创建 API Key
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              {tokens.map((token: { id: string; name: string; status: string; key: string; last_used_at?: string; created_at: string; usage?: { request_count: number; total_tokens: number; total_cost: number } }) => (
                <div
                  key={token.id}
                  className="flex items-center gap-4 p-4 border rounded-lg hover:bg-accent/50 transition-colors"
                >
                  <div className="w-10 h-10 bg-primary/10 rounded-lg flex items-center justify-center">
                    <Key className="h-5 w-5 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h4 className="font-medium">{token.name}</h4>
                      <span
                        className={`px-2 py-0.5 text-xs rounded-full ${
                          token.status === 'active'
                            ? 'bg-green-500/10 text-green-500'
                            : 'bg-red-500/10 text-red-500'
                        }`}
                      >
                        {token.status === 'active' ? '正常' : '已禁用'}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 mt-1">
                      <code className="text-sm text-muted-foreground font-mono">
                        {visibleKeys.has(token.id) ? token.key : maskKey(token.key)}
                      </code>
                      <button
                        onClick={() => toggleKeyVisibility(token.id)}
                        className="text-muted-foreground hover:text-foreground"
                      >
                        {visibleKeys.has(token.id) ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </button>
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      创建于 {formatDate(token.created_at)}
                      {token.last_used_at && ` · 最后使用 ${formatDate(token.last_used_at)}`}
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      用量：{formatNumber(token.usage?.request_count ?? 0)} 次 · {formatNumber(token.usage?.total_tokens ?? 0)} Tokens ·{' '}
                      {formatCurrency(Number(token.usage?.total_cost ?? 0))}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      size="icon"
                      variant="ghost"
                      onClick={() => handleCopy(token.key)}
                      title={token.key.includes('...') ? '出于安全考虑，完整 Key 仅在创建时展示一次' : '复制'}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      onClick={() => handleToggleStatus(token)}
                      title={token.status === 'active' ? '禁用' : '启用'}
                    >
                      {token.status === 'active' ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      onClick={() => openAdvancedSettings(token)}
                      title="高级设置"
                    >
                      <MoreVertical className="h-4 w-4" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setDeleteTarget(token)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <ConfirmDialog
        open={!!deleteTarget}
        title="删除 API Key"
        description={deleteTarget ? `确定要删除「${deleteTarget.name}」吗？删除后将无法恢复。` : undefined}
        confirmText="确认删除"
        cancelText="取消"
        destructive
        loading={deleteToken.isPending}
        onCancel={() => setDeleteTarget(null)}
        onConfirm={handleConfirmDelete}
      />

      {/* 创建 Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
          <Card className="w-full max-w-md animate-fade-in">
            <CardHeader>
              <CardTitle>创建 API Key</CardTitle>
              <CardDescription>
                为您的 API Key 设置一个名称以便识别
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">名称</label>
                <Input
                  placeholder="例如：生产环境、测试环境"
                  value={newTokenName}
                  onChange={(e) => setNewTokenName(e.target.value)}
                />
              </div>
              
              {/* 高级设置 */}
              <div className="space-y-4 pt-4 border-t">
                <h4 className="text-sm font-medium mb-3">高级设置（可选）</h4>
                
                <div className="space-y-2">
                  <label className="text-sm font-medium">配额限制（金额）</label>
                  <Input
                    type="number"
                    placeholder="例如：100.00"
                    value={tokenQuota}
                    onChange={(e) => setTokenQuota(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">留空表示无限制（单位与账户余额一致）</p>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">过期时间</label>
                  <Input
                    type="date"
                    value={tokenExpiry}
                    onChange={(e) => setTokenExpiry(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">留空表示永不过期</p>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">IP 白名单</label>
                  <Input
                    placeholder="例如：192.168.1.1, 10.0.0.1"
                    value={tokenIpWhitelist}
                    onChange={(e) => setTokenIpWhitelist(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">多个 IP 用逗号分隔，留空表示不限制</p>
                </div>

                <div className="space-y-2">
                  <AllowedModelsPicker
                    selected={tokenAllowedModels}
                    onChange={setTokenAllowedModels}
                    availableModels={availableModels}
                    loading={false}
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">速率限制（每分钟请求数，可选）</label>
                  <Input
                    type="number"
                    placeholder="例如：60"
                    value={tokenRateLimit}
                    onChange={(e) => setTokenRateLimit(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">留空表示不限制</p>
                </div>
              </div>

              <div className="flex justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowCreateModal(false)
                    resetForm()
                  }}
                >
                  取消
                </Button>
                <Button 
                  onClick={handleCreate} 
                  disabled={createToken.isPending || !newTokenName.trim()}
                >
                  {createToken.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      创建中...
                    </>
                  ) : '创建'}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 创建成功后的 Key 展示（仅展示一次） */}
      {createdTokenKey && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
          <Card className="w-full max-w-md animate-fade-in">
            <CardHeader>
              <CardTitle>请保存您的 API Key</CardTitle>
              <CardDescription>完整 Key 仅展示一次，请立即复制并妥善保存</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-2 p-3 border rounded-lg bg-muted/20">
                <code className="text-sm font-mono break-all flex-1">{createdTokenKey}</code>
                <Button size="icon" variant="ghost" onClick={() => handleCopy(createdTokenKey)}>
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
              <div className="flex justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    setGatewayApiKey(createdTokenKey)
                    toast({
                      title: '已设置默认 Key',
                      description: '图片/视频生成将使用该 Key 调用网关 /v1 接口',
                    })
                  }}
                >
                  设为默认测试 Key
                </Button>
                <Button
                  onClick={() => {
                    setCreatedTokenKey(null)
                  }}
                >
                  我已保存
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 高级设置弹窗 */}
      {showAdvancedSettings && selectedToken && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
          <Card className="w-full max-w-md animate-fade-in">
            <CardHeader>
              <CardTitle>高级设置</CardTitle>
              <CardDescription>
                配置 API Key 的高级选项
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">配额限制（金额）</label>
                <Input
                  type="number"
                  placeholder="例如：100.00"
                  value={tokenQuota}
                  onChange={(e) => setTokenQuota(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">留空表示无限制（单位与账户余额一致）</p>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">过期时间</label>
                <Input
                  type="date"
                  value={tokenExpiry}
                  onChange={(e) => setTokenExpiry(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">留空表示永不过期</p>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">IP 白名单</label>
                <Input
                  placeholder="例如：192.168.1.1, 10.0.0.1"
                  value={tokenIpWhitelist}
                  onChange={(e) => setTokenIpWhitelist(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">多个 IP 用逗号分隔，留空表示不限制</p>
              </div>

              <div className="space-y-2">
                <AllowedModelsPicker
                  selected={tokenAllowedModels}
                  onChange={setTokenAllowedModels}
                  availableModels={availableModels}
                  loading={false}
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">速率限制（每分钟请求数，可选）</label>
                <Input
                  type="number"
                  placeholder="例如：60"
                  value={tokenRateLimit}
                  onChange={(e) => setTokenRateLimit(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">留空表示不限制</p>
              </div>

              <div className="flex justify-end gap-2 pt-4">
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowAdvancedSettings(false)
                    setSelectedToken(null)
                  }}
                >
                  取消
                </Button>
                <Button 
                  onClick={handleSaveAdvancedSettings} 
                  disabled={updateToken.isPending}
                >
                  {updateToken.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      保存中...
                    </>
                  ) : '保存'}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}