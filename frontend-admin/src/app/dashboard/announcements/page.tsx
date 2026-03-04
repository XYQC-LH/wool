'use client'

import { useCallback, useEffect, useState } from 'react'
import { announcementApi, Announcement, getErrorMessage } from '@/lib/api'
import { useToast } from '@/components/ui/use-toast'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import {
  Plus,
  MoreVertical,
  Edit,
  Trash2,
  RefreshCw,
  Bell,
  Send,
  Archive,
  Eye,
  Info,
  AlertTriangle,
  CheckCircle,
  XCircle,
} from 'lucide-react'
import { formatDate, getStatusColor, getStatusText } from '@/lib/utils'

export default function AnnouncementsPage() {
  const { toast } = useToast()
  const [announcements, setAnnouncements] = useState<Announcement[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [statusFilter, setStatusFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [actionMenuId, setActionMenuId] = useState<number | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [editingAnnouncement, setEditingAnnouncement] = useState<Announcement | null>(null)
  const [deletingAnnouncement, setDeletingAnnouncement] = useState<Announcement | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const [formData, setFormData] = useState({
    title: '',
    content: '',
    type: 'info',
    status: 'draft',
    priority: 0,
    expires_at: '',
  })

  const loadAnnouncements = useCallback(async () => {
    setLoading(true)
    try {
      const res = await announcementApi.list({
        page,
        page_size: pageSize,
        status: statusFilter || undefined,
        type: typeFilter || undefined,
      })
      if (res.data) {
        setAnnouncements(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (error) {
      toast({
        title: '加载公告失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, statusFilter, toast, typeFilter])

  useEffect(() => {
    loadAnnouncements()
  }, [loadAnnouncements])

  const handleCreate = () => {
    setEditingAnnouncement(null)
    setFormData({
      title: '',
      content: '',
      type: 'info',
      status: 'draft',
      priority: 0,
      expires_at: '',
    })
    setShowModal(true)
  }

  const handleEdit = (announcement: Announcement) => {
    setEditingAnnouncement(announcement)
    setFormData({
      title: announcement.title,
      content: announcement.content,
      type: announcement.type,
      status: announcement.status,
      priority: announcement.priority,
      expires_at: announcement.expires_at ? new Date(announcement.expires_at).toISOString().split('T')[0] : '',
    })
    setShowModal(true)
    setActionMenuId(null)
  }

  const handleSubmit = async () => {
    try {
      const data = {
        ...formData,
        expires_at: formData.expires_at ? new Date(formData.expires_at) : undefined,
      }
      if (editingAnnouncement) {
        await announcementApi.update(editingAnnouncement.id, data)
      } else {
        await announcementApi.create(data)
      }
      setShowModal(false)
      toast({ title: '保存成功', description: editingAnnouncement ? '公告已更新' : '公告已创建' })
      loadAnnouncements()
    } catch (error) {
      toast({
        title: '保存失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
  }

  const requestDelete = (announcement: Announcement) => {
    setDeletingAnnouncement(announcement)
    setShowDeleteConfirm(true)
    setActionMenuId(null)
  }

  const confirmDelete = useCallback(async () => {
    if (!deletingAnnouncement) return
    setDeleting(true)
    try {
      await announcementApi.delete(deletingAnnouncement.id)
      toast({ title: '删除成功', description: '公告已删除' })
      setShowDeleteConfirm(false)
      setDeletingAnnouncement(null)
      loadAnnouncements()
    } catch (error) {
      toast({
        title: '删除失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setDeleting(false)
    }
  }, [deletingAnnouncement, loadAnnouncements, toast])

  const handlePublish = async (id: number) => {
    try {
      await announcementApi.publish(id)
      toast({ title: '发布成功', description: '公告已发布' })
      loadAnnouncements()
    } catch (error) {
      toast({
        title: '发布失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
    setActionMenuId(null)
  }

  const handleArchive = async (id: number) => {
    try {
      await announcementApi.archive(id)
      toast({ title: '归档成功', description: '公告已归档' })
      loadAnnouncements()
    } catch (error) {
      toast({
        title: '归档失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    }
    setActionMenuId(null)
  }

  const totalPages = Math.ceil(total / pageSize)

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'info':
        return <Info className="w-4 h-4" />
      case 'warning':
        return <AlertTriangle className="w-4 h-4" />
      case 'success':
        return <CheckCircle className="w-4 h-4" />
      case 'error':
        return <XCircle className="w-4 h-4" />
      default:
        return <Bell className="w-4 h-4" />
    }
  }

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'info':
        return 'bg-blue-500/10 text-blue-500'
      case 'warning':
        return 'bg-yellow-500/10 text-yellow-500'
      case 'success':
        return 'bg-green-500/10 text-green-500'
      case 'error':
        return 'bg-red-500/10 text-red-500'
      default:
        return 'bg-gray-500/10 text-gray-500'
    }
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">公告管理</h1>
          <p className="text-muted-foreground">管理系统公告和通知</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={loadAnnouncements}
            className="flex items-center gap-2 px-4 py-2 border border-border rounded-lg hover:bg-accent transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
            刷新
          </button>
          <button
            onClick={handleCreate}
            className="flex items-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
            添加公告
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <select
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value)
            setPage(1)
          }}
          className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
        >
          <option value="">全部状态</option>
          <option value="draft">草稿</option>
          <option value="published">已发布</option>
          <option value="archived">已归档</option>
        </select>
        <select
          value={typeFilter}
          onChange={(e) => {
            setTypeFilter(e.target.value)
            setPage(1)
          }}
          className="px-4 py-2 bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
        >
          <option value="">全部类型</option>
          <option value="info">信息</option>
          <option value="warning">警告</option>
          <option value="success">成功</option>
          <option value="error">错误</option>
        </select>
      </div>

      {/* Announcements Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {loading ? (
          <div className="col-span-full flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-orange-500"></div>
          </div>
        ) : announcements.length === 0 ? (
          <div className="col-span-full text-center py-12 text-muted-foreground">
            暂无公告数据
          </div>
        ) : (
          announcements.map((announcement) => (
            <div
              key={announcement.id}
              className="bg-card border border-border rounded-xl p-6 hover:shadow-lg transition-shadow"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg ${getTypeColor(announcement.type)}`}>
                    {getTypeIcon(announcement.type)}
                  </div>
                  <div>
                    <h3 className="font-semibold line-clamp-1">{announcement.title}</h3>
                    <p className="text-sm text-muted-foreground">
                      优先级: {announcement.priority}
                    </p>
                  </div>
                </div>
                <div className="relative">
                  <button
                    onClick={() => setActionMenuId(actionMenuId === announcement.id ? null : announcement.id)}
                    className="p-2 hover:bg-accent rounded-lg transition-colors"
                  >
                    <MoreVertical className="w-4 h-4" />
                  </button>
                  {actionMenuId === announcement.id && (
                    <div className="absolute right-0 top-full mt-1 w-40 bg-card border border-border rounded-lg shadow-lg py-1 z-10">
                      <button
                        onClick={() => handleEdit(announcement)}
                        className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                      >
                        <Edit className="w-4 h-4" />
                        编辑
                      </button>
                      {announcement.status === 'draft' && (
                        <button
                          onClick={() => handlePublish(announcement.id)}
                          className="w-full flex items-center gap-2 px-4 py-2 text-sm text-green-500 hover:bg-accent"
                        >
                          <Send className="w-4 h-4" />
                          发布
                        </button>
                      )}
                      {announcement.status === 'published' && (
                        <button
                          onClick={() => handleArchive(announcement.id)}
                          className="w-full flex items-center gap-2 px-4 py-2 text-sm text-yellow-500 hover:bg-accent"
                        >
                          <Archive className="w-4 h-4" />
                          归档
                        </button>
                      )}
                      <button
                        onClick={() => requestDelete(announcement)}
                        className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-500 hover:bg-accent"
                      >
                        <Trash2 className="w-4 h-4" />
                        删除
                      </button>
                    </div>
                  )}
                </div>
              </div>

              {/* Content Preview */}
              <div className="mb-4">
                <p className="text-sm text-muted-foreground line-clamp-3">
                  {announcement.content}
                </p>
              </div>

              {/* Status & Meta */}
              <div className="flex items-center justify-between mb-4">
                <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(announcement.status)}`}>
                  {getStatusText(announcement.status)}
                </span>
                {announcement.expires_at && (
                  <span className="text-xs text-muted-foreground">
                    过期: {formatDate(announcement.expires_at)}
                  </span>
                )}
              </div>

              {/* Footer */}
              <div className="pt-4 border-t border-border">
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>创建于 {formatDate(announcement.created_at)}</span>
                  <button
                    onClick={() => handleEdit(announcement)}
                    className="flex items-center gap-1 hover:text-accent-foreground"
                  >
                    <Eye className="w-3 h-3" />
                    查看详情
                  </button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            共 {total} 条公告，第 {page} / {totalPages} 页
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(Math.max(1, page - 1))}
              disabled={page === 1}
              className="px-3 py-1 border border-border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent"
            >
              上一页
            </button>
            <button
              onClick={() => setPage(Math.min(totalPages, page + 1))}
              disabled={page === totalPages}
              className="px-3 py-1 border border-border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent"
            >
              下一页
            </button>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={(open) => {
          setShowDeleteConfirm(open)
          if (!open) setDeletingAnnouncement(null)
        }}
        title="确认删除公告？"
        description={deletingAnnouncement ? `将删除公告「${deletingAnnouncement.title}」。此操作不可撤销。` : undefined}
        confirmText="删除"
        variant="destructive"
        loading={deleting}
        onConfirm={confirmDelete}
      />

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <h3 className="text-lg font-semibold mb-6">
              {editingAnnouncement ? '编辑公告' : '添加公告'}
            </h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">标题</label>
                <input
                  type="text"
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  placeholder="公告标题"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">内容</label>
                <textarea
                  value={formData.content}
                  onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                  className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  rows={6}
                  placeholder="公告内容"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">类型</label>
                  <select
                    value={formData.type}
                    onChange={(e) => setFormData({ ...formData, type: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  >
                    <option value="info">信息</option>
                    <option value="warning">警告</option>
                    <option value="success">成功</option>
                    <option value="error">错误</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">优先级</label>
                  <input
                    type="number"
                    value={formData.priority}
                    onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 0 })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                    min="0"
                    max="100"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">状态</label>
                  <select
                    value={formData.status}
                    onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  >
                    <option value="draft">草稿</option>
                    <option value="published">已发布</option>
                    <option value="archived">已归档</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">过期时间（可选）</label>
                  <input
                    type="date"
                    value={formData.expires_at}
                    onChange={(e) => setFormData({ ...formData, expires_at: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  />
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowModal(false)}
                className="px-4 py-2 border border-border rounded-lg hover:bg-accent"
              >
                取消
              </button>
              <button
                onClick={handleSubmit}
                disabled={!formData.title || !formData.content}
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-500/50 text-white rounded-lg"
              >
                {editingAnnouncement ? '保存' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
