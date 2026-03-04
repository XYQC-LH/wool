import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

export default function TermsPage() {
  return (
    <div className="min-h-screen bg-background p-4 md:p-8">
      <div className="max-w-3xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">服务条款</h1>
            <p className="text-muted-foreground mt-1">使用 Nexus API Gateway 前请阅读</p>
          </div>
          <Link href="/">
            <Button variant="outline">返回首页</Button>
          </Link>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>概述</CardTitle>
            <CardDescription>本条款适用于 Nexus API Gateway 的网站与用户控制台服务。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 text-sm leading-6">
            <p>
              通过访问或使用本服务，即表示你已阅读、理解并同意遵守本服务条款。如你不同意本条款，请停止使用本服务。
            </p>

            <div className="space-y-2">
              <h2 className="text-base font-semibold">1. 账号与安全</h2>
              <ul className="list-disc pl-5 space-y-1 text-muted-foreground">
                <li>你应妥善保管账号、密码与 API Key，不得向他人泄露。</li>
                <li>如发现未授权访问或安全漏洞，应立即修改密钥并联系支持。</li>
              </ul>
            </div>

            <div className="space-y-2">
              <h2 className="text-base font-semibold">2. 合规使用</h2>
              <ul className="list-disc pl-5 space-y-1 text-muted-foreground">
                <li>你应遵守适用法律法规，不得利用本服务从事违法活动。</li>
                <li>你应对通过本服务发起的请求及其内容承担责任。</li>
              </ul>
            </div>

            <div className="space-y-2">
              <h2 className="text-base font-semibold">3. 计费与退款</h2>
              <ul className="list-disc pl-5 space-y-1 text-muted-foreground">
                <li>服务采用按使用量计费，具体以控制台展示的定价与账单为准。</li>
                <li>退款规则以订单状态与支付渠道规则为准，必要时将进行人工审核。</li>
              </ul>
            </div>

            <div className="space-y-2">
              <h2 className="text-base font-semibold">4. 免责声明</h2>
              <ul className="list-disc pl-5 space-y-1 text-muted-foreground">
                <li>本服务按“现状”提供，可能受上游服务波动、网络等因素影响。</li>
                <li>对于间接损失、利润损失等，除法律另有规定外不承担责任。</li>
              </ul>
            </div>

            <div className="space-y-2">
              <h2 className="text-base font-semibold">5. 联系方式</h2>
              <p className="text-muted-foreground">
                如需支持，请通过控制台帮助文档中的联系方式与我们联系。
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

