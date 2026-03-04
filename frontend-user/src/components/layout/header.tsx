'use client'

import { useAuthStore } from '@/store/auth'
import { Button } from '@/components/ui/button'
import { ThemeToggle } from '@/components/theme/theme-toggle'
import {
  Bell,
  User,
  LogOut,
  Settings,
  CreditCard,
} from 'lucide-react'
import Link from 'next/link'
import { useCallback, useEffect, useRef, useState } from 'react'
import { formatCurrency, formatRelativeTime } from '@/lib/utils'
import { publicApi, type Announcement } from '@/lib/api'

const LAST_SEEN_ANNOUNCEMENT_ID_KEY = 'nexus_last_seen_announcement_id'

export function Header() {
  const { user, isAuthenticated, logout } = useAuthStore()
  const [showDropdown, setShowDropdown] = useState(false)
  const [showNotifications, setShowNotifications] = useState(false)
  const [announcementLoading, setAnnouncementLoading] = useState(false)
  const [announcementError, setAnnouncementError] = useState<string | null>(null)
  const [announcements, setAnnouncements] = useState<Announcement[]>([])
  const [hasUnreadAnnouncements, setHasUnreadAnnouncements] = useState(false)

  const dropdownRef = useRef<HTMLDivElement>(null)
  const notificationRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      const target = event.target as Node
      const clickOutsideUserMenu = dropdownRef.current && !dropdownRef.current.contains(target)
      const clickOutsideNotifications = notificationRef.current && !notificationRef.current.contains(target)

      if (clickOutsideUserMenu) {
        setShowDropdown(false)
      }
      if (clickOutsideNotifications) {
        setShowNotifications(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const loadAnnouncements = useCallback(async () => {
    setAnnouncementLoading(true)
    setAnnouncementError(null)
    try {
      const response = await publicApi.getAnnouncements()
      if (response.code !== 0) {
        setAnnouncements([])
        setHasUnreadAnnouncements(false)
        setAnnouncementError(response.message || '请求失败')
        return
      }

      const list = response.data || []
      setAnnouncements(list)

      const maxId = list.reduce((acc, item) => Math.max(acc, Number(item.id) || 0), 0)
      const lastSeen = Number(window.localStorage.getItem(LAST_SEEN_ANNOUNCEMENT_ID_KEY) || 0)
      setHasUnreadAnnouncements(maxId > lastSeen)
    } catch {
      setAnnouncements([])
      setHasUnreadAnnouncements(false)
      setAnnouncementError('加载公告失败，请稍后重试')
    } finally {
      setAnnouncementLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!isAuthenticated) {
      setAnnouncements([])
      setHasUnreadAnnouncements(false)
      setAnnouncementError(null)
      return
    }
    loadAnnouncements()
  }, [isAuthenticated, loadAnnouncements])

  useEffect(() => {
    if (!showNotifications) return
    if (announcements.length === 0) return

    const maxId = announcements.reduce((acc, item) => Math.max(acc, Number(item.id) || 0), 0)
    window.localStorage.setItem(LAST_SEEN_ANNOUNCEMENT_ID_KEY, String(maxId))
    setHasUnreadAnnouncements(false)
  }, [showNotifications, announcements])

  // 未登录状态
  if (!isAuthenticated) {
    return (
      <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-card px-6">
        {/* 左侧 - Logo 和标题 */}
        <div>
          <h1 className="text-lg font-semibold">Nexus API</h1>
          <p className="text-sm text-muted-foreground">智能聚合 · 高可用 · 低成本</p>
        </div>

        {/* 右侧 - 登录/注册按钮 */}
        <div className="flex items-center gap-2">
          <Link href="/login">
            <Button variant="outline">登录</Button>
          </Link>
          <Link href="/register">
            <Button>注册</Button>
          </Link>
        </div>
      </header>
    )
  }

  // 已登录状态
  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-card px-6">
      {/* 左侧 - 欢迎信息 */}
      <div>
        <h1 className="text-lg font-semibold">
          欢迎回来，{user?.username || '用户'}
        </h1>
        <p className="text-sm text-muted-foreground">
          管理您的 API 密钥和使用情况
        </p>
      </div>

      {/* 右侧 - 操作区 */}
      <div className="flex items-center gap-4">
        {/* 主题切换 */}
        <ThemeToggle />
        
        {/* 余额显示 */}
        <div className="flex items-center gap-2 rounded-lg bg-muted px-3 py-2">
          <CreditCard className="w-4 h-4 text-muted-foreground" />
          <span className="text-sm font-medium">
            {formatCurrency(user?.balance || 0)}
          </span>
          <Link href="/dashboard/orders">
            <Button size="sm" variant="outline" className="h-6 text-xs">
              充值
            </Button>
          </Link>
        </div>

        {/* 通知 */}
        <div className="relative" ref={notificationRef}>
          <button
            onClick={() => setShowNotifications(!showNotifications)}
            className="relative p-2 rounded-lg hover:bg-accent"
            aria-label="系统公告"
          >
            <Bell className="w-5 h-5 text-muted-foreground" />
            {hasUnreadAnnouncements && (
              <span className="absolute top-1 right-1 w-2 h-2 bg-primary rounded-full" />
            )}
          </button>

          {showNotifications && (
            <div className="absolute right-0 top-full mt-2 w-80 rounded-lg border border-border bg-card shadow-lg animate-fade-in overflow-hidden">
              <div className="p-3 border-b border-border flex items-center justify-between">
                <div>
                  <p className="font-medium">系统公告</p>
                  <p className="text-xs text-muted-foreground">最近更新与通知</p>
                </div>
                <Button size="sm" variant="outline" className="h-8" onClick={loadAnnouncements} disabled={announcementLoading}>
                  {announcementLoading ? '刷新中' : '刷新'}
                </Button>
              </div>

              <div className="max-h-80 overflow-y-auto">
                {announcementLoading ? (
                  <div className="p-4 text-sm text-muted-foreground">加载中...</div>
                ) : announcementError ? (
                  <div className="p-4 text-sm">
                    <p className="text-destructive font-medium">加载失败</p>
                    <p className="text-xs text-muted-foreground mt-1">{announcementError}</p>
                  </div>
                ) : announcements.length === 0 ? (
                  <div className="p-4 text-sm text-muted-foreground">暂无公告</div>
                ) : (
                  <div className="divide-y divide-border">
                    {announcements.slice(0, 5).map((item) => (
                      <div key={item.id} className="p-3">
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0">
                            <p className="text-sm font-medium truncate">{item.title}</p>
                            <p className="text-sm text-muted-foreground mt-1 line-clamp-2">{item.content}</p>
                            <p className="text-xs text-muted-foreground mt-2">{formatRelativeTime(item.created_at)}</p>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="p-2 border-t border-border flex justify-end">
                <Link
                  href="/dashboard"
                  className="text-sm text-primary hover:underline underline-offset-2"
                  onClick={() => setShowNotifications(false)}
                >
                  前往仪表盘查看更多
                </Link>
              </div>
            </div>
          )}
        </div>

        {/* 用户菜单 */}
        <div className="relative" ref={dropdownRef}>
          <button
            onClick={() => setShowDropdown(!showDropdown)}
            className="flex items-center gap-2 p-2 rounded-lg hover:bg-accent"
          >
            <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
              <User className="w-4 h-4 text-primary-foreground" />
            </div>
          </button>

          {showDropdown && (
            <div className="absolute right-0 top-full mt-2 w-48 rounded-lg border border-border bg-card shadow-lg animate-fade-in">
              <div className="p-3 border-b border-border">
                <p className="font-medium">{user?.username}</p>
                <p className="text-sm text-muted-foreground">{user?.email}</p>
              </div>
              <div className="p-1">
                <Link
                  href="/dashboard/settings"
                  className="flex items-center gap-2 px-3 py-2 text-sm rounded-md hover:bg-accent"
                  onClick={() => setShowDropdown(false)}
                >
                  <Settings className="w-4 h-4" />
                  设置
                </Link>
                <button
                  onClick={() => {
                    logout()
                    setShowDropdown(false)
                  }}
                  className="flex items-center gap-2 px-3 py-2 text-sm rounded-md hover:bg-accent w-full text-left text-destructive"
                >
                  <LogOut className="w-4 h-4" />
                  退出登录
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
