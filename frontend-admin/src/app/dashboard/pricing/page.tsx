import Link from 'next/link'
import { Card } from '@/components/ui/card'
import { ArrowRight, CreditCard, GitBranch } from 'lucide-react'

export default function PricingHomePage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">价格系统</h1>
        <p className="text-sm text-muted-foreground mt-1">
          以模型为导向管理源头价格，并配置模型功能与基础定价。
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Link href="/dashboard/pricing/sources" className="block">
          <Card className="p-6 hover:shadow-md transition-shadow">
            <div className="flex items-start justify-between gap-4">
              <div>
                <div className="flex items-center gap-2">
                  <div className="p-2 bg-orange-100 rounded-full">
                    <GitBranch className="w-5 h-5 text-orange-600" />
                  </div>
                  <h2 className="text-lg font-semibold">源头定价（按模型）</h2>
                </div>
                <p className="text-sm text-muted-foreground mt-2">
                  先看模型列表，再进入模型查看源头状态、累计/最近调用，以及可配置的价格规则。
                </p>
              </div>
              <ArrowRight className="w-5 h-5 text-muted-foreground flex-shrink-0 mt-1" />
            </div>
          </Card>
        </Link>

        <Link href="/dashboard/pricing/models" className="block">
          <Card className="p-6 hover:shadow-md transition-shadow">
            <div className="flex items-start justify-between gap-4">
              <div>
                <div className="flex items-center gap-2">
                  <div className="p-2 bg-orange-100 rounded-full">
                    <CreditCard className="w-5 h-5 text-orange-600" />
                  </div>
                  <h2 className="text-lg font-semibold">模型定价</h2>
                </div>
                <p className="text-sm text-muted-foreground mt-2">
                  配置模型基础价格（input/output）与可用功能（operation 开关）。
                </p>
              </div>
              <ArrowRight className="w-5 h-5 text-muted-foreground flex-shrink-0 mt-1" />
            </div>
          </Card>
        </Link>
      </div>
    </div>
  )
}

