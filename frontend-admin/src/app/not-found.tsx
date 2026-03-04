import Link from 'next/link'
import { Home, Search } from 'lucide-react'

export default function NotFound() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <div className="max-w-md w-full text-center">
        <div className="mb-6 flex justify-center">
          <div className="p-6 rounded-full bg-orange-500/10">
            <Search className="h-16 w-16 text-orange-500" />
          </div>
        </div>
        
        <h1 className="mb-2 text-4xl font-bold text-foreground">
          404
        </h1>
        
        <h2 className="mb-4 text-2xl font-semibold text-foreground">
          页面未找到
        </h2>
        
        <p className="mb-8 text-muted-foreground">
          抱歉，您访问的页面不存在。请检查 URL 或返回首页。
        </p>
        
        <div className="flex flex-col gap-3 sm:flex-row sm:justify-center">
          <Link
            href="/dashboard"
            className="flex items-center gap-2 px-6 py-3 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
          >
            <Home className="h-5 w-5" />
            返回控制台
          </Link>
          <Link
            href="/login"
            className="flex items-center gap-2 px-6 py-3 border border-border bg-card hover:bg-accent rounded-lg transition-colors"
          >
            返回登录
          </Link>
        </div>
      </div>
    </div>
  )
}