'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { Sidebar, useSidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'
import { cn } from '@/lib/utils'

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { token, isLoading } = useAuthStore()
  const router = useRouter()
  const { collapsed } = useSidebar()

  useEffect(() => {
    if (!isLoading && !token) {
      router.push('/login')
    }
  }, [token, isLoading, router])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-orange-500"></div>
      </div>
    )
  }

  if (!token) {
    return null
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <Header />
      <main className={cn(
        'pt-16 min-h-screen transition-[margin-left] duration-200 ease-out',
        // 桌面端：根据侧边栏折叠状态调整 margin-left
        'md:ml-64',
        collapsed && 'md:ml-16',
        // 移动端：无 margin
        'ml-0'
      )}>
        <div className="p-4 md:p-6">
          {children}
        </div>
      </main>
    </div>
  )
}