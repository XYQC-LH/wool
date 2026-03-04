'use client'

import { useEffect } from 'react'
import Link from 'next/link'
import { AlertTriangle, Home, RefreshCw } from 'lucide-react'

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    if (process.env.NODE_ENV !== 'production') {
      console.error('Application error:', error)
    }
  }, [error])

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <div className="max-w-md w-full text-center">
        <div className="mb-6 flex justify-center">
          <div className="p-6 rounded-full bg-red-500/10">
            <AlertTriangle className="h-16 w-16 text-red-500" />
          </div>
        </div>

        <h1 className="mb-2 text-4xl font-bold text-foreground">500</h1>
        <h2 className="mb-4 text-2xl font-semibold text-foreground">页面出错了</h2>
        <p className="mb-8 text-muted-foreground">抱歉，页面遇到了一个错误。请稍后重试或返回首页。</p>

        {process.env.NODE_ENV !== 'production' && error.message ? (
          <div className="mb-6 p-4 bg-red-500/10 border border-red-500/20 rounded-lg">
            <p className="text-sm text-red-500 font-mono break-all">{error.message}</p>
          </div>
        ) : null}

        <div className="flex flex-col gap-3 sm:flex-row sm:justify-center">
          <button
            onClick={reset}
            className="flex items-center justify-center gap-2 px-6 py-3 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors"
          >
            <RefreshCw className="h-5 w-5" />
            重试
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
  )
}

