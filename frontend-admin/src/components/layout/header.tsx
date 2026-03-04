'use client'

import { useAuthStore } from '@/store/auth'
import { useRouter } from 'next/navigation'
import { alertApi, Alert } from '@/lib/api'
import {
  LogOut,
  User,
  Bell,
  Settings,
} from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useSidebar } from './sidebar'
import { cn, formatRelativeTime } from '@/lib/utils'

export function Header() {
  const { logout } = useAuthStore()
  const router = useRouter()
  const [showDropdown, setShowDropdown] = useState(false)
  const [showAlerts, setShowAlerts] = useState(false)
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [alertsLoading, setAlertsLoading] = useState(false)
  const [alertsError, setAlertsError] = useState<string | null>(null)
  const { collapsed } = useSidebar()

  const loadActiveAlerts = useCallback(async () => {
    setAlertsLoading(true)
    setAlertsError(null)
    try {
      const res = await alertApi.active()
      if (res.code !== 0) {
        setAlerts([])
        setAlertsError(res.message || '请求失败')
        return
      }
      setAlerts(Array.isArray(res.data) ? res.data : [])
    } catch {
      setAlerts([])
      setAlertsError('加载活跃告警失败，请稍后重试')
    } finally {
      setAlertsLoading(false)
    }
  }, [])

  useEffect(() => {
    loadActiveAlerts()
  }, [loadActiveAlerts])

  const handleLogout = () => {
    logout()
    router.push('/login')
  }

  return (
    <header className={cn(
      'fixed top-0 right-0 z-30 h-16 border-b border-border bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/60 transition-[left] duration-200 ease-out',
      // 桌面端：根据侧边栏折叠状态调整 left 值
      'md:left-64',
      collapsed && 'md:left-16',
      // 移动端：无 left 偏移
      'left-0'
    )}>
      <div className="flex h-full items-center justify-between px-6">
        {/* Left side */}
        <div className="flex items-center gap-4">
          <h1 className="text-lg font-semibold">管理控制台</h1>
        </div>

        {/* Right side */}
        <div className="flex items-center gap-4">
          {/* Notifications */}
          <div className="relative">
            <button
              className="relative p-2 rounded-lg hover:bg-accent text-muted-foreground"
              onClick={() => {
                setShowDropdown(false)
                setShowAlerts((v) => !v)
                if (!showAlerts) {
                  loadActiveAlerts()
                }
              }}
            >
              <Bell className="w-5 h-5" />
              {alerts.length > 0 && (
                <span className="absolute top-1 right-1 min-w-2 h-2 bg-red-500 rounded-full"></span>
              )}
            </button>

            {showAlerts && (
              <div className="absolute right-0 top-full mt-2 w-96 bg-card border border-border rounded-lg shadow-lg py-2 z-50">
                <div className="px-4 py-2 flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium">活跃告警</div>
                    <div className="text-xs text-muted-foreground">未处理: {alerts.length}</div>
                  </div>
                  <button
                    onClick={loadActiveAlerts}
                    className="text-xs text-muted-foreground hover:text-foreground"
                  >
                    刷新
                  </button>
                </div>

                <div className="max-h-80 overflow-y-auto">
                  {alertsLoading ? (
                    <div className="px-4 py-6 text-sm text-muted-foreground text-center">加载中...</div>
                  ) : alertsError ? (
                    <div className="px-4 py-6 text-sm text-center">
                      <div className="text-destructive font-medium">加载失败</div>
                      <div className="text-xs text-muted-foreground mt-1">{alertsError}</div>
                    </div>
                  ) : alerts.length === 0 ? (
                    <div className="px-4 py-6 text-sm text-muted-foreground text-center">暂无活跃告警</div>
                  ) : (
                    alerts.slice(0, 5).map((a) => (
                      <div key={a.id} className="px-4 py-3 hover:bg-accent/50">
                        <div className="flex items-center justify-between gap-3">
                          <span
                            className={cn(
                              'px-2 py-0.5 rounded-full text-xs font-medium',
                              a.severity === 'critical'
                                ? 'bg-red-500/10 text-red-500'
                                : a.severity === 'error'
                                  ? 'bg-red-500/10 text-red-500'
                                  : a.severity === 'warning'
                                    ? 'bg-yellow-500/10 text-yellow-500'
                                    : 'bg-muted text-muted-foreground'
                            )}
                          >
                            {a.severity}
                          </span>
                          <span className="text-xs text-muted-foreground">{formatRelativeTime(a.created_at)}</span>
                        </div>
                        <div className="mt-1 text-sm font-medium">{a.title}</div>
                        <div className="text-xs text-muted-foreground line-clamp-2">{a.message}</div>
                      </div>
                    ))
                  )}
                </div>

                <div className="px-4 pt-2 border-t border-border">
                  <button
                    onClick={() => {
                      setShowAlerts(false)
                      router.push('/dashboard/alerts')
                    }}
                    className="w-full py-2 text-sm hover:bg-accent rounded-md"
                  >
                    查看全部
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Settings */}
          <button
            className="p-2 rounded-lg hover:bg-accent text-muted-foreground"
            onClick={() => router.push('/dashboard/settings')}
          >
            <Settings className="w-5 h-5" />
          </button>

          {/* User dropdown */}
          <div className="relative">
            <button
              onClick={() => {
                setShowAlerts(false)
                setShowDropdown(!showDropdown)
              }}
              className="flex items-center gap-2 p-2 rounded-lg hover:bg-accent"
            >
              <div className="w-8 h-8 bg-orange-500 rounded-full flex items-center justify-center">
                <User className="w-4 h-4 text-white" />
              </div>
              <span className="text-sm font-medium">管理员</span>
            </button>

            {showDropdown && (
              <div className="absolute right-0 top-full mt-2 w-48 bg-card border border-border rounded-lg shadow-lg py-1 z-50">
                <button
                  onClick={() => {
                    setShowDropdown(false)
                    router.push('/dashboard/settings')
                  }}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                >
                  <Settings className="w-4 h-4" />
                  系统设置
                </button>
                <hr className="my-1 border-border" />
                <button
                  onClick={handleLogout}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-500 hover:bg-accent"
                >
                  <LogOut className="w-4 h-4" />
                  退出登录
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  )
}
