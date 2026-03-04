import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

export default function PrivacyPage() {
  return (
    <div className="min-h-screen bg-background p-4 md:p-8">
      <div className="max-w-3xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">隐私政策</h1>
            <p className="text-muted-foreground mt-1">我们如何收集、使用与保护你的信息</p>
          </div>
          <Link href="/">
            <Button variant="outline">返回首页</Button>
          </Link>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>隐私保护承诺</CardTitle>
            <CardDescription>我们尊重并保护你的隐私，以下说明将帮助你了解相关实践。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 text-sm leading-6">
            <div className="space-y-2">
              <h2 className="text-base font-semibold">1. 我们收集的信息</h2>
              <ul className="list-disc pl-5 space-y-1 text-muted-foreground">
                <li>账号信息：用户名、邮箱等注册与登录所需信息。</li>
                <li>使用信息：调用日志、用量统计、计费记录等用于运营与审计的数据。</li>
                <li>设备信息：可能包含 IP、User-Agent 等用于安全风控与故障排查的信息。</li>
              </ul>
            </div>

            <div className="space-y-2">
              <h2 className="text-base font-semibold">2. 信息的使用</h2>
              <ul className="list-disc pl-5 space-y-1 text-muted-foreground">
                <li>提供与改进服务：鉴权、计费、风控、故障定位与性能优化。</li>
                <li>安全与合规：反欺诈、审计、处理滥用与满足法律法规要求。</li>
                <li>通知与沟通：系统公告、重要变更、账单与安全提醒（可在设置中管理偏好）。</li>
              </ul>
            </div>

            <div className="space-y-2">
              <h2 className="text-base font-semibold">3. 数据共享</h2>
              <p className="text-muted-foreground">
                我们不会出售你的个人信息。仅在为提供服务所必需、遵守法律或获得你授权的情况下，才会与第三方共享。
              </p>
            </div>

            <div className="space-y-2">
              <h2 className="text-base font-semibold">4. 数据安全</h2>
              <ul className="list-disc pl-5 space-y-1 text-muted-foreground">
                <li>采用访问控制、传输加密等措施保护数据安全。</li>
                <li>建议你妥善保管 API Key，避免在不可信设备或代码仓库中泄露。</li>
              </ul>
            </div>

            <div className="space-y-2">
              <h2 className="text-base font-semibold">5. 你的权利</h2>
              <ul className="list-disc pl-5 space-y-1 text-muted-foreground">
                <li>你可以在控制台中查看与更新个人资料与通知偏好。</li>
                <li>你可以创建、轮换、禁用或删除 API Key。</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

