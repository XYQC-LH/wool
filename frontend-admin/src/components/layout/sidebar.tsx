'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  Users,
  Server,
  BarChart3,
  Cpu,
  FileText,
  CreditCard,
  Settings,
  Bell,
  Shield,
  ChevronLeft,
  ChevronRight,
  Database,
  Menu,
  X,
  GitBranch,
  Zap,
  Share2,
  BadgeDollarSign,
  SlidersHorizontal,
} from 'lucide-react'
import { useState, useEffect, createContext, useContext } from 'react'

// 创建 Context 用于共享侧边栏折叠状态
const SidebarContext = createContext<{
  collapsed: boolean
  setCollapsed: (collapsed: boolean) => void
}>({
  collapsed: false,
  setCollapsed: () => {},
})

export const useSidebar = () => useContext(SidebarContext)

const menuItems = [
  {
    title: '仪表盘',
    href: '/dashboard',
    icon: LayoutDashboard,
  },
  {
    title: '智能调度',
    href: '/dashboard/dispatch',
    icon: Zap,
  },
  {
    title: '用户管理',
    href: '/dashboard/users',
    icon: Users,
  },
  {
    title: '渠道管理',
    href: '/dashboard/channels',
    icon: Server,
  },
  {
    title: '模型管理',
    href: '/dashboard/models',
    icon: Cpu,
  },
  {
    title: '模型能力',
    href: '/dashboard/model-capabilities',
    icon: SlidersHorizontal,
  },
  {
    title: '模型源头',
    href: '/dashboard/providers',
    icon: GitBranch,
  },
  {
    title: '价格系统',
    href: '/dashboard/pricing',
    icon: BadgeDollarSign,
  },
  {
    title: '链路拓扑',
    href: '/dashboard/topology',
    icon: Share2,
  },
  {
    title: '计费规则',
    href: '/dashboard/pricing-rules',
    icon: CreditCard,
  },
  {
    title: '限流规则',
    href: '/dashboard/rate-limit-rules',
    icon: Shield,
  },
  {
    title: '资源账户',
    href: '/dashboard/resources',
    icon: Database,
  },
  {
    title: '日志查询',
    href: '/dashboard/logs',
    icon: FileText,
  },
  {
    title: '订单管理',
    href: '/dashboard/orders',
    icon: CreditCard,
  },
  {
    title: '财务报表',
    href: '/dashboard/finance',
    icon: BarChart3,
  },
  {
    title: '公告管理',
    href: '/dashboard/announcements',
    icon: Bell,
  },
  {
    title: '系统设置',
    href: '/dashboard/settings',
    icon: Settings,
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
    <SidebarContext.Provider value={{ collapsed, setCollapsed }}>
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
            'fixed left-0 top-0 z-40 h-screen border-r border-border bg-card transition-[width,transform] duration-200 ease-out',
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
            <div className="w-8 h-8 bg-orange-500 rounded-lg flex items-center justify-center">
              <Shield className="w-5 h-5 text-white" />
            </div>
            {!collapsed && (
              <span className="text-lg font-bold">Nexus Admin</span>
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
        <nav className="flex flex-col gap-1 p-2 overflow-y-auto h-[calc(100vh-4rem)]">
          {menuItems.map((item) => {
            // 仪表盘只精确匹配，其他菜单使用前缀匹配
            const isActive = item.href === '/dashboard'
              ? pathname === '/dashboard'
              : pathname === item.href || pathname.startsWith(item.href + '/')
            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={closeMobileMenu}
                className={cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors duration-150',
                  isActive
                    ? 'bg-orange-500 text-white'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                )}
              >
                <item.icon className="w-5 h-5 flex-shrink-0" />
                {!collapsed && <span>{item.title}</span>}
              </Link>
            )
          })}
        </nav>
      </aside>
      </>
    </SidebarContext.Provider>
  )
}
