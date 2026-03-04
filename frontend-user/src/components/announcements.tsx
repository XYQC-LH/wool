'use client'

import { useCallback, useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Bell, AlertCircle, CheckCircle, Info, RefreshCw } from 'lucide-react'
import { announcementApi, type Announcement as ApiAnnouncement } from '@/lib/api'

interface AnnouncementsProps {
  limit?: number
}

export function Announcements({ limit = 3 }: AnnouncementsProps) {
  const [announcements, setAnnouncements] = useState<ApiAnnouncement[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchAnnouncements = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await announcementApi.list()
      if (response.code === 0) {
        setAnnouncements(response.data || [])
        return
      }
      setAnnouncements([])
      setError(response.message || '请求失败')
    } catch {
      setAnnouncements([])
      setError('加载公告失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchAnnouncements()
  }, [fetchAnnouncements])

  const getIcon = (type: ApiAnnouncement['type']) => {
    switch (type) {
      case 'success':
        return <CheckCircle className="w-4 h-4 text-emerald-500" />
      case 'warning':
        return <AlertCircle className="w-4 h-4 text-amber-500" />
      case 'error':
        return <AlertCircle className="w-4 h-4 text-red-500" />
      case 'info':
      default:
        return <Info className="w-4 h-4 text-blue-500" />
    }
  }

  const getDotColor = (type: ApiAnnouncement['type']) => {
    switch (type) {
      case 'success':
        return 'bg-emerald-500'
      case 'warning':
        return 'bg-amber-500'
      case 'error':
        return 'bg-red-500'
      case 'info':
      default:
        return 'bg-blue-500'
    }
  }

  const formatRelativeTime = (dateString: string) => {
    const date = new Date(dateString)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffMins < 1) return '刚刚'
    if (diffMins < 60) return `${diffMins} 分钟前`
    if (diffHours < 24) return `${diffHours} 小时前`
    return `${diffDays} 天前`
  }

  const displayAnnouncements = limit ? announcements.slice(0, limit) : announcements

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bell className="w-5 h-5" />
            系统公告
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="animate-pulse flex gap-3">
                <div className="w-2 h-2 rounded-full bg-muted mt-1.5" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-muted rounded w-3/4" />
                  <div className="h-3 bg-muted rounded w-1/2" />
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bell className="w-5 h-5" />
            系统公告
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8">
            <Bell className="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p className="text-sm font-medium text-destructive">加载公告失败</p>
            <p className="text-xs text-muted-foreground mt-1">{error}</p>
            <Button
              variant="outline"
              size="sm"
              className="mt-4"
              onClick={fetchAnnouncements}
            >
              <RefreshCw className="w-4 h-4 mr-2" />
              重试
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Bell className="w-5 h-5" />
          系统公告
        </CardTitle>
      </CardHeader>
      <CardContent>
        {displayAnnouncements.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <Bell className="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p>暂无公告</p>
          </div>
        ) : (
          <div className="space-y-4">
            {displayAnnouncements.map((announcement) => (
              <div key={announcement.id} className="flex gap-3">
                <div className={`mt-1.5 w-2 h-2 rounded-full ${getDotColor(announcement.type)} shrink-0`} />
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    {getIcon(announcement.type)}
                    <p className="text-sm font-medium">{announcement.title}</p>
                  </div>
                  <p className="text-sm text-muted-foreground">{announcement.content}</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {formatRelativeTime(announcement.created_at)}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
