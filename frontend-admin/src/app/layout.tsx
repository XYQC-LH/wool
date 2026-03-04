import type { Metadata } from 'next'
import './globals.css'
import { ToastProvider } from '@/components/providers/toast-provider'

export const metadata: Metadata = {
  title: 'Nexus API - 管理后台',
  description: '高性能 API 聚合与转发平台 - 管理后台',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="zh-CN" className="dark">
      <body>
        {children}
        <ToastProvider />
      </body>
    </html>
  )
}
