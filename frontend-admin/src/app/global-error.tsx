'use client'

import { Home, AlertTriangle } from 'lucide-react'
import Link from 'next/link'

export default function GlobalError({
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  return (
    <html>
      <body>
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
          <div className="max-w-md w-full text-center">
            <div className="mb-6 flex justify-center">
              <div className="p-6 rounded-full bg-red-500/10">
                <AlertTriangle className="h-16 w-16 text-red-500" />
              </div>
            </div>
            
            <h1 className="mb-2 text-4xl font-bold text-foreground">
              系统错误
            </h1>
            
            <h2 className="mb-4 text-2xl font-semibold text-foreground">
              应用程序遇到了严重错误
            </h2>
            
            <p className="mb-8 text-muted-foreground">
              抱歉，应用程序遇到了一个严重错误。请刷新页面或联系管理员。
            </p>
            
            <div className="flex flex-col gap-3 sm:flex-row sm:justify-center">
              <button
                onClick={reset}
                className="flex items-center justify-center gap-2 px-6 py-3 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors"
              >
                刷新页面
              </button>
              <Link
                href="/"
                className="flex items-center justify-center gap-2 px-6 py-3 border border-border bg-card hover:bg-accent rounded-lg transition-colors"
              >
                <Home className="h-5 w-5" />
                返回首页
              </Link>
            </div>
          </div>
        </div>
      </body>
    </html>
  )
}
