'use client'

import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useToast } from '@/components/ui/use-toast'
import { useAuthStore } from '@/store/auth'
import { imageApi, publicApi, ImageGenerationRequest, GenerationTaskResponse, Model, getGatewayApiKey } from '@/lib/api'
import { handleApiError } from '@/lib/error-handler'
import { getModelCategory } from '@/lib/model-utils'
import { Loader2, Download, RefreshCw, Image as ImageIcon, Sparkles } from 'lucide-react'
import { ReferenceImageUpload } from '@/components/reference-image-upload'
import { GatewayApiKeyManager } from '@/components/gateway-api-key-manager'

// 图片生成模型配置接口
interface ImageModelConfig {
  id: string
  name: string
  provider: string
  description: string
  resolutions: string[]
  aspectRatios: string[]
}

export default function ImagesPage() {
  const { toast } = useToast()
  const { isAuthenticated } = useAuthStore()
  const [isGenerating, setIsGenerating] = useState(false)
  const [prompt, setPrompt] = useState('')
  const [selectedModel, setSelectedModel] = useState<ImageModelConfig | null>(null)
  const [aspectRatio, setAspectRatio] = useState('1:1')
  const [resolution, setResolution] = useState('1K')
  const [referenceImagePreview, setReferenceImagePreview] = useState<string | null>(null)
  const [generatedImages, setGeneratedImages] = useState<string[]>([])
  const [currentTask, setCurrentTask] = useState<GenerationTaskResponse | null>(null)
  const [history, setHistory] = useState<GenerationTaskResponse[]>([])
  const [isLoadingHistory, setIsLoadingHistory] = useState(false)
  const [isLoadingModels, setIsLoadingModels] = useState(false)
  const [imageModels, setImageModels] = useState<ImageModelConfig[]>([])

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
        .filter((model) => getModelCategory(model) === 'image')
        .map((model) => ({
          id: model.id,
          name: model.name,
          provider: model.provider || 'Unknown',
          description: model.description || '',
          resolutions: ['1K'],
          aspectRatios: ['1:1', '16:9', '9:16'],
        }))
      
      setImageModels(models)
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
      const response = await imageApi.listTasks({ type: 'image', page: 1, page_size: 20 })
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
    if (selectedModel || imageModels.length === 0) return
    const first = imageModels[0]
    setSelectedModel(first)
    setResolution(first.resolutions[0])
    setAspectRatio(first.aspectRatios[0])
  }, [imageModels, selectedModel])

  // 生成图片
  const handleGenerate = async () => {
    if (!isAuthenticated) {
      toast({
        title: '需要登录',
        description: '请先登录以使用图片生成功能',
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
        description: '请先选择一个图片生成模型',
        variant: 'destructive',
      })
      return
    }

    setIsGenerating(true)
    setGeneratedImages([])
    setCurrentTask(null)
    try {
      const request: ImageGenerationRequest = {
        model: selectedModel.name,
        prompt: prompt.trim(),
        aspect_ratio: aspectRatio,
        resolution: resolution,
        n: 1,
        image: referenceImagePreview || undefined,
      }

      const response = await imageApi.generate(request)
      
      if (response.data && response.data.length > 0) {
        const urls = response.data.map(img => img.url || '').filter(url => url)
        setGeneratedImages(urls)
        toast({
          title: '生成成功',
          description: `成功生成 ${urls.length} 张图片`,
        })
        // 刷新历史记录
        loadHistory()
        setIsGenerating(false)
        return
      }

      if (response.task_id) {
        const taskId = response.task_id
        setCurrentTask({
          id: taskId,
          type: 'image',
          model: selectedModel.name,
          status: 'pending',
          progress: 0,
          cost: 0,
          duration: 0,
          created_at: new Date().toISOString(),
        })
        toast({
          title: '已提交任务',
          description: '图片生成任务已进入队列，请稍候…',
        })
        pollTaskStatus(taskId)
        return
      }

      toast({
        title: '生成已提交',
        description: '未获取到可展示的结果，请稍后在历史记录中查看',
      })
      setIsGenerating(false)
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : '图片生成失败，请稍后重试'
      toast({
        title: '生成失败',
        description: errorMessage,
        variant: 'destructive',
      })
      setIsGenerating(false)
    } finally {
      // isGenerating 由同步成功/轮询结束控制
    }
  }

  // 轮询任务状态（兼容异步图片生成）
  const pollTaskStatus = async (taskId: string) => {
    const maxAttempts = 60 // 最多轮询 5 分钟 (60 * 5秒)
    let attempts = 0

    const poll = async () => {
      if (attempts >= maxAttempts) {
        toast({
          title: '生成超时',
          description: '图片生成时间较长，请稍后查看历史记录',
          variant: 'destructive',
        })
        setIsGenerating(false)
        return
      }

      try {
        const task = await imageApi.getTaskStatus(taskId)
        setCurrentTask(task)

        if (task.status === 'completed' && task.result_url) {
          setGeneratedImages([task.result_url])
          setIsGenerating(false)
          toast({
            title: '生成成功',
            description: '图片生成完成',
          })
          loadHistory()
        } else if (task.status === 'failed' || task.status === 'expired') {
          toast({
            title: task.status === 'expired' ? '产物已过期' : '生成失败',
            description: task.error_message || (task.status === 'expired' ? '图片已被回收，请重新生成' : '图片生成失败'),
            variant: 'destructive',
          })
          setIsGenerating(false)
        } else {
          attempts++
          setTimeout(poll, 5000)
        }
      } catch {
        attempts++
        setTimeout(poll, 5000)
      }
    }

    poll()
  }

  // 下载图片
  const handleDownload = async (url: string, index: number) => {
    try {
      const response = await fetch(url)
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }
      const blob = await response.blob()
      const downloadUrl = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = downloadUrl
      a.download = `generated-image-${index + 1}.png`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(downloadUrl)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : '无法下载图片'
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
        <h1 className="text-3xl font-bold">图片生成</h1>
        <p className="text-muted-foreground mt-1">
          使用 AI 模型生成高质量图片
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
              ) : imageModels.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <ImageIcon className="w-12 h-12 mx-auto mb-2 opacity-50" />
                  <p>暂无可用模型</p>
                </div>
              ) : (
                imageModels.map((model) => (
                  <div
                    key={model.id}
                    className={`p-3 rounded-lg border cursor-pointer transition-colors ${
                      selectedModel?.id === model.id
                        ? 'border-primary bg-primary/5'
                        : 'border-border hover:border-primary/50'
                    }`}
                    onClick={() => {
                      setSelectedModel(model)
                      // 重置分辨率和画幅比例
                      if (!model.resolutions.includes(resolution)) {
                        setResolution(model.resolutions[0])
                      }
                      if (!model.aspectRatios.includes(aspectRatio)) {
                        setAspectRatio(model.aspectRatios[0])
                      }
                    }}
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

              {/* 分辨率 */}
              <div>
                <label className="text-sm font-medium mb-2 block">分辨率</label>
                <div className="grid grid-cols-3 gap-2">
                  {selectedModel?.resolutions.map((res) => (
                    <Button
                      key={res}
                      variant={resolution === res ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setResolution(res)}
                    >
                      {res}
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
                描述你想要生成的图片内容，越详细效果越好
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <textarea
                className="w-full min-h-[120px] p-3 rounded-lg border border-border bg-background resize-none focus:outline-none focus:ring-2 focus:ring-primary"
                placeholder="例如：一只可爱的橘猫坐在窗台上，阳光透过窗户洒在它身上，温暖的午后氛围，高清摄影风格"
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
                  模型: {selectedModel?.name || '未选择'} | 画幅: {aspectRatio} | 分辨率: {resolution}
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
                      生成图片
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
                  <ImageIcon className="mx-auto h-12 w-12 text-muted-foreground" />
                  <h3 className="mt-4 text-lg font-medium">需要登录</h3>
                  <p className="mt-2 text-sm text-muted-foreground">
                    请先登录以使用图片生成功能
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

          {/* 生成结果 */}
          {generatedImages.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <ImageIcon className="w-5 h-5" />
                  生成结果
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {generatedImages.map((url, index) => (
                    <div key={index} className="relative group">
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img
                        src={url}
                        alt={`Generated image ${index + 1}`}
                        className="w-full rounded-lg border border-border"
                      />
                      <div className="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity rounded-lg flex items-center justify-center">
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => handleDownload(url, index)}
                        >
                          <Download className="w-4 h-4 mr-2" />
                          下载
                        </Button>
                      </div>
                    </div>
                  ))}
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
                    <span>{Math.round((currentTask.progress || 0) * 100)}%</span>
                  </div>
                  <div className="w-full bg-muted rounded-full h-2">
                    <div
                      className="bg-primary h-2 rounded-full transition-all duration-300"
                      style={{ width: `${(currentTask.progress || 0) * 100}%` }}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground">
                    图片生成可能需要一些时间，请耐心等待...
                  </p>
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
                    <ImageIcon className="w-12 h-12 mx-auto mb-2 opacity-50" />
                    <p>暂无生成记录</p>
                  </div>
                ) : (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                    {history.map((task) => (
                      <div
                        key={task.id}
                        className="relative aspect-square rounded-lg border border-border overflow-hidden"
                      >
                        {task.result_url ? (
                          <>
                            {/* eslint-disable-next-line @next/next/no-img-element */}
                            <img
                              src={task.result_url}
                              alt={task.model}
                              className="w-full h-full object-cover"
                            />
                          </>
                        ) : (
                          <div className="w-full h-full flex items-center justify-center bg-muted">
                            {task.status === 'processing' ? (
                              <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
                            ) : task.status === 'failed' ? (
                              <span className="text-xs text-destructive">失败</span>
                            ) : (
                              <ImageIcon className="w-6 h-6 text-muted-foreground" />
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
