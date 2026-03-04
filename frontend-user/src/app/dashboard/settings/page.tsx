'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/store/auth'
import { useToast } from '@/components/ui/use-toast'
import { getGatewayPublicBaseUrl, setGatewayPublicBaseUrl, userApi } from '@/lib/api'
import { handleApiError } from '@/lib/error-handler'
import {
  User,
  Mail,
  Lock,
  Bell,
  Shield,
  CreditCard,
  Globe,
  Save,
  RefreshCw,
  Eye,
  EyeOff,
} from 'lucide-react'

const userStatusConfig: Record<string, { label: string; className: string }> = {
  active: { label: '正常', className: 'text-green-500' },
  disabled: { label: '已禁用', className: 'text-yellow-500' },
  banned: { label: '已封禁', className: 'text-red-500' },
}

export default function SettingsPage() {
  const { user, isAuthenticated, fetchProfile } = useAuthStore()
  const { toast } = useToast()
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  
  // 个人信息表单
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  
  // 密码修改表单
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  
  // 通知设置
  const [emailNotifications, setEmailNotifications] = useState(true)
  const [usageAlerts, setUsageAlerts] = useState(true)
  const [billingAlerts, setBillingAlerts] = useState(true)
  
  // API 设置
  const [apiEndpoint, setApiEndpoint] = useState('')
  const [apiEndpointSaving, setApiEndpointSaving] = useState(false)

  useEffect(() => {
    if (user) {
      setUsername(user.username || '')
      setEmail(user.email || '')
      setEmailNotifications(user.email_notifications ?? true)
      setUsageAlerts(user.usage_alerts ?? true)
      setBillingAlerts(user.billing_alerts ?? true)
    }
  }, [user])

  useEffect(() => {
    setApiEndpoint(getGatewayPublicBaseUrl())
  }, [])

  const handleUpdateProfile = async () => {
    setLoading(true)
    try {
      const response = await userApi.updateProfile({
        username: username || undefined,
        email: email || undefined,
      })
      if (response.code !== 0) {
        toast({
          title: '更新失败',
          description: response.message || '请求失败',
          variant: 'destructive',
        })
        return
      }
      toast({
        title: '更新成功',
        description: '个人信息已更新',
      })
      fetchProfile()
    } catch (error) {
      handleApiError(error, { customMessage: '更新个人信息失败，请稍后重试' })
    } finally {
      setLoading(false)
    }
  }

  const handleChangePassword = async () => {
    if (newPassword !== confirmPassword) {
      toast({
        title: '密码不匹配',
        description: '两次输入的密码不一致',
        variant: 'destructive',
      })
      return
    }

    if (newPassword.length < 8) {
      toast({
        title: '密码太短',
        description: '密码长度至少为 8 位',
        variant: 'destructive',
      })
      return
    }

    setLoading(true)
    try {
      const response = await userApi.changePassword({
        old_password: currentPassword,
        new_password: newPassword,
      })
      if (response.code !== 0) {
        toast({
          title: '修改失败',
          description: response.message || '请求失败',
          variant: 'destructive',
        })
        return
      }
      toast({
        title: '密码修改成功',
        description: '请使用新密码重新登录',
      })
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
    } catch (error) {
      handleApiError(error, { customMessage: '密码修改失败，请稍后重试' })
    } finally {
      setLoading(false)
    }
  }

  const handleSaveNotifications = async () => {
    setLoading(true)
    try {
      const response = await userApi.updateNotifications({
        email_notifications: emailNotifications,
        usage_alerts: usageAlerts,
        billing_alerts: billingAlerts,
      })
      if (response.code !== 0) {
        toast({
          title: '保存失败',
          description: response.message || '请求失败',
          variant: 'destructive',
        })
        return
      }
      toast({
        title: '设置已保存',
        description: '通知偏好已更新',
      })
      fetchProfile()
    } catch (error) {
      handleApiError(error, { customMessage: '保存通知设置失败，请稍后重试' })
    } finally {
      setLoading(false)
    }
  }

  const handleSaveApiEndpoint = async () => {
    const value = apiEndpoint.trim()
    if (!value) {
      toast({
        title: '请输入 API 端点',
        description: '例如：https://api.example.com/v1',
        variant: 'destructive',
      })
      return
    }

    try {
      new URL(value)
    } catch {
      toast({
        title: '端点格式不正确',
        description: '请填写合法的 URL（包含 http/https）',
        variant: 'destructive',
      })
      return
    }

    setApiEndpointSaving(true)
    try {
      setGatewayPublicBaseUrl(value)
      toast({
        title: '已保存',
        description: 'API 端点已更新（用于文档与复制示例，不影响控制台本身的接口调用）',
      })
    } finally {
      setApiEndpointSaving(false)
    }
  }

  const accountStatus = user?.status
    ? (userStatusConfig[user.status] ?? { label: user.status, className: 'text-muted-foreground' })
    : { label: 'N/A', className: 'text-muted-foreground' }

  // 未登录状态
  if (!isAuthenticated) {
    return (
      <div className="space-y-6 animate-fade-in">
        <div>
          <h1 className="text-3xl font-bold">设置</h1>
          <p className="text-muted-foreground mt-1">
            管理您的账户设置和偏好
          </p>
        </div>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center py-12">
              <User className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">需要登录</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                请先登录以管理您的账户设置
              </p>
              <div className="flex justify-center gap-2 mt-4">
                <Link href="/login">
                  <Button>登录</Button>
                </Link>
                <Link href="/register">
                  <Button variant="outline">注册</Button>
                </Link>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* 页面标题 */}
      <div>
        <h1 className="text-3xl font-bold">设置</h1>
        <p className="text-muted-foreground mt-1">
          管理您的账户设置和偏好
        </p>
      </div>

      <div className="grid gap-6">
        {/* 个人信息 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <User className="w-5 h-5" />
              个人信息
            </CardTitle>
            <CardDescription>
              更新您的个人资料和联系方式
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <label className="text-sm font-medium">用户名</label>
                <Input
                  placeholder="请输入用户名"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">邮箱地址</label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                  <Input
                    placeholder="请输入邮箱"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="pl-10"
                  />
                </div>
              </div>
            </div>
            <div className="flex justify-end">
              <Button onClick={handleUpdateProfile} disabled={loading}>
                {loading ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    保存中...
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4 mr-2" />
                    保存更改
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* 安全设置 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Shield className="w-5 h-5" />
              安全设置
            </CardTitle>
            <CardDescription>
              修改您的密码和安全选项
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <div className="space-y-2">
                <label className="text-sm font-medium">当前密码</label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                  <Input
                    placeholder="请输入当前密码"
                    type={showPassword ? 'text' : 'password'}
                    value={currentPassword}
                    onChange={(e) => setCurrentPassword(e.target.value)}
                    className="pl-10 pr-10"
                  />
                </div>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">新密码</label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                  <Input
                    placeholder="请输入新密码"
                    type={showPassword ? 'text' : 'password'}
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    className="pl-10 pr-10"
                  />
                </div>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">确认新密码</label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                  <Input
                    placeholder="再次输入新密码"
                    type={showPassword ? 'text' : 'password'}
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    className="pl-10 pr-10"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  >
                    {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </button>
                </div>
              </div>
            </div>
            <div className="flex justify-end">
              <Button onClick={handleChangePassword} disabled={loading}>
                {loading ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    修改中...
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4 mr-2" />
                    修改密码
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* 通知设置 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bell className="w-5 h-5" />
              通知设置
            </CardTitle>
            <CardDescription>
              管理您的通知偏好
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-4">
              <div className="flex items-center justify-between p-4 border rounded-lg">
                <div>
                  <h4 className="font-medium">邮件通知</h4>
                  <p className="text-sm text-muted-foreground">接收重要更新和公告</p>
                </div>
                <button
                  onClick={() => setEmailNotifications(!emailNotifications)}
                  className={`relative w-12 h-6 rounded-full transition-colors ${
                    emailNotifications ? 'bg-primary' : 'bg-muted'
                  }`}
                >
                  <span
                    className={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${
                      emailNotifications ? 'left-7' : 'left-1'
                    }`}
                  />
                </button>
              </div>

              <div className="flex items-center justify-between p-4 border rounded-lg">
                <div>
                  <h4 className="font-medium">使用量提醒</h4>
                  <p className="text-sm text-muted-foreground">当使用量达到阈值时提醒</p>
                </div>
                <button
                  onClick={() => setUsageAlerts(!usageAlerts)}
                  className={`relative w-12 h-6 rounded-full transition-colors ${
                    usageAlerts ? 'bg-primary' : 'bg-muted'
                  }`}
                >
                  <span
                    className={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${
                      usageAlerts ? 'left-7' : 'left-1'
                    }`}
                  />
                </button>
              </div>

              <div className="flex items-center justify-between p-4 border rounded-lg">
                <div>
                  <h4 className="font-medium">账单提醒</h4>
                  <p className="text-sm text-muted-foreground">接收账单和支付提醒</p>
                </div>
                <button
                  onClick={() => setBillingAlerts(!billingAlerts)}
                  className={`relative w-12 h-6 rounded-full transition-colors ${
                    billingAlerts ? 'bg-primary' : 'bg-muted'
                  }`}
                >
                  <span
                    className={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${
                      billingAlerts ? 'left-7' : 'left-1'
                    }`}
                  />
                </button>
              </div>
            </div>
            <div className="flex justify-end">
              <Button onClick={handleSaveNotifications} disabled={loading}>
                {loading ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    保存中...
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4 mr-2" />
                    保存设置
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* API 设置 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Globe className="w-5 h-5" />
              API 设置
            </CardTitle>
            <CardDescription>
              配置对外暴露的 API 端点（用于文档与复制示例）
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">API 端点</label>
              <div className="relative">
                <Globe className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="https://api.nexus.example.com/v1"
                  value={apiEndpoint}
                  onChange={(e) => setApiEndpoint(e.target.value)}
                  className="pl-10"
                />
              </div>
              <p className="text-xs text-muted-foreground">
                该端点用于文档/示例复制；控制台与网关仍通过内部地址通信。
              </p>
            </div>
            <div className="flex justify-end">
              <Button onClick={handleSaveApiEndpoint} disabled={apiEndpointSaving}>
                {apiEndpointSaving ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    保存中...
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4 mr-2" />
                    保存设置
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* 账户信息 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CreditCard className="w-5 h-5" />
              账户信息
            </CardTitle>
            <CardDescription>
              查看您的账户详情
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="p-4 border rounded-lg">
                <p className="text-sm text-muted-foreground">用户 ID</p>
                <p className="font-mono text-sm mt-1">{user?.id || 'N/A'}</p>
              </div>
              <div className="p-4 border rounded-lg">
                <p className="text-sm text-muted-foreground">注册时间</p>
                <p className="text-sm mt-1">{user?.created_at ? new Date(user.created_at).toLocaleDateString('zh-CN') : 'N/A'}</p>
              </div>
              <div className="p-4 border rounded-lg">
                <p className="text-sm text-muted-foreground">账户状态</p>
                <p className={`text-sm mt-1 ${accountStatus.className}`}>{accountStatus.label}</p>
              </div>
              <div className="p-4 border rounded-lg">
                <p className="text-sm text-muted-foreground">当前余额</p>
                <p className="text-lg font-bold mt-1">${user?.balance?.toFixed(2) || '0.00'}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
