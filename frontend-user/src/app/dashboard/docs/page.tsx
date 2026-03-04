'use client'

import { useState } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  BookOpen,
  Code,
  Zap,
  HelpCircle,
  Search,
  ChevronRight,
  Copy,
  ExternalLink,
  Terminal,
  Key,
  CreditCard,
  AlertCircle,
  CheckCircle,
} from 'lucide-react'
import { copyToClipboard } from '@/lib/utils'
import { useToast } from '@/components/ui/use-toast'
import { getGatewayPublicBaseUrl, getSwaggerUrl } from '@/lib/api'

export default function DocsPage() {
  const { toast } = useToast()
  const [searchQuery, setSearchQuery] = useState('')
  const [activeSection, setActiveSection] = useState('quickstart')
  const gatewayBaseUrl = getGatewayPublicBaseUrl()

  const sections = [
    { id: 'quickstart', title: '快速开始', icon: Zap },
    { id: 'api', title: 'API 文档', icon: Code },
    { id: 'authentication', title: '身份验证', icon: Key },
    { id: 'models', title: '模型列表', icon: BookOpen },
    { id: 'billing', title: '计费说明', icon: CreditCard },
    { id: 'faq', title: '常见问题', icon: HelpCircle },
  ]

  const query = searchQuery.trim().toLowerCase()
  const filteredSections = query
    ? sections.filter((section) => `${section.title} ${section.id}`.toLowerCase().includes(query))
    : sections

  const handleCopy = async (text: string) => {
    await copyToClipboard(text)
    toast({
      title: '已复制',
      description: '代码已复制到剪贴板',
    })
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* 页面标题 */}
      <div>
        <h1 className="text-3xl font-bold">帮助文档</h1>
        <p className="text-muted-foreground mt-1">
          快速了解如何使用 Nexus API
        </p>
      </div>

      {/* 搜索框 */}
      <Card>
        <CardContent className="pt-6">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="搜索文档..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-6 lg:grid-cols-4">
        {/* 侧边导航 */}
        <div className="lg:col-span-1">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">文档目录</CardTitle>
            </CardHeader>
            <CardContent className="p-2">
              <nav className="space-y-1">
                {filteredSections.length === 0 ? (
                  <div className="px-3 py-2 text-sm text-muted-foreground">
                    未找到匹配的文档目录
                  </div>
                ) : filteredSections.map((section) => {
                  const Icon = section.icon
                  return (
                    <button
                      key={section.id}
                      onClick={() => setActiveSection(section.id)}
                      className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                        activeSection === section.id
                          ? 'bg-primary text-primary-foreground'
                          : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                      }`}
                    >
                      <Icon className="w-4 h-4 flex-shrink-0" />
                      <span>{section.title}</span>
                      {activeSection === section.id && (
                        <ChevronRight className="w-4 h-4 ml-auto" />
                      )}
                    </button>
                  )
                })}
              </nav>
            </CardContent>
          </Card>
        </div>

        {/* 文档内容 */}
        <div className="lg:col-span-3">
          {activeSection === 'quickstart' && (
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Zap className="w-5 h-5" />
                    快速开始
                  </CardTitle>
                  <CardDescription>
                    在 5 分钟内开始使用 Nexus API
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-4">
                    <div className="flex gap-4">
                      <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground font-bold flex-shrink-0">
                        1
                      </div>
                      <div>
                        <h3 className="font-semibold mb-2">创建 API Key</h3>
                        <p className="text-sm text-muted-foreground mb-3">
                          前往 API Keys 页面创建您的第一个 API 密钥
                        </p>
                        <Link href="/dashboard/tokens">
                          <Button size="sm">
                            <Key className="w-4 h-4 mr-2" />
                            创建 API Key
                          </Button>
                        </Link>
                      </div>
                    </div>

                    <div className="flex gap-4">
                      <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground font-bold flex-shrink-0">
                        2
                      </div>
                      <div>
                        <h3 className="font-semibold mb-2">安装 SDK</h3>
                        <p className="text-sm text-muted-foreground mb-3">
                          使用您喜欢的编程语言安装我们的 SDK
                        </p>
                        <div className="space-y-2">
                          <div className="relative">
                            <pre className="rounded-lg bg-muted p-3 text-sm overflow-x-auto">
                              <code>npm install @nexus/sdk</code>
                            </pre>
                            <button
                              onClick={() => handleCopy('npm install @nexus/sdk')}
                              className="absolute top-2 right-2 p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
                            >
                              <Copy size={14} />
                            </button>
                          </div>
                        </div>
                      </div>
                    </div>

                    <div className="flex gap-4">
                      <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground font-bold flex-shrink-0">
                        3
                      </div>
                      <div>
                        <h3 className="font-semibold mb-2">发送第一个请求</h3>
                        <p className="text-sm text-muted-foreground mb-3">
                          使用您的 API Key 发送第一个请求
                        </p>
                        <div className="relative">
                          <pre className="rounded-lg bg-muted p-3 text-sm overflow-x-auto">
{`curl ${gatewayBaseUrl}/chat/completions \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`}
                          </pre>
                          <button
                            onClick={() =>
                              handleCopy(
                                `curl ${gatewayBaseUrl}/chat/completions \\\n  -H \"Authorization: Bearer YOUR_API_KEY\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\n    \"model\": \"gpt-4\",\n    \"messages\": [{\"role\": \"user\", \"content\": \"Hello!\"}]\n  }'`
                              )
                            }
                            className="absolute top-2 right-2 p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
                          >
                            <Copy size={14} />
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {activeSection === 'api' && (
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Code className="w-5 h-5" />
                    API 文档
                  </CardTitle>
                  <CardDescription>
                    Nexus API 完整参考文档
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div>
                    <h3 className="font-semibold mb-3">基础信息</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between p-3 bg-muted rounded-lg">
                        <span className="text-muted-foreground">API 端点</span>
                        <code className="font-mono">{gatewayBaseUrl}</code>
                      </div>
                      <div className="flex justify-between p-3 bg-muted rounded-lg">
                        <span className="text-muted-foreground">认证方式</span>
                        <code className="font-mono">Bearer Token</code>
                      </div>
                      <div className="flex justify-between p-3 bg-muted rounded-lg">
                        <span className="text-muted-foreground">请求格式</span>
                        <code className="font-mono">JSON</code>
                      </div>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-3">主要端点</h3>
                    <div className="space-y-3">
                      <div className="p-4 border rounded-lg">
                        <div className="flex items-center gap-2 mb-2">
                          <span className="px-2 py-1 bg-green-500/10 text-green-500 text-xs font-mono rounded">POST</span>
                          <code className="text-sm">/chat/completions</code>
                        </div>
                        <p className="text-sm text-muted-foreground">创建聊天完成</p>
                      </div>
                      <div className="p-4 border rounded-lg">
                        <div className="flex items-center gap-2 mb-2">
                          <span className="px-2 py-1 bg-green-500/10 text-green-500 text-xs font-mono rounded">POST</span>
                          <code className="text-sm">/images/generations</code>
                        </div>
                        <p className="text-sm text-muted-foreground">生成图片</p>
                      </div>
                      <div className="p-4 border rounded-lg">
                        <div className="flex items-center gap-2 mb-2">
                          <span className="px-2 py-1 bg-green-500/10 text-green-500 text-xs font-mono rounded">POST</span>
                          <code className="text-sm">/videos/generations</code>
                        </div>
                        <p className="text-sm text-muted-foreground">生成视频</p>
                      </div>
                    </div>
                  </div>

                  <div className="flex justify-center">
                    <Button onClick={() => window.open(getSwaggerUrl(), '_blank', 'noopener,noreferrer')}>
                      <ExternalLink className="w-4 h-4 mr-2" />
                      查看完整 API 文档
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {activeSection === 'authentication' && (
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Key className="w-5 h-5" />
                    身份验证
                  </CardTitle>
                  <CardDescription>
                    如何安全地使用 API Key 进行身份验证
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div>
                    <h3 className="font-semibold mb-3">使用 API Key</h3>
                    <p className="text-sm text-muted-foreground mb-4">
                      在每个 API 请求的 Authorization 头中包含您的 API Key
                    </p>
                    <div className="relative">
                      <pre className="rounded-lg bg-muted p-3 text-sm overflow-x-auto">
{`Authorization: Bearer sk-proj-xxxxxxxxxxxx`}
                      </pre>
                      <button
                        onClick={() => handleCopy('Authorization: Bearer sk-proj-xxxxxxxxxxxx')}
                        className="absolute top-2 right-2 p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
                      >
                        <Copy size={14} />
                      </button>
                    </div>
                  </div>

                  <div className="p-4 bg-amber-500/10 border border-amber-500/20 rounded-lg">
                    <div className="flex gap-3">
                      <AlertCircle className="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
                      <div>
                        <h4 className="font-semibold text-amber-500 mb-1">安全提示</h4>
                        <ul className="text-sm text-muted-foreground space-y-1">
                          <li>• 不要将 API Key 提交到版本控制系统</li>
                          <li>• 不要在前端代码中暴露 API Key</li>
                          <li>• 定期轮换您的 API Key</li>
                          <li>• 为不同环境使用不同的 API Key</li>
                        </ul>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {activeSection === 'models' && (
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <BookOpen className="w-5 h-5" />
                    模型列表
                  </CardTitle>
                  <CardDescription>
                    可用的 AI 模型及其定价
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div>
                    <h3 className="font-semibold mb-3">文本模型</h3>
                    <div className="space-y-3">
                      <div className="p-4 border rounded-lg">
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="font-medium">GPT-4 Turbo</h4>
                          <span className="text-sm text-muted-foreground">$0.01 / 1K tokens</span>
                        </div>
                        <p className="text-sm text-muted-foreground">最先进的语言模型，适合复杂任务</p>
                      </div>
                      <div className="p-4 border rounded-lg">
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="font-medium">GPT-3.5 Turbo</h4>
                          <span className="text-sm text-muted-foreground">$0.002 / 1K tokens</span>
                        </div>
                        <p className="text-sm text-muted-foreground">快速且经济，适合大多数应用场景</p>
                      </div>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-3">图片模型</h3>
                    <div className="space-y-3">
                      <div className="p-4 border rounded-lg">
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="font-medium">DALL-E 3</h4>
                          <span className="text-sm text-muted-foreground">$0.04 / 张</span>
                        </div>
                        <p className="text-sm text-muted-foreground">高质量图片生成，支持自然语言描述</p>
                      </div>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-3">视频模型</h3>
                    <div className="space-y-3">
                      <div className="p-4 border rounded-lg">
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="font-medium">Sora 2</h4>
                          <span className="text-sm text-muted-foreground">$0.50 / 10秒</span>
                        </div>
                        <p className="text-sm text-muted-foreground">高质量视频生成，支持多种画幅比例</p>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {activeSection === 'billing' && (
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <CreditCard className="w-5 h-5" />
                    计费说明
                  </CardTitle>
                  <CardDescription>
                    了解我们的定价和计费方式
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div>
                    <h3 className="font-semibold mb-3">计费方式</h3>
                    <div className="space-y-3">
                      <div className="flex gap-3">
                        <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5" />
                        <div>
                          <h4 className="font-medium">按使用量计费</h4>
                          <p className="text-sm text-muted-foreground">只为实际使用的资源付费，无月费</p>
                        </div>
                      </div>
                      <div className="flex gap-3">
                        <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5" />
                        <div>
                          <h4 className="font-medium">实时计费</h4>
                          <p className="text-sm text-muted-foreground">每次请求后立即扣除费用</p>
                        </div>
                      </div>
                      <div className="flex gap-3">
                        <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5" />
                        <div>
                          <h4 className="font-medium">灵活充值</h4>
                          <p className="text-sm text-muted-foreground">支持多种支付方式，随时充值</p>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-3">充值方式</h3>
                    <div className="grid gap-3 md:grid-cols-2">
                      <div className="p-4 border rounded-lg">
                        <div className="flex items-center gap-2 mb-2">
                          <CreditCard className="w-4 h-4" />
                          <h4 className="font-medium">信用卡</h4>
                        </div>
                        <p className="text-sm text-muted-foreground">支持 Visa、MasterCard 等</p>
                      </div>
                      <div className="p-4 border rounded-lg">
                        <div className="flex items-center gap-2 mb-2">
                          <Terminal className="w-4 h-4" />
                          <h4 className="font-medium">加密货币</h4>
                        </div>
                        <p className="text-sm text-muted-foreground">支持 USDT、BTC 等</p>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {activeSection === 'faq' && (
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <HelpCircle className="w-5 h-5" />
                    常见问题
                  </CardTitle>
                  <CardDescription>
                    查找常见问题的解答
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-4">
                    <div className="p-4 border rounded-lg">
                      <h4 className="font-medium mb-2">如何获取 API Key？</h4>
                      <p className="text-sm text-muted-foreground">
                        登录后前往“API Keys”页面，点击“创建 API Key”按钮即可生成新的密钥。
                      </p>
                    </div>

                    <div className="p-4 border rounded-lg">
                      <h4 className="font-medium mb-2">API Key 有有效期吗？</h4>
                      <p className="text-sm text-muted-foreground">
                        API Key 长期有效，除非您手动删除或禁用。建议定期轮换密钥以提高安全性。
                      </p>
                    </div>

                    <div className="p-4 border rounded-lg">
                      <h4 className="font-medium mb-2">如何查看我的使用量？</h4>
                      <p className="text-sm text-muted-foreground">
                        在“使用日志”页面可以查看详细的调用记录和统计数据，包括请求数、Token 消耗和费用。
                      </p>
                    </div>

                    <div className="p-4 border rounded-lg">
                      <h4 className="font-medium mb-2">支持哪些编程语言？</h4>
                      <p className="text-sm text-muted-foreground">
                        我们提供官方 SDK 支持 Python、JavaScript、Go 等主流编程语言。您也可以直接使用 REST API。
                      </p>
                    </div>

                    <div className="p-4 border rounded-lg">
                      <h4 className="font-medium mb-2">如何联系技术支持？</h4>
                      <p className="text-sm text-muted-foreground">
                        如遇到问题，请发送邮件至 support@nexus.example.com，我们会在 24 小时内回复。
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
