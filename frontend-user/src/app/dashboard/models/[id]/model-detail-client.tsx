'use client'

import { useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import { useAuthStore } from '@/store/auth'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { GatewayApiKeyManager } from '@/components/gateway-api-key-manager'
import { toast } from '@/components/ui/use-toast'
import {
  API_BASE_URL,
  chatApi,
  imageApi,
  videoApi,
  publicApi,
  getGatewayApiKey,
  type ChatCompletionResponse,
  type GenerationTaskResponse,
  type ImageGenerationRequest,
  type Model,
  type VideoGenerationRequest,
} from '@/lib/api'
import { copyToClipboard, formatDate, formatNumber } from '@/lib/utils'
import { formatModelPricing, getModelCategory, getModelCategoryLabel } from '@/lib/model-utils'
import {
  ArrowLeft,
  CheckCircle2,
  Copy,
  ExternalLink,
  Image as ImageIcon,
  Loader2,
  MessageSquareText,
  Music,
  Video as VideoIcon,
  XCircle,
} from 'lucide-react'

function StatusPill({ enabled }: { enabled: boolean }) {
  return (
    <div
      className={[
        'inline-flex items-center gap-1 rounded-full px-2 py-1 text-xs font-medium',
        enabled ? 'bg-green-500/10 text-green-600' : 'bg-muted text-muted-foreground',
      ].join(' ')}
    >
      {enabled ? <CheckCircle2 className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}
      {enabled ? '可用' : '不可用'}
    </div>
  )
}

function CodeBlock({ title, code }: { title: string; code: string }) {
  const onCopy = async () => {
    await copyToClipboard(code)
    toast({ title: '已复制', description: title })
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <div className="text-sm font-medium">{title}</div>
        <Button size="sm" variant="outline" className="h-8 gap-2" onClick={onCopy}>
          <Copy className="w-4 h-4" />
          复制
        </Button>
      </div>
      <pre className="rounded-md bg-muted p-3 text-sm overflow-x-auto">
        {code}
      </pre>
    </div>
  )
}

function CategoryIcon({ category }: { category: ReturnType<typeof getModelCategory> }) {
  if (category === 'image') return <ImageIcon className="w-4 h-4" />
  if (category === 'video') return <VideoIcon className="w-4 h-4" />
  if (category === 'audio') return <Music className="w-4 h-4" />
  return <MessageSquareText className="w-4 h-4" />
}

export default function ModelDetailClient({ id }: { id: string }) {
  const modelId = useMemo(() => decodeURIComponent(id), [id])
  const { isAuthenticated } = useAuthStore()

  const [loading, setLoading] = useState(true)
  const [model, setModel] = useState<Model | null>(null)

  const [chatPrompt, setChatPrompt] = useState('你好，简单介绍一下你自己。')
  const [chatLoading, setChatLoading] = useState(false)
  const [chatResult, setChatResult] = useState<string | null>(null)
  const [chatUsage, setChatUsage] = useState<ChatCompletionResponse['usage'] | null>(null)

  const [imagePrompt, setImagePrompt] = useState('一只戴着墨镜的橘猫，赛博朋克风格，超高清')
  const [imageLoading, setImageLoading] = useState(false)
  const [imageUrls, setImageUrls] = useState<string[]>([])

  const [videoPrompt, setVideoPrompt] = useState('一座未来城市的延时摄影，霓虹灯闪烁，电影质感')
  const [videoLoading, setVideoLoading] = useState(false)
  const [videoUrl, setVideoUrl] = useState<string | null>(null)
  const [currentTask, setCurrentTask] = useState<GenerationTaskResponse | null>(null)

  const [ttsInput, setTtsInput] = useState('你好，我是 Nexus API。')
  const [ttsVoice, setTtsVoice] = useState('alloy')
  const [ttsFormat, setTtsFormat] = useState<'mp3' | 'wav' | 'opus'>('mp3')
  const [ttsSpeed, setTtsSpeed] = useState('')
  const [ttsLoading, setTtsLoading] = useState(false)
  const [ttsAudioUrl, setTtsAudioUrl] = useState<string | null>(null)

  const [asrFile, setAsrFile] = useState<File | null>(null)
  const [asrLoading, setAsrLoading] = useState(false)
  const [asrResult, setAsrResult] = useState<string | null>(null)

  useEffect(() => {
    return () => {
      if (ttsAudioUrl) URL.revokeObjectURL(ttsAudioUrl)
    }
  }, [ttsAudioUrl])

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      try {
        const res = await publicApi.getModels()
        if (res.code !== 0) {
          setModel(null)
          toast({
            title: '加载失败',
            description: res.message || '无法获取模型详情，请稍后重试',
            variant: 'destructive',
          })
          return
        }
        const list = res.data || []
        const found = list.find((m) => m.id === modelId || m.name === modelId) || null
        setModel(found)
      } catch {
        toast({
          title: '加载失败',
          description: '无法获取模型详情，请稍后重试',
          variant: 'destructive',
        })
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [modelId])

  const category = useMemo(() => (model ? getModelCategory(model) : 'text'), [model])
  const pricing = useMemo(() => (model ? formatModelPricing(model) : null), [model])

  const ensureCanCallGateway = () => {
    if (!isAuthenticated) {
      toast({
        title: '需要登录',
        description: '请先登录后再发起任务',
        variant: 'destructive',
      })
      return false
    }
    if (!getGatewayApiKey()) {
      toast({
        title: '需要 API Key',
        description: '请先在下方设置默认 API Key，用于调用网关 /v1 接口',
        variant: 'destructive',
      })
      return false
    }
    return true
  }

  const runChatTest = async () => {
    if (!model) return
    if (!ensureCanCallGateway()) return
    const prompt = chatPrompt.trim()
    if (!prompt) {
      toast({ title: '请输入内容', description: '提示词不能为空', variant: 'destructive' })
      return
    }

    setChatLoading(true)
    setChatResult(null)
    setChatUsage(null)
    try {
      const res = await chatApi.chatCompletions({
        model: model.name,
        messages: [{ role: 'user', content: prompt }],
        stream: false,
      })
      const content = res.choices?.[0]?.message?.content ?? ''
      setChatResult(content || '（无内容返回）')
      setChatUsage(res.usage ?? null)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : '请求失败，请稍后重试'
      toast({ title: '调用失败', description: message, variant: 'destructive' })
    } finally {
      setChatLoading(false)
    }
  }

  const runImageTest = async () => {
    if (!model) return
    if (!ensureCanCallGateway()) return
    const prompt = imagePrompt.trim()
    if (!prompt) {
      toast({ title: '请输入提示词', description: '提示词不能为空', variant: 'destructive' })
      return
    }

    setImageLoading(true)
    setImageUrls([])
    try {
      const req: ImageGenerationRequest = {
        model: model.name,
        prompt,
        n: 1,
        response_format: 'url',
        resolution: '1K',
        aspect_ratio: '1:1',
      }
      const res = await imageApi.generate(req)
      const urls = (res.data || []).map((d) => d.url || '').filter(Boolean)
      setImageUrls(urls)
      if (urls.length === 0) {
        toast({ title: '生成完成', description: '未返回图片 URL，请检查服务端配置', variant: 'destructive' })
      }
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : '生成失败，请稍后重试'
      toast({ title: '生成失败', description: message, variant: 'destructive' })
    } finally {
      setImageLoading(false)
    }
  }

  const pollVideoTask = async (taskId: string) => {
    const maxAttempts = 180
    let attempts = 0

    const poll = async () => {
      if (attempts >= maxAttempts) {
        toast({
          title: '生成超时',
          description: '视频生成时间较长，请稍后在历史记录中查看',
          variant: 'destructive',
        })
        setVideoLoading(false)
        return
      }

      try {
        const task = await videoApi.getTaskStatus(taskId)
        setCurrentTask(task)
        if (task.status === 'completed' && task.result_url) {
          setVideoUrl(task.result_url)
          setVideoLoading(false)
          return
        }
        if (task.status === 'failed') {
          toast({
            title: '生成失败',
            description: task.error_message || '视频生成失败',
            variant: 'destructive',
          })
          setVideoLoading(false)
          return
        }
      } catch {
      }

      attempts += 1
      setTimeout(poll, 5000)
    }

    poll()
  }

  const runVideoTest = async () => {
    if (!model) return
    if (!ensureCanCallGateway()) return
    const prompt = videoPrompt.trim()
    if (!prompt) {
      toast({ title: '请输入提示词', description: '提示词不能为空', variant: 'destructive' })
      return
    }

    setVideoLoading(true)
    setVideoUrl(null)
    setCurrentTask(null)
    try {
      const req: VideoGenerationRequest = {
        model: model.name,
        prompt,
        aspect_ratio: '9:16',
        duration: 10,
        size: 'small',
      }
      const res = await videoApi.generate(req)
      if (res.status === 'completed' && res.data?.url) {
        setVideoUrl(res.data.url)
        setVideoLoading(false)
        return
      }
      if (res.status === 'processing') {
        setCurrentTask({
          id: res.id,
          type: 'video',
          model: model.name,
          status: res.status,
          progress: res.progress,
          cost: 0,
          duration: 0,
          created_at: new Date().toISOString(),
        })
        pollVideoTask(res.id)
        return
      }
      if (res.error) {
        throw new Error(res.error)
      }

      toast({ title: '生成失败', description: '未返回可识别的任务状态', variant: 'destructive' })
      setVideoLoading(false)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : '生成失败，请稍后重试'
      toast({ title: '生成失败', description: message, variant: 'destructive' })
      setVideoLoading(false)
    }
  }

  const runTtsTest = async () => {
    if (!model) return
    if (!ensureCanCallGateway()) return

    const input = ttsInput.trim()
    if (!input) {
      toast({ title: '请输入内容', description: '文本不能为空', variant: 'destructive' })
      return
    }

    setTtsLoading(true)
    setTtsAudioUrl(null)
    try {
      const apiKey = getGatewayApiKey()
      const payload = {
        model: model.name,
        input,
        voice: ttsVoice,
        response_format: ttsFormat,
        ...(ttsSpeed.trim() ? { speed: Number(ttsSpeed) } : {}),
      }

      const res = await fetch(`${API_BASE_URL}/v1/audio/speech`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${apiKey}`,
        },
        body: JSON.stringify(payload),
      })

      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || `HTTP ${res.status}`)
      }

      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      setTtsAudioUrl(url)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : '生成失败，请稍后重试'
      toast({ title: '语音合成失败', description: message, variant: 'destructive' })
    } finally {
      setTtsLoading(false)
    }
  }

  const runAsrTest = async () => {
    if (!model) return
    if (!ensureCanCallGateway()) return
    if (!asrFile) {
      toast({ title: '请选择音频文件', description: '需要上传一个音频文件用于转写', variant: 'destructive' })
      return
    }

    setAsrLoading(true)
    setAsrResult(null)
    try {
      const apiKey = getGatewayApiKey()
      const form = new FormData()
      form.append('file', asrFile)
      form.append('model', model.name)

      const res = await fetch(`${API_BASE_URL}/v1/audio/transcriptions`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${apiKey}`,
        },
        body: form,
      })

      const text = await res.text()
      if (!res.ok) {
        throw new Error(text || `HTTP ${res.status}`)
      }

      try {
        const json = JSON.parse(text) as { text?: string }
        setAsrResult(json.text || text)
      } catch {
        setAsrResult(text)
      }
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : '转写失败，请稍后重试'
      toast({ title: '音频转写失败', description: message, variant: 'destructive' })
    } finally {
      setAsrLoading(false)
    }
  }

  const copyModelName = async () => {
    if (!model) return
    await copyToClipboard(model.name)
    toast({ title: '已复制', description: model.name })
  }

  const curlBase = 'https://<YOUR_HOST>'
  const curlAuthHeader = 'Authorization: Bearer sk-xxxxxxxx'

  const chatCurl = model
    ? `curl ${curlBase}/v1/chat/completions \\\n  -H "${curlAuthHeader}" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "${model.name}",\n    "messages": [{ "role": "user", "content": "你好" }]\n  }'`
    : ''

  const imageCurl = model
    ? `curl ${curlBase}/v1/images/generations \\\n  -H "${curlAuthHeader}" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "${model.name}",\n    "prompt": "一只戴着墨镜的橘猫，赛博朋克风格"\n  }'`
    : ''

  const videoCurl = model
    ? `curl ${curlBase}/v1/videos/generations \\\n  -H "${curlAuthHeader}" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "${model.name}",\n    "prompt": "一座未来城市的延时摄影，电影质感",\n    "duration": 10,\n    "aspect_ratio": "9:16",\n    "size": "small"\n  }'`
    : ''

  const audioSpeechCurl = model
    ? `curl ${curlBase}/v1/audio/speech \\\n  -H "${curlAuthHeader}" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "${model.name}",\n    "input": "你好，我是 Nexus API。",\n    "voice": "alloy",\n    "response_format": "mp3"\n  }' \\\n  --output speech.mp3`
    : ''

  const audioAsrCurl = model
    ? `curl ${curlBase}/v1/audio/transcriptions \\\n  -H "${curlAuthHeader}" \\\n  -F "file=@audio.mp3" \\\n  -F "model=${model.name}"`
    : ''

  if (loading) {
    return (
      <div className="space-y-6 animate-fade-in">
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="w-4 h-4 animate-spin" />
          加载中...
        </div>
      </div>
    )
  }

  if (!model) {
    return (
      <div className="space-y-6 animate-fade-in">
        <div className="flex items-center justify-between">
          <Link href="/dashboard/models">
            <Button variant="outline" className="gap-2">
              <ArrowLeft className="w-4 h-4" />
              返回模型列表
            </Button>
          </Link>
        </div>
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            未找到该模型：<span className="font-mono">{modelId}</span>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <Link href="/dashboard/models">
          <Button variant="outline" className="gap-2">
            <ArrowLeft className="w-4 h-4" />
            返回模型列表
          </Button>
        </Link>

        <div className="flex items-center gap-2">
          <Button variant="outline" className="gap-2" onClick={copyModelName}>
            <Copy className="w-4 h-4" />
            复制调用名
          </Button>
          {category === 'image' && (
            <Link href="/dashboard/images">
              <Button className="gap-2">
                <ImageIcon className="w-4 h-4" />
                去图片生成
              </Button>
            </Link>
          )}
          {category === 'video' && (
            <Link href="/dashboard/videos">
              <Button className="gap-2">
                <VideoIcon className="w-4 h-4" />
                去视频生成
              </Button>
            </Link>
          )}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-2xl flex items-center gap-2">
            <span className="inline-flex items-center gap-2">
              <CategoryIcon category={category} />
              {model.display_name || model.name}
            </span>
            <StatusPill enabled={model.enabled} />
          </CardTitle>
          <CardDescription>
            {model.provider} · <span className="font-mono">{model.name}</span>
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <div className="text-sm text-muted-foreground">分类</div>
            <div className="text-sm font-medium flex items-center gap-2">
              <CategoryIcon category={category} />
              {getModelCategoryLabel(category)}
            </div>
          </div>

          <div className="space-y-2">
            <div className="text-sm text-muted-foreground">价格</div>
            <div className="text-sm font-medium">
              {pricing ? (
                <>
                  输入 {pricing.input} / {pricing.unitLabel} · 输出 {pricing.output} / {pricing.unitLabel}
                </>
              ) : (
                '—'
              )}
            </div>
          </div>

          <div className="space-y-2">
            <div className="text-sm text-muted-foreground">上下文</div>
            <div className="text-sm font-medium">
              {Math.max(model.context_length || 0, model.max_context || 0).toLocaleString()}
            </div>
          </div>

          <div className="space-y-2">
            <div className="text-sm text-muted-foreground">创建时间</div>
            <div className="text-sm font-medium">{formatDate(model.created_at)}</div>
          </div>

          <div className="md:col-span-2 space-y-2">
            <div className="text-sm text-muted-foreground">描述</div>
            <div className="text-sm">{model.description || '暂无描述'}</div>
          </div>
        </CardContent>
      </Card>

      <GatewayApiKeyManager />

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">调用信息</CardTitle>
          <CardDescription>OpenAI 兼容接口（/v1），可直接用于 SDK / curl 调用</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {category === 'text' && <CodeBlock title="curl：聊天完成 /v1/chat/completions" code={chatCurl} />}
          {category === 'image' && <CodeBlock title="curl：图片生成 /v1/images/generations" code={imageCurl} />}
          {category === 'video' && <CodeBlock title="curl：视频生成 /v1/videos/generations" code={videoCurl} />}
          {category === 'audio' && (
            <div className="space-y-4">
              <CodeBlock title="curl：语音合成 /v1/audio/speech" code={audioSpeechCurl} />
              <CodeBlock title="curl：音频转写 /v1/audio/transcriptions" code={audioAsrCurl} />
            </div>
          )}
          <div className="text-xs text-muted-foreground">
            提示：上述示例中的 <span className="font-mono">{curlBase}</span> 请替换为你的部署域名；
            <span className="font-mono"> Authorization</span> 使用你创建的 API Key。
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">在线测试 / 发起任务</CardTitle>
          <CardDescription>
            该功能会使用你设置的默认 API Key 调用网关 /v1 接口（未登录仅可查看，发起任务需登录）。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {category === 'text' && (
            <div className="space-y-3">
              <div className="space-y-2">
                <div className="text-sm font-medium">提示词</div>
                <Input value={chatPrompt} onChange={(e) => setChatPrompt(e.target.value)} />
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button onClick={runChatTest} disabled={chatLoading}>
                  {chatLoading ? '请求中...' : '发送'}
                </Button>
              </div>

              {chatResult && (
                <Card className="border-dashed">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">响应</CardTitle>
                    {chatUsage && (
                      <CardDescription>
                        Token：{formatNumber(chatUsage.total_tokens)}（输入 {formatNumber(chatUsage.prompt_tokens)} / 输出 {formatNumber(chatUsage.completion_tokens)}）
                      </CardDescription>
                    )}
                  </CardHeader>
                  <CardContent className="text-sm whitespace-pre-wrap">{chatResult}</CardContent>
                </Card>
              )}
            </div>
          )}

          {category === 'image' && (
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="text-sm font-medium">提示词</div>
                <Input value={imagePrompt} onChange={(e) => setImagePrompt(e.target.value)} />
              </div>
              <div className="flex items-center justify-end">
                <Button onClick={runImageTest} disabled={imageLoading}>
                  {imageLoading ? '生成中...' : '生成图片'}
                </Button>
              </div>

              {imageLoading && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="w-4 h-4 animate-spin" />
                  正在生成...
                </div>
              )}

              {imageUrls.length > 0 && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {imageUrls.map((url, index) => (
                    <div key={url + index} className="relative group">
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img src={url} alt={`Generated ${index + 1}`} className="w-full rounded-lg border border-border" />
                      <div className="absolute top-2 right-2">
                        <a href={url} target="_blank" rel="noreferrer">
                          <Button size="sm" variant="secondary" className="h-8 gap-2">
                            查看
                            <ExternalLink className="w-4 h-4" />
                          </Button>
                        </a>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {category === 'video' && (
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="text-sm font-medium">提示词</div>
                <Input value={videoPrompt} onChange={(e) => setVideoPrompt(e.target.value)} />
              </div>
              <div className="flex items-center justify-end">
                <Button onClick={runVideoTest} disabled={videoLoading}>
                  {videoLoading ? '生成中...' : '生成视频'}
                </Button>
              </div>

              {currentTask && (
                <div className="rounded-lg border p-4 space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">状态：{currentTask.status}</span>
                    <span>{Math.round((currentTask.progress || 0) * 100)}%</span>
                  </div>
                  <div className="w-full bg-muted rounded-full h-2">
                    <div
                      className="bg-primary h-2 rounded-full transition-all duration-300"
                      style={{ width: `${Math.round((currentTask.progress || 0) * 100)}%` }}
                    />
                  </div>
                </div>
              )}

              {videoUrl && (
                <div className="space-y-2">
                  <video src={videoUrl} controls className="w-full rounded-lg border border-border" />
                </div>
              )}
            </div>
          )}

          {category === 'audio' && (
            <div className="space-y-6">
              <div className="space-y-3">
                <div className="text-sm font-medium">语音合成（TTS）</div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                  <div className="space-y-2">
                    <div className="text-sm text-muted-foreground">Voice</div>
                    <select
                      value={ttsVoice}
                      onChange={(e) => setTtsVoice(e.target.value)}
                      className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                    >
                      {['alloy', 'echo', 'fable', 'onyx', 'nova', 'shimmer'].map((voice) => (
                        <option key={voice} value={voice}>
                          {voice}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div className="space-y-2">
                    <div className="text-sm text-muted-foreground">格式</div>
                    <select
                      value={ttsFormat}
                      onChange={(e) => setTtsFormat(e.target.value as typeof ttsFormat)}
                      className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                    >
                      <option value="mp3">mp3</option>
                      <option value="wav">wav</option>
                      <option value="opus">opus</option>
                    </select>
                  </div>

                  <div className="space-y-2">
                    <div className="text-sm text-muted-foreground">Speed（可选）</div>
                    <Input
                      value={ttsSpeed}
                      onChange={(e) => setTtsSpeed(e.target.value)}
                      placeholder="例如：1.0"
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="text-sm text-muted-foreground">文本</div>
                  <Input value={ttsInput} onChange={(e) => setTtsInput(e.target.value)} />
                </div>

                <div className="flex items-center justify-end">
                  <Button onClick={runTtsTest} disabled={ttsLoading}>
                    {ttsLoading ? '生成中...' : '生成语音'}
                  </Button>
                </div>

                {ttsAudioUrl && (
                  <div className="space-y-2">
                    <audio controls src={ttsAudioUrl} className="w-full" />
                    <a
                      href={ttsAudioUrl}
                      download={`speech.${ttsFormat}`}
                      className="text-sm text-primary hover:underline underline-offset-2"
                    >
                      下载音频
                    </a>
                  </div>
                )}
              </div>

              <div className="space-y-3">
                <div className="text-sm font-medium">音频转写（ASR）</div>
                <div className="space-y-2">
                  <div className="text-sm text-muted-foreground">上传音频文件</div>
                  <input
                    type="file"
                    accept="audio/*"
                    onChange={(e) => setAsrFile(e.target.files?.[0] || null)}
                    className="block w-full text-sm text-muted-foreground file:mr-4 file:rounded-md file:border-0 file:bg-muted file:px-3 file:py-2 file:text-sm file:font-medium hover:file:bg-accent"
                  />
                </div>
                <div className="flex items-center justify-end">
                  <Button onClick={runAsrTest} disabled={asrLoading || !asrFile}>
                    {asrLoading ? '转写中...' : '开始转写'}
                  </Button>
                </div>

                {asrResult && (
                  <Card className="border-dashed">
                    <CardHeader className="pb-3">
                      <CardTitle className="text-base">转写结果</CardTitle>
                      <CardDescription>仅展示文本结果；如需更多字段可扩展展示。</CardDescription>
                    </CardHeader>
                    <CardContent className="text-sm whitespace-pre-wrap">{asrResult}</CardContent>
                  </Card>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
