'use client'

import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Zap,
  Shield,
  DollarSign,
  Code,
  MessageSquare,
  Image as ImageIcon,
  Video,
  ArrowRight,
  CheckCircle2,
  Cpu,
  Globe,
  Lock,
  BarChart3,
} from 'lucide-react'

export default function HomePage() {
  const features = [
    {
      icon: <Zap className="h-6 w-6" />,
      title: '智能调度',
      description: '自动选择最优上游，确保高可用性和低延迟',
    },
    {
      icon: <Shield className="h-6 w-6" />,
      title: '熔断保护',
      description: '自动检测故障并切换，保障服务稳定性',
    },
    {
      icon: <DollarSign className="h-6 w-6" />,
      title: '成本优化',
      description: '智能成本优先调度，降低 API 调用成本',
    },
    {
      icon: <Cpu className="h-6 w-6" />,
      title: '多模型支持',
      description: '支持 GPT-4、Claude、Gemini 等主流模型',
    },
  ]

  const services = [
    {
      icon: <MessageSquare className="h-8 w-8" />,
      title: '对话模型',
      models: ['GPT-4', 'GPT-3.5', 'Claude 3', 'Gemini Pro'],
      description: '强大的对话能力，支持多轮对话和上下文理解',
    },
    {
      icon: <ImageIcon className="h-8 w-8" />,
      title: '图像生成',
      models: ['DALL-E 3', 'Midjourney', 'Stable Diffusion'],
      description: '高质量图像生成，支持多种风格和尺寸',
    },
    {
      icon: <Video className="h-8 w-8" />,
      title: '视频生成',
      models: ['Sora', 'Runway', 'Pika'],
      description: 'AI 视频生成，创造动态视觉内容',
    },
  ]

  const advantages = [
    'OpenAI 兼容 API，无需修改现有代码',
    '99.9% 可用性保障，自动故障切换',
    '实时监控和日志，透明计费',
    '灵活的定价方案，按需付费',
    '企业级安全，数据加密传输',
    '7x24 技术支持',
  ]

  return (
    <div className="min-h-screen bg-background">
      {/* Hero Section */}
      <section className="relative overflow-hidden gradient-bg">
        <div className="container mx-auto px-4 py-20 md:py-32">
          <div className="max-w-4xl mx-auto text-center">
            <h1 className="text-4xl md:text-6xl font-bold mb-6 animate-fade-in">
              <span className="text-gradient">Nexus API Gateway</span>
            </h1>
            <p className="text-xl md:text-2xl text-muted-foreground mb-8 animate-fade-in">
              智能聚合 · 高可用 · 低成本
            </p>
            <p className="text-lg text-muted-foreground mb-12 max-w-2xl mx-auto">
              统一的 AI 模型 API 网关，提供 OpenAI 兼容接口，智能调度多上游，
              自动故障切换，降低使用成本，提升服务稳定性。
            </p>
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
              <Link href="/register">
                <Button size="lg" className="text-lg px-8">
                  免费开始使用
                  <ArrowRight className="ml-2 h-5 w-5" />
                </Button>
              </Link>
              <Link href="/login">
                <Button size="lg" variant="outline" className="text-lg px-8">
                  登录账户
                </Button>
              </Link>
            </div>
          </div>
        </div>
        {/* 装饰性背景元素 */}
        <div className="absolute top-0 left-0 w-full h-full overflow-hidden pointer-events-none">
          <div className="absolute top-20 left-10 w-72 h-72 bg-primary/10 rounded-full blur-3xl" />
          <div className="absolute bottom-20 right-10 w-96 h-96 bg-accent/10 rounded-full blur-3xl" />
        </div>
      </section>

      {/* Features Section */}
      <section className="py-20 bg-background">
        <div className="container mx-auto px-4">
          <div className="text-center mb-16">
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              核心功能
            </h2>
            <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
              强大的功能，满足您的 AI 应用需求
            </p>
          </div>
          <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-4">
            {features.map((feature, index) => (
              <Card key={index} className="card-hover border-2">
                <CardHeader>
                  <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                    <div className="text-primary">
                      {feature.icon}
                    </div>
                  </div>
                  <CardTitle className="text-xl">{feature.title}</CardTitle>
                </CardHeader>
                <CardContent>
                  <CardDescription className="text-base">
                    {feature.description}
                  </CardDescription>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* Services Section */}
      <section className="py-20 bg-muted/30">
        <div className="container mx-auto px-4">
          <div className="text-center mb-16">
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              支持的服务
            </h2>
            <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
              一站式 AI 能力，覆盖多种应用场景
            </p>
          </div>
          <div className="grid gap-8 md:grid-cols-3">
            {services.map((service, index) => (
              <Card key={index} className="card-hover">
                <CardHeader>
                  <div className="w-16 h-16 rounded-xl bg-gradient-to-br from-primary/20 to-accent/20 flex items-center justify-center mb-4">
                    <div className="text-primary">
                      {service.icon}
                    </div>
                  </div>
                  <CardTitle className="text-2xl">{service.title}</CardTitle>
                  <CardDescription className="text-base">
                    {service.description}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-muted-foreground">支持模型：</p>
                    <div className="flex flex-wrap gap-2">
                      {service.models.map((model, i) => (
                        <span
                          key={i}
                          className="px-3 py-1 bg-primary/10 text-primary text-sm rounded-full"
                        >
                          {model}
                        </span>
                      ))}
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* Advantages Section */}
      <section className="py-20 bg-background">
        <div className="container mx-auto px-4">
          <div className="grid gap-12 md:grid-cols-2 items-center">
            <div>
              <h2 className="text-3xl md:text-4xl font-bold mb-6">
                为什么选择 Nexus？
              </h2>
              <p className="text-lg text-muted-foreground mb-8">
                我们致力于为您提供最稳定、最经济、最易用的 AI API 服务
              </p>
              <div className="space-y-4">
                {advantages.map((advantage, index) => (
                  <div key={index} className="flex items-start gap-3">
                    <CheckCircle2 className="h-6 w-6 text-primary flex-shrink-0 mt-0.5" />
                    <span className="text-base">{advantage}</span>
                  </div>
                ))}
              </div>
            </div>
            <div className="grid gap-6 md:grid-cols-2">
              <Card className="glass">
                <CardHeader>
                  <Globe className="h-12 w-12 text-primary mb-4" />
                  <CardTitle className="text-2xl">全球部署</CardTitle>
                </CardHeader>
                <CardContent>
                  <CardDescription className="text-base">
                    多地域部署，就近接入，确保低延迟访问
                  </CardDescription>
                </CardContent>
              </Card>
              <Card className="glass">
                <CardHeader>
                  <Lock className="h-12 w-12 text-primary mb-4" />
                  <CardTitle className="text-2xl">安全可靠</CardTitle>
                </CardHeader>
                <CardContent>
                  <CardDescription className="text-base">
                    企业级安全标准，数据加密，隐私保护
                  </CardDescription>
                </CardContent>
              </Card>
              <Card className="glass">
                <CardHeader>
                  <BarChart3 className="h-12 w-12 text-primary mb-4" />
                  <CardTitle className="text-2xl">实时监控</CardTitle>
                </CardHeader>
                <CardContent>
                  <CardDescription className="text-base">
                    完整的监控面板，实时查看使用情况
                  </CardDescription>
                </CardContent>
              </Card>
              <Card className="glass">
                <CardHeader>
                  <Code className="h-12 w-12 text-primary mb-4" />
                  <CardTitle className="text-2xl">易于集成</CardTitle>
                </CardHeader>
                <CardContent>
                  <CardDescription className="text-base">
                    标准 REST API，提供多语言 SDK
                  </CardDescription>
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 gradient-bg">
        <div className="container mx-auto px-4">
          <div className="max-w-3xl mx-auto text-center">
            <h2 className="text-3xl md:text-4xl font-bold mb-6">
              准备好开始了吗？
            </h2>
            <p className="text-xl text-muted-foreground mb-8">
              立即注册，免费试用，体验强大的 AI API 服务
            </p>
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
              <Link href="/register">
                <Button size="lg" className="text-lg px-8">
                  免费注册
                  <ArrowRight className="ml-2 h-5 w-5" />
                </Button>
              </Link>
              <Link href="/dashboard/docs">
                <Button size="lg" variant="outline" className="text-lg px-8">
                  查看文档
                </Button>
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-8 border-t bg-background">
        <div className="container mx-auto px-4">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <p className="text-sm text-muted-foreground">
              © 2024 Nexus API Gateway. All rights reserved.
            </p>
            <div className="flex gap-6">
              <Link href="/dashboard/docs" className="text-sm text-muted-foreground hover:text-foreground">
                文档
              </Link>
              <Link href="/login" className="text-sm text-muted-foreground hover:text-foreground">
                登录
              </Link>
              <Link href="/register" className="text-sm text-muted-foreground hover:text-foreground">
                注册
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}