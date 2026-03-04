'use client'

import { useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useToast } from '@/components/ui/use-toast'
import { copyToClipboard } from '@/lib/utils'
import { getGatewayApiKey, setGatewayApiKey } from '@/lib/api'
import { Copy, Eye, EyeOff, Key, Trash2 } from 'lucide-react'

export function GatewayApiKeyManager() {
  const { toast } = useToast()
  const [storedKey, setStoredKey] = useState<string | null>(null)
  const [inputKey, setInputKey] = useState('')
  const [showKey, setShowKey] = useState(false)

  useEffect(() => {
    setStoredKey(getGatewayApiKey())
  }, [])

  const maskedKey = useMemo(() => {
    if (!storedKey) return ''
    if (showKey) return storedKey
    if (storedKey.length <= 12) return storedKey
    return `${storedKey.slice(0, 8)}...${storedKey.slice(-4)}`
  }, [storedKey, showKey])

  const handleSave = () => {
    const key = inputKey.trim()
    if (!key) return
    if (!key.startsWith('sk-')) {
      toast({
        title: 'API Key 格式不正确',
        description: '请填写以 sk- 开头的 API Key',
        variant: 'destructive',
      })
      return
    }

    setGatewayApiKey(key)
    setStoredKey(key)
    setInputKey('')
    toast({
      title: '已保存',
      description: '默认 API Key 已保存到本地浏览器',
    })
  }

  const handleClear = () => {
    setGatewayApiKey(null)
    setStoredKey(null)
    setShowKey(false)
    toast({
      title: '已清除',
      description: '默认 API Key 已从本地移除',
    })
  }

  const handleCopy = async () => {
    if (!storedKey) return
    await copyToClipboard(storedKey)
    toast({
      title: '已复制',
      description: 'API Key 已复制到剪贴板',
    })
  }

  return (
    <Card className="border-primary/20 bg-gradient-to-br from-primary/5 to-background">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Key className="h-5 w-5" />
          默认 API Key（用于 /v1 网关调用）
        </CardTitle>
        <CardDescription>
          图片/视频生成会通过网关调用，需要设置一个 API Key（仅保存到当前浏览器 localStorage）。
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {storedKey ? (
          <div className="space-y-2">
            <label className="text-sm font-medium">当前默认 Key</label>
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded-md bg-muted px-3 py-2 text-sm overflow-x-auto">
                {maskedKey}
              </code>
              <Button size="icon" variant="outline" onClick={() => setShowKey(!showKey)}>
                {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
              <Button size="icon" variant="outline" onClick={handleCopy}>
                <Copy className="h-4 w-4" />
              </Button>
              <Button size="icon" variant="destructive" onClick={handleClear}>
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              出于安全，列表页仅展示脱敏 Key；如需创建新的 Key，请前往{' '}
              <Link className="underline underline-offset-2 hover:text-foreground" href="/dashboard/tokens">
                API Keys
              </Link>
              。
            </p>
          </div>
        ) : (
          <div className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
            未设置默认 API Key。请先在下方粘贴保存，或前往{' '}
            <Link className="underline underline-offset-2 hover:text-foreground" href="/dashboard/tokens">
              API Keys
            </Link>{' '}
            创建。
          </div>
        )}

        <div className="space-y-2">
          <label className="text-sm font-medium">设置/更新 Key</label>
          <div className="flex gap-2">
            <Input
              placeholder="sk-..."
              value={inputKey}
              onChange={(e) => setInputKey(e.target.value)}
            />
            <Button onClick={handleSave} disabled={!inputKey.trim()}>
              保存
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">
            注意：该 Key 会用于向网关发起请求（请求头 Authorization），请勿在不可信设备上保存。
          </p>
        </div>
      </CardContent>
    </Card>
  )
}

