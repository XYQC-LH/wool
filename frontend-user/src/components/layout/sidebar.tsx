'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  Key,
  FileText,
  CreditCard,
  Settings,
  HelpCircle,
  Zap,
  Cpu,
  ChevronLeft,
  ChevronRight,
  Image,
  Video,
  Menu,
  X,
} from 'lucide-react'
import { useState, useEffect } from 'react'

const menuItems = [
  {
    title: '仪表盘',
    href: '/dashboard',
    icon: LayoutDashboard,
  },
  {
    title: '图片生成',
    href: '/dashboard/images',
    icon: Image,
  },
  {
    title: '视频生成',
    href: '/dashboard/videos',
    icon: Video,
  },
  {
    title: '模型列表',
    href: '/dashboard/models',
    icon: Cpu,
  },
  {
    title: 'API Keys',
    href: '/dashboard/tokens',
    icon: Key,
  },
  {
    title: '使用日志',
    href: '/dashboard/logs',
    icon: FileText,
  },
  {
    title: '充值账单',
    href: '/dashboard/orders',
    icon: CreditCard,
  },
  {
    title: '设置',
    href: '/dashboard/settings',
    icon: Settings,
  },
  {
    title: '帮助文档',
    href: '/dashboard/docs',
    icon: HelpCircle,
  },
]

export function Sidebar() {
  const pathname = usePathname()
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [isMobile, setIsMobile] = useState(false)

  // 检测屏幕尺寸
  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768)
      if (window.innerWidth >= 768) {
        setMobileOpen(false)
      }
    }
    
    checkMobile()
    window.addEventListener('resize', checkMobile)
    return () => window.removeEventListener('resize', checkMobile)
  }, [])

  // 关闭移动端菜单
  const closeMobileMenu = () => {
    if (isMobile) {
      setMobileOpen(false)
    }
  }

  return (
    <>
      {/* 移动端汉堡菜单按钮 */}
      {isMobile && (
        <button
          onClick={() => setMobileOpen(!mobileOpen)}
          className="fixed top-4 left-4 z-50 p-2 rounded-lg bg-card border border-border shadow-lg md:hidden"
        >
          {mobileOpen ? (
            <X className="w-5 h-5" />
          ) : (
            <Menu className="w-5 h-5" />
          )}
        </button>
      )}

      {/* 移动端遮罩层 */}
      {isMobile && mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={closeMobileMenu}
        />
      )}

      <aside
        className={cn(
          'fixed left-0 top-0 z-40 h-screen border-r border-border bg-card transition-all duration-300',
          // 桌面端
          'md:translate-x-0',
          // 移动端
          isMobile
            ? mobileOpen
              ? 'translate-x-0 w-64'
              : '-translate-x-full w-64'
            : collapsed
              ? 'w-16'
              : 'w-64'
        )}
      >
      {/* Logo */}
      <div className="flex h-16 items-center justify-between border-b border-border px-4">
        <Link href="/dashboard" className="flex items-center gap-2">
          <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
            <Zap className="w-5 h-5 text-primary-foreground" />
          </div>
          {!collapsed && (
            <span className="text-lg font-bold text-gradient">Nexus API</span>
          )}
        </Link>
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="p-1 rounded-md hover:bg-accent text-muted-foreground"
        >
          {collapsed ? (
            <ChevronRight className="w-4 h-4" />
          ) : (
            <ChevronLeft className="w-4 h-4" />
          )}
        </button>
      </div>

      {/* Navigation */}
      <nav className="flex flex-col gap-1 p-2">
        {menuItems.map((item) => {
          const isActive = pathname === item.href
          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={closeMobileMenu}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors',
                isActive
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
              )}
            >
              <item.icon className="w-5 h-5 flex-shrink-0" />
              {!collapsed && <span>{item.title}</span>}
            </Link>
          )
        })}
      </nav>

      {/* Footer */}
      {!collapsed && (
        <div className="absolute bottom-4 left-4 right-4">
          <div className="rounded-lg bg-muted p-3">
            <p className="text-xs text-muted-foreground">
              需要帮助？查看我们的
              <Link href="/dashboard/docs" className="text-primary hover:underline ml-1">
                API 文档
              </Link>
            </p>
          </div>
        </div>
      )}
    </aside>
    </>
  )
}
