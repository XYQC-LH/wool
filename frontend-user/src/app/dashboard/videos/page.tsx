'use client'

import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useToast } from '@/components/ui/use-toast'
import { useAuthStore } from '@/store/auth'
import { videoApi, publicApi, VideoGenerationRequest, GenerationTaskResponse, Model, getGatewayApiKey } from '@/lib/api'
import { handleApiError } from '@/lib/error-handler'
import { getModelCategory } from '@/lib/model-utils'
import { Loader2, Download, RefreshCw, Video as VideoIcon, Sparkles, Play } from 'lucide-react'
import { ReferenceImageUpload } from '@/components/reference-image-upload'
import { GatewayApiKeyManager } from '@/components/gateway-api-key-manager'

// 视频生成模型配置接口
interface VideoModelConfig {
  id: string
  name: string
  provider: string
  description: string
  aspectRatios: string[]
  durations: number[]
  sizes: string[]
}

export default function VideosPage() {
  const { toast } = useToast()
  const { isAuthenticated } = useAuthStore()
  const [isGenerating, setIsGenerating] = useState(false)
  const [prompt, setPrompt] = useState('')
  const [selectedModel, setSelectedModel] = useState<VideoModelConfig | null>(null)
  const [aspectRatio, setAspectRatio] = useState('9:16')
  const [duration, setDuration] = useState(10)
  const [size, setSize] = useState('small')
  const [referenceImagePreview, setReferenceImagePreview] = useState<string | null>(null)
  const [generatedVideo, setGeneratedVideo] = useState<string | null>(null)
  const [history, setHistory] = useState<GenerationTaskResponse[]>([])
  const [isLoadingHistory, setIsLoadingHistory] = useState(false)
  const [currentTask, setCurrentTask] = useState<GenerationTaskResponse | null>(null)
  const [isLoadingModels, setIsLoadingModels] = useState(false)
  const [videoModels, setVideoModels] = useState<VideoModelConfig[]>([])

  const loadModels = useCallback(async () => {
    setIsLoadingModels(true)
    try {
      const response = await publicApi.getModels()
      if (response.code !== 0) {
        toast({
          title: '加载模型失败',
          description: response.message || '请求失败',
          variant: 'destructive',
        })
        return
      }

      const models = ((response.data || []) as Model[])
        .filter((model) => model.enabled)
        .filter((model) => getModelCategory(model) === 'video')
        .map((model) => ({
          id: model.id,
          name: model.name,
          provider: model.provider || 'Unknown',
          description: model.description || '',
          aspectRatios: ['9:16', '16:9', '1:1'],
          durations: [10, 15],
          sizes: ['small', 'large'],
        }))
      
      setVideoModels(models)
    } catch (error) {
      handleApiError(error, { customMessage: '无法获取可用模型列表' })
    } finally {
      setIsLoadingModels(false)
    }
  }, [toast])

  const loadHistory = useCallback(async () => {
    if (!isAuthenticated) return
    if (!getGatewayApiKey()) {
      setHistory([])
      return
    }
    
    setIsLoadingHistory(true)
    try {
      const response = await videoApi.listTasks({ type: 'video', page: 1, page_size: 20 })
      setHistory(response.data || [])
    } catch (error) {
      handleApiError(error, { customMessage: '加载历史记录失败，请稍后重试' })
    } finally {
      setIsLoadingHistory(false)
    }
  }, [isAuthenticated])

  // 加载模型列表
  useEffect(() => {
    loadModels()
  }, [isAuthenticated, loadModels])

  // 加载历史记录
  useEffect(() => {
    if (isAuthenticated) {
      loadHistory()
    } else {
      setHistory([])
    }
  }, [isAuthenticated, loadHistory])

  // 默认选择第一个模型
  useEffect(() => {
    if (selectedModel || videoModels.length === 0) return
    const first = videoModels[0]
    setSelectedModel(first)
    setAspectRatio(first.aspectRatios[0])
    setDuration(first.durations[0])
    setSize(first.sizes[0])
  }, [selectedModel, videoModels])

  // 生成视频
  const handleGenerate = async () => {
    if (!isAuthenticated) {
      toast({
        title: '需要登录',
        description: '请先登录以使用视频生成功能',
        variant: 'destructive',
      })
      return
    }

    if (!getGatewayApiKey()) {
      toast({
        title: '需要 API Key',
        description: '请先在页面顶部设置默认 API Key，用于调用网关 /v1 接口',
        variant: 'destructive',
      })
      return
    }

    if (!prompt.trim()) {
      toast({
        title: '请输入提示词',
        description: '提示词不能为空',
        variant: 'destructive',
      })
      return
    }

    if (!selectedModel) {
      toast({
        title: '请选择模型',
        description: '请先选择一个视频生成模型',
        variant: 'destructive',
      })
      return
    }

    setIsGenerating(true)
    setGeneratedVideo(null)
    setCurrentTask(null)

    try {
      const request: VideoGenerationRequest = {
        model: selectedModel.name,
        prompt: prompt.trim(),
        aspect_ratio: aspectRatio,
        duration: duration,
        size: size,
        image_url: referenceImagePreview || undefined,
      }

      const response = await videoApi.generate(request)
      
      if (response.status === 'completed' && response.data?.url) {
        setGeneratedVideo(response.data.url)
        toast({
          title: '生成成功',
          description: '视频生成完成',
        })
        // 刷新历史记录
        loadHistory()
      } else if (response.status === 'processing') {
        // 任务正在处理中，开始轮询
        setCurrentTask({
          id: response.id,
          type: 'video',
          model: selectedModel.name,
          status: response.status,
          progress: response.progress,
          cost: 0,
          duration: 0,
          created_at: new Date().toISOString(),
        })
        pollTaskStatus(response.id)
      } else if (response.error) {
        throw new Error(response.error)
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : '视频生成失败，请稍后重试'
      toast({
        title: '生成失败',
        description: errorMessage,
        variant: 'destructive',
      })
      setIsGenerating(false)
    }
  }

  // 轮询任务状态
  const pollTaskStatus = async (taskId: string) => {
    const maxAttempts = 180 // 最多轮询 15 分钟 (180 * 5秒)
    let attempts = 0

    const poll = async () => {
      if (attempts >= maxAttempts) {
        toast({
          title: '生成超时',
          description: '视频生成时间过长，请稍后查看历史记录',
          variant: 'destructive',
        })
        setIsGenerating(false)
        return
      }

      try {
        const task = await videoApi.getTaskStatus(taskId)
        setCurrentTask(task)

        if (task.status === 'completed' && task.result_url) {
          setGeneratedVideo(task.result_url)
          setIsGenerating(false)
          toast({
            title: '生成成功',
            description: '视频生成完成',
          })
          loadHistory()
        } else if (task.status === 'failed') {
          toast({
            title: '生成失败',
            description: task.error_message || '视频生成失败',
            variant: 'destructive',
          })
          setIsGenerating(false)
        } else {
          // 继续轮询
          attempts++
          setTimeout(poll, 5000) // 5秒后再次查询
        }
      } catch {
        attempts++
        setTimeout(poll, 5000)
      }
    }

    poll()
  }

  // 下载视频
  const handleDownload = async (url: string) => {
    try {
      const response = await fetch(url)
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }
      const blob = await response.blob()
      const downloadUrl = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = downloadUrl
      a.download = `generated-video-${Date.now()}.mp4`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(downloadUrl)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : '无法下载视频'
      toast({
        title: '下载失败',
        description: message,
        variant: 'destructive',
      })
    }
  }

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-3xl font-bold">视频生成</h1>
        <p className="text-muted-foreground mt-1">
          使用 AI 模型生成高质量视频
        </p>
      </div>

      <GatewayApiKeyManager />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 左侧：生成配置 */}
        <div className="lg:col-span-1 space-y-4">
          {/* 模型选择 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">选择模型</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {isLoadingModels ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
                </div>
              ) : videoModels.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <VideoIcon className="w-12 h-12 mx-auto mb-2 opacity-50" />
                  <p>暂无可用模型</p>
                </div>
              ) : (
                videoModels.map((model) => (
                  <div
                    key={model.id}
                    className={`p-3 rounded-lg border cursor-pointer transition-colors ${
                      selectedModel?.id === model.id
                        ? 'border-primary bg-primary/5'
                        : 'border-border hover:border-primary/50'
                    }`}
                    onClick={() => setSelectedModel(model)}
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-medium">{model.name}</span>
                      <span className="text-xs text-muted-foreground">{model.provider}</span>
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">{model.description}</p>
                  </div>
                ))
              )}
            </CardContent>
          </Card>

          {/* 参数配置 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">参数配置</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* 画幅比例 */}
              <div>
                <label className="text-sm font-medium mb-2 block">画幅比例</label>
                <div className="grid grid-cols-3 gap-2">
                  {selectedModel?.aspectRatios.map((ratio) => (
                    <Button
                      key={ratio}
                      variant={aspectRatio === ratio ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setAspectRatio(ratio)}
                    >
                      {ratio}
                    </Button>
                  ))}
                </div>
              </div>

              {/* 时长 */}
              <div>
                <label className="text-sm font-medium mb-2 block">视频时长</label>
                <div className="grid grid-cols-2 gap-2">
                  {selectedModel?.durations.map((dur) => (
                    <Button
                      key={dur}
                      variant={duration === dur ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setDuration(dur)}
                    >
                      {dur}秒
                    </Button>
                  ))}
                </div>
              </div>

              {/* 尺寸 */}
              <div>
                <label className="text-sm font-medium mb-2 block">视频尺寸</label>
                <div className="grid grid-cols-2 gap-2">
                  {selectedModel?.sizes.map((s) => (
                    <Button
                      key={s}
                      variant={size === s ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setSize(s)}
                    >
                      {s === 'small' ? '普清' : '高清'}
                    </Button>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* 右侧：提示词和结果 */}
        <div className="lg:col-span-2 space-y-4">
          {/* 提示词输入 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <Sparkles className="w-5 h-5" />
                提示词
              </CardTitle>
              <CardDescription>
                描述你想要生成的视频内容，包括场景、动作、风格等
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <textarea
                className="w-full min-h-[120px] p-3 rounded-lg border border-border bg-background resize-none focus:outline-none focus:ring-2 focus:ring-primary"
                placeholder="例如：一只金毛犬在海边奔跑，阳光明媚，海浪轻拍沙滩，慢动作镜头，电影质感"
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
              />
              
              {/* 参考图上传 */}
              <div className="space-y-2">
                <label className="text-sm font-medium">参考图片（可选）</label>
                <ReferenceImageUpload
                  preview={referenceImagePreview}
                  onUpload={(file) => {
                    const reader = new FileReader()
                    reader.onloadend = () => {
                      setReferenceImagePreview(reader.result as string)
                    }
                    reader.readAsDataURL(file)
                  }}
                  onRemove={() => {
                    setReferenceImagePreview(null)
                  }}
                />
              </div>
              
              <div className="flex justify-between items-center">
                <span className="text-xs text-muted-foreground">
                  模型: {selectedModel?.name || '未选择'} | 画幅: {aspectRatio} | 时长: {duration}秒 | 尺寸: {size === 'small' ? '普清' : '高清'}
                </span>
                <Button
                  onClick={handleGenerate}
                  disabled={isGenerating || !prompt.trim() || !isAuthenticated}
                >
                  {!isAuthenticated ? (
                    '请先登录'
                  ) : isGenerating ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      生成中...
                    </>
                  ) : (
                    <>
                      <Sparkles className="w-4 h-4 mr-2" />
                      生成视频
                    </>
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* 未登录提示 */}
          {!isAuthenticated && (
            <Card>
              <CardContent className="pt-6">
                <div className="text-center py-8">
                  <VideoIcon className="mx-auto h-12 w-12 text-muted-foreground" />
                  <h3 className="mt-4 text-lg font-medium">需要登录</h3>
                  <p className="mt-2 text-sm text-muted-foreground">
                    请先登录以使用视频生成功能
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
          )}

          {/* 生成进度 */}
          {isGenerating && currentTask && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <Loader2 className="w-5 h-5 animate-spin" />
                  生成进度
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <div className="flex justify-between text-sm">
                    <span>状态: {currentTask.status === 'processing' ? '处理中' : currentTask.status}</span>
                    <span>{Math.round(currentTask.progress * 100)}%</span>
                  </div>
                  <div className="w-full bg-muted rounded-full h-2">
                    <div
                      className="bg-primary h-2 rounded-full transition-all duration-300"
                      style={{ width: `${currentTask.progress * 100}%` }}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground">
                    视频生成可能需要几分钟时间，请耐心等待...
                  </p>
                </div>
              </CardContent>
            </Card>
          )}

          {/* 生成结果 */}
          {generatedVideo && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <VideoIcon className="w-5 h-5" />
                  生成结果
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="relative group">
                  <video
                    src={generatedVideo}
                    controls
                    className="w-full rounded-lg border border-border"
                  />
                  <div className="mt-3 flex justify-end">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDownload(generatedVideo)}
                    >
                      <Download className="w-4 h-4 mr-2" />
                      下载视频
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* 历史记录 */}
          {isAuthenticated && (
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-lg">生成历史</CardTitle>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={loadHistory}
                    disabled={isLoadingHistory}
                  >
                    <RefreshCw className={`w-4 h-4 ${isLoadingHistory ? 'animate-spin' : ''}`} />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                {history.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    <VideoIcon className="w-12 h-12 mx-auto mb-2 opacity-50" />
                    <p>暂无生成记录</p>
                  </div>
                ) : (
                  <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                    {history.map((task) => (
                      <div
                        key={task.id}
                        className="relative aspect-video rounded-lg border border-border overflow-hidden cursor-pointer"
                        onClick={() => task.result_url && setGeneratedVideo(task.result_url)}
                      >
                        {task.result_url ? (
                          <div className="relative w-full h-full bg-muted">
                            <video
                              src={task.result_url}
                              className="w-full h-full object-cover"
                            />
                            <div className="absolute inset-0 flex items-center justify-center bg-black/30 opacity-0 hover:opacity-100 transition-opacity">
                              <Play className="w-8 h-8 text-white" />
                            </div>
                          </div>
                        ) : (
                          <div className="w-full h-full flex items-center justify-center bg-muted">
                            {task.status === 'processing' ? (
                              <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
                            ) : task.status === 'failed' ? (
                              <span className="text-xs text-destructive">失败</span>
                            ) : (
                              <VideoIcon className="w-6 h-6 text-muted-foreground" />
                            )}
                          </div>
                        )}
                        <div className="absolute bottom-0 left-0 right-0 bg-black/60 text-white text-xs p-1 truncate">
                          {task.model}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}
