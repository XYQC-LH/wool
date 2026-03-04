'use client'

import { useEffect, useState } from 'react'
import {
  Save,
  RefreshCw,
  Shield,
  Server,
  Bell,
  Mail,
  Globe,
  Key,
  CheckCircle,
} from 'lucide-react'
import { useToast } from '@/components/ui/use-toast'
import { getErrorMessage, systemApi } from '@/lib/api'

export default function SettingsPage() {
  const { toast } = useToast()
  const [activeTab, setActiveTab] = useState('general')
  const [saving, setSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState(false)

  const [generalSettings, setGeneralSettings] = useState({
    site_name: 'Nexus API',
    site_url: 'https://api.nexus.com',
    admin_email: 'admin@nexus.com',
    timezone: 'Asia/Shanghai',
    language: 'zh-CN',
  })

  const [securitySettings, setSecuritySettings] = useState({
    jwt_secret: '•••••••••••••••••',
    session_timeout: 7200,
    max_login_attempts: 5,
    password_min_length: 8,
    enable_2fa: false,
  })

  const [notificationSettings, setNotificationSettings] = useState({
    email_enabled: true,
    email_smtp_host: 'smtp.gmail.com',
    email_smtp_port: 587,
    email_smtp_user: 'noreply@nexus.com',
    email_smtp_password: '•••••••••••••••••',
    webhook_enabled: false,
    webhook_url: '',
  })

  const [systemSettings, setSystemSettings] = useState({
    maintenance_mode: false,
    maintenance_message: '系统维护中，请稍后再试',
    enable_registration: true,
    default_user_balance: 0,
    max_tokens_per_request: 4096,
    rate_limit_per_minute: 60,
  })

  useEffect(() => {
    let cancelled = false

    const loadSettings = async () => {
      try {
        const [generalRes, securityRes, notificationRes, systemRes] = await Promise.all([
          systemApi.getSettings('general'),
          systemApi.getSettings('security'),
          systemApi.getSettings('notification'),
          systemApi.getSettings('system'),
        ])

        if (cancelled) return

        const errors: string[] = []

        const getData = (res: { code: number; message?: string; data?: Record<string, unknown> }, label: string) => {
          if (res.code !== 0) {
            errors.push(`${label}：${res.message || '请求失败'}`)
            return null
          }
          if (!res.data || typeof res.data !== 'object') {
            errors.push(`${label}：返回数据为空`)
            return null
          }
          return res.data
        }

        const general = getData(generalRes, '通用设置')
        if (general) {
          setGeneralSettings((prev) => ({
            ...prev,
            site_name: typeof general.site_name === 'string' ? general.site_name : prev.site_name,
            site_url: typeof general.site_url === 'string' ? general.site_url : prev.site_url,
            admin_email: typeof general.admin_email === 'string' ? general.admin_email : prev.admin_email,
            timezone: typeof general.timezone === 'string' ? general.timezone : prev.timezone,
            language: typeof general.language === 'string' ? general.language : prev.language,
          }))
        }

        const security = getData(securityRes, '安全设置')
        if (security) {
          setSecuritySettings((prev) => ({
            ...prev,
            jwt_secret: typeof security.jwt_secret === 'string' ? security.jwt_secret : prev.jwt_secret,
            session_timeout: typeof security.session_timeout === 'number' ? security.session_timeout : prev.session_timeout,
            max_login_attempts: typeof security.max_login_attempts === 'number' ? security.max_login_attempts : prev.max_login_attempts,
            password_min_length: typeof security.password_min_length === 'number' ? security.password_min_length : prev.password_min_length,
            enable_2fa: typeof security.enable_2fa === 'boolean' ? security.enable_2fa : prev.enable_2fa,
          }))
        }

        const notification = getData(notificationRes, '通知设置')
        if (notification) {
          setNotificationSettings((prev) => ({
            ...prev,
            email_enabled: typeof notification.email_enabled === 'boolean' ? notification.email_enabled : prev.email_enabled,
            email_smtp_host: typeof notification.email_smtp_host === 'string' ? notification.email_smtp_host : prev.email_smtp_host,
            email_smtp_port: typeof notification.email_smtp_port === 'number' ? notification.email_smtp_port : prev.email_smtp_port,
            email_smtp_user: typeof notification.email_smtp_user === 'string' ? notification.email_smtp_user : prev.email_smtp_user,
            email_smtp_password:
              typeof notification.email_smtp_password === 'string' ? notification.email_smtp_password : prev.email_smtp_password,
            webhook_enabled: typeof notification.webhook_enabled === 'boolean' ? notification.webhook_enabled : prev.webhook_enabled,
            webhook_url: typeof notification.webhook_url === 'string' ? notification.webhook_url : prev.webhook_url,
          }))
        }

        const system = getData(systemRes, '系统设置')
        if (system) {
          setSystemSettings((prev) => ({
            ...prev,
            maintenance_mode: typeof system.maintenance_mode === 'boolean' ? system.maintenance_mode : prev.maintenance_mode,
            maintenance_message: typeof system.maintenance_message === 'string' ? system.maintenance_message : prev.maintenance_message,
            enable_registration: typeof system.enable_registration === 'boolean' ? system.enable_registration : prev.enable_registration,
            default_user_balance: typeof system.default_user_balance === 'number' ? system.default_user_balance : prev.default_user_balance,
            max_tokens_per_request: typeof system.max_tokens_per_request === 'number' ? system.max_tokens_per_request : prev.max_tokens_per_request,
            rate_limit_per_minute: typeof system.rate_limit_per_minute === 'number' ? system.rate_limit_per_minute : prev.rate_limit_per_minute,
          }))
        }

        if (errors.length > 0) {
          toast({
            title: '部分设置加载失败',
            description: errors.join('；'),
            variant: 'destructive',
          })
        }
      } catch (error) {
        toast({
          title: '加载设置失败',
          description: getErrorMessage(error),
          variant: 'destructive',
        })
      }
    }

    loadSettings()

    return () => {
      cancelled = true
    }
  }, [toast])

  const handleSave = async (section: string) => {
    setSaving(true)
    setSaveSuccess(false)
    
    try {
      let data: Record<string, unknown> = {}
      
      switch (section) {
        case 'general':
          data = generalSettings
          break
        case 'security':
          data = securitySettings
          break
        case 'notification':
          data = notificationSettings
          break
        case 'system':
          data = systemSettings
          break
      }
      
      const res = await systemApi.saveSettings(section, data)
      if (res.code !== 0) {
        toast({
          title: '保存设置失败',
          description: res.message || '请求失败',
          variant: 'destructive',
        })
        return
      }
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 3000)
    } catch (error) {
      toast({
        title: '保存设置失败',
        description: getErrorMessage(error),
        variant: 'destructive',
      })
    } finally {
      setSaving(false)
    }
  }

  const tabs = [
    { id: 'general', label: '通用设置', icon: Globe },
    { id: 'security', label: '安全设置', icon: Shield },
    { id: 'notification', label: '通知设置', icon: Bell },
    { id: 'system', label: '系统设置', icon: Server },
  ]

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">系统设置</h1>
          <p className="text-muted-foreground">管理系统配置和参数</p>
        </div>
        {saveSuccess && (
          <div className="flex items-center gap-2 px-4 py-2 bg-green-500/10 text-green-500 rounded-lg">
            <CheckCircle className="w-4 h-4" />
            保存成功
          </div>
        )}
      </div>

      <div className="flex gap-6">
        {/* Sidebar Tabs */}
        <div className="w-64 space-y-2">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                activeTab === tab.id
                  ? 'bg-orange-500 text-white'
                  : 'hover:bg-accent text-muted-foreground'
              }`}
            >
              <tab.icon className="w-5 h-5" />
              <span>{tab.label}</span>
            </button>
          ))}
        </div>

        {/* Settings Content */}
        <div className="flex-1">
          {/* General Settings */}
          {activeTab === 'general' && (
            <div className="bg-card border border-border rounded-xl p-6 space-y-6">
              <div className="flex items-center gap-3 mb-6">
                <Globe className="w-5 h-5 text-orange-500" />
                <h2 className="text-lg font-semibold">通用设置</h2>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm font-medium mb-2">站点名称</label>
                  <input
                    type="text"
                    value={generalSettings.site_name}
                    onChange={(e) => setGeneralSettings({ ...generalSettings, site_name: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">站点 URL</label>
                  <input
                    type="url"
                    value={generalSettings.site_url}
                    onChange={(e) => setGeneralSettings({ ...generalSettings, site_url: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">管理员邮箱</label>
                  <input
                    type="email"
                    value={generalSettings.admin_email}
                    onChange={(e) => setGeneralSettings({ ...generalSettings, admin_email: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">时区</label>
                  <select
                    value={generalSettings.timezone}
                    onChange={(e) => setGeneralSettings({ ...generalSettings, timezone: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  >
                    <option value="Asia/Shanghai">Asia/Shanghai</option>
                    <option value="Asia/Tokyo">Asia/Tokyo</option>
                    <option value="America/New_York">America/New_York</option>
                    <option value="Europe/London">Europe/London</option>
                    <option value="UTC">UTC</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">语言</label>
                  <select
                    value={generalSettings.language}
                    onChange={(e) => setGeneralSettings({ ...generalSettings, language: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                  >
                    <option value="zh-CN">简体中文</option>
                    <option value="en-US">English</option>
                    <option value="ja-JP">日本語</option>
                  </select>
                </div>
              </div>

              <div className="flex justify-end pt-4 border-t border-border">
                <button
                  onClick={() => handleSave('general')}
                  disabled={saving}
                  className="flex items-center gap-2 px-6 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-500/50 text-white rounded-lg transition-colors"
                >
                  {saving ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      保存中...
                    </>
                  ) : (
                    <>
                      <Save className="w-4 h-4" />
                      保存设置
                    </>
                  )}
                </button>
              </div>
            </div>
          )}

          {/* Security Settings */}
          {activeTab === 'security' && (
            <div className="bg-card border border-border rounded-xl p-6 space-y-6">
              <div className="flex items-center gap-3 mb-6">
                <Shield className="w-5 h-5 text-orange-500" />
                <h2 className="text-lg font-semibold">安全设置</h2>
              </div>

              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-medium mb-2">JWT 密钥</label>
                  <div className="flex gap-2">
                    <input
                      type="password"
                      value={securitySettings.jwt_secret}
                      onChange={(e) => setSecuritySettings({ ...securitySettings, jwt_secret: e.target.value })}
                      className="flex-1 px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                    />
                    <button className="px-4 py-2 border border-border rounded-lg hover:bg-accent">
                      <Key className="w-4 h-4" />
                    </button>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">
                    用于生成和验证 JWT 令牌的密钥
                  </p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium mb-2">会话超时（秒）</label>
                    <input
                      type="number"
                      value={securitySettings.session_timeout}
                      onChange={(e) => setSecuritySettings({ ...securitySettings, session_timeout: parseInt(e.target.value) || 7200 })}
                      className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">最大登录尝试次数</label>
                    <input
                      type="number"
                      value={securitySettings.max_login_attempts}
                      onChange={(e) => setSecuritySettings({ ...securitySettings, max_login_attempts: parseInt(e.target.value) || 5 })}
                      className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">密码最小长度</label>
                    <input
                      type="number"
                      value={securitySettings.password_min_length}
                      onChange={(e) => setSecuritySettings({ ...securitySettings, password_min_length: parseInt(e.target.value) || 8 })}
                      className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                    />
                  </div>
                </div>

                <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
                  <div>
                    <p className="font-medium">启用双因素认证</p>
                    <p className="text-sm text-muted-foreground">为管理员账户启用 2FA</p>
                  </div>
                  <button
                    onClick={() => setSecuritySettings({ ...securitySettings, enable_2fa: !securitySettings.enable_2fa })}
                    className={`w-12 h-6 rounded-full transition-colors ${
                      securitySettings.enable_2fa ? 'bg-orange-500' : 'bg-gray-600'
                    }`}
                  >
                    <div className={`w-5 h-5 bg-white rounded-full transition-transform ${
                      securitySettings.enable_2fa ? 'translate-x-6' : 'translate-x-0.5'
                    }`} />
                  </button>
                </div>
              </div>

              <div className="flex justify-end pt-4 border-t border-border">
                <button
                  onClick={() => handleSave('security')}
                  disabled={saving}
                  className="flex items-center gap-2 px-6 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-500/50 text-white rounded-lg transition-colors"
                >
                  {saving ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      保存中...
                    </>
                  ) : (
                    <>
                      <Save className="w-4 h-4" />
                      保存设置
                    </>
                  )}
                </button>
              </div>
            </div>
          )}

          {/* Notification Settings */}
          {activeTab === 'notification' && (
            <div className="bg-card border border-border rounded-xl p-6 space-y-6">
              <div className="flex items-center gap-3 mb-6">
                <Bell className="w-5 h-5 text-orange-500" />
                <h2 className="text-lg font-semibold">通知设置</h2>
              </div>

              <div className="space-y-6">
                <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
                  <div>
                    <p className="font-medium">启用邮件通知</p>
                    <p className="text-sm text-muted-foreground">通过邮件发送系统通知</p>
                  </div>
                  <button
                    onClick={() => setNotificationSettings({ ...notificationSettings, email_enabled: !notificationSettings.email_enabled })}
                    className={`w-12 h-6 rounded-full transition-colors ${
                      notificationSettings.email_enabled ? 'bg-orange-500' : 'bg-gray-600'
                    }`}
                  >
                    <div className={`w-5 h-5 bg-white rounded-full transition-transform ${
                      notificationSettings.email_enabled ? 'translate-x-6' : 'translate-x-0.5'
                    }`} />
                  </button>
                </div>

                {notificationSettings.email_enabled && (
                  <div className="space-y-4 p-4 border border-border rounded-lg">
                    <h3 className="font-medium flex items-center gap-2">
                      <Mail className="w-4 h-4" />
                      SMTP 配置
                    </h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium mb-2">SMTP 主机</label>
                        <input
                          type="text"
                          value={notificationSettings.email_smtp_host}
                          onChange={(e) => setNotificationSettings({ ...notificationSettings, email_smtp_host: e.target.value })}
                          className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium mb-2">SMTP 端口</label>
                        <input
                          type="number"
                          value={notificationSettings.email_smtp_port}
                          onChange={(e) => setNotificationSettings({ ...notificationSettings, email_smtp_port: parseInt(e.target.value) || 587 })}
                          className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium mb-2">SMTP 用户</label>
                        <input
                          type="text"
                          value={notificationSettings.email_smtp_user}
                          onChange={(e) => setNotificationSettings({ ...notificationSettings, email_smtp_user: e.target.value })}
                          className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium mb-2">SMTP 密码</label>
                        <input
                          type="password"
                          value={notificationSettings.email_smtp_password}
                          onChange={(e) => setNotificationSettings({ ...notificationSettings, email_smtp_password: e.target.value })}
                          className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                        />
                      </div>
                    </div>
                  </div>
                )}

                <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
                  <div>
                    <p className="font-medium">启用 Webhook</p>
                    <p className="text-sm text-muted-foreground">通过 Webhook 发送事件通知</p>
                  </div>
                  <button
                    onClick={() => setNotificationSettings({ ...notificationSettings, webhook_enabled: !notificationSettings.webhook_enabled })}
                    className={`w-12 h-6 rounded-full transition-colors ${
                      notificationSettings.webhook_enabled ? 'bg-orange-500' : 'bg-gray-600'
                    }`}
                  >
                    <div className={`w-5 h-5 bg-white rounded-full transition-transform ${
                      notificationSettings.webhook_enabled ? 'translate-x-6' : 'translate-x-0.5'
                    }`} />
                  </button>
                </div>

                {notificationSettings.webhook_enabled && (
                  <div>
                    <label className="block text-sm font-medium mb-2">Webhook URL</label>
                    <input
                      type="url"
                      value={notificationSettings.webhook_url}
                      onChange={(e) => setNotificationSettings({ ...notificationSettings, webhook_url: e.target.value })}
                      className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                      placeholder="https://your-webhook-url.com"
                    />
                  </div>
                )}
              </div>

              <div className="flex justify-end pt-4 border-t border-border">
                <button
                  onClick={() => handleSave('notification')}
                  disabled={saving}
                  className="flex items-center gap-2 px-6 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-500/50 text-white rounded-lg transition-colors"
                >
                  {saving ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      保存中...
                    </>
                  ) : (
                    <>
                      <Save className="w-4 h-4" />
                      保存设置
                    </>
                  )}
                </button>
              </div>
            </div>
          )}

          {/* System Settings */}
          {activeTab === 'system' && (
            <div className="bg-card border border-border rounded-xl p-6 space-y-6">
              <div className="flex items-center gap-3 mb-6">
                <Server className="w-5 h-5 text-orange-500" />
                <h2 className="text-lg font-semibold">系统设置</h2>
              </div>

              <div className="space-y-6">
                <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
                  <div>
                    <p className="font-medium">维护模式</p>
                    <p className="text-sm text-muted-foreground">启用后，普通用户无法访问系统</p>
                  </div>
                  <button
                    onClick={() => setSystemSettings({ ...systemSettings, maintenance_mode: !systemSettings.maintenance_mode })}
                    className={`w-12 h-6 rounded-full transition-colors ${
                      systemSettings.maintenance_mode ? 'bg-orange-500' : 'bg-gray-600'
                    }`}
                  >
                    <div className={`w-5 h-5 bg-white rounded-full transition-transform ${
                      systemSettings.maintenance_mode ? 'translate-x-6' : 'translate-x-0.5'
                    }`} />
                  </button>
                </div>

                {systemSettings.maintenance_mode && (
                  <div>
                    <label className="block text-sm font-medium mb-2">维护消息</label>
                    <textarea
                      value={systemSettings.maintenance_message}
                      onChange={(e) => setSystemSettings({ ...systemSettings, maintenance_message: e.target.value })}
                      className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                      rows={3}
                    />
                  </div>
                )}

                <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
                  <div>
                    <p className="font-medium">允许用户注册</p>
                    <p className="text-sm text-muted-foreground">开放新用户注册功能</p>
                  </div>
                  <button
                    onClick={() => setSystemSettings({ ...systemSettings, enable_registration: !systemSettings.enable_registration })}
                    className={`w-12 h-6 rounded-full transition-colors ${
                      systemSettings.enable_registration ? 'bg-orange-500' : 'bg-gray-600'
                    }`}
                  >
                    <div className={`w-5 h-5 bg-white rounded-full transition-transform ${
                      systemSettings.enable_registration ? 'translate-x-6' : 'translate-x-0.5'
                    }`} />
                  </button>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium mb-2">默认用户余额</label>
                    <input
                      type="number"
                      step="0.01"
                      value={systemSettings.default_user_balance}
                      onChange={(e) => setSystemSettings({ ...systemSettings, default_user_balance: parseFloat(e.target.value) || 0 })}
                      className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">最大 Token 数/请求</label>
                    <input
                      type="number"
                      value={systemSettings.max_tokens_per_request}
                      onChange={(e) => setSystemSettings({ ...systemSettings, max_tokens_per_request: parseInt(e.target.value) || 4096 })}
                      className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">速率限制（次/分钟）</label>
                    <input
                      type="number"
                      value={systemSettings.rate_limit_per_minute}
                      onChange={(e) => setSystemSettings({ ...systemSettings, rate_limit_per_minute: parseInt(e.target.value) || 60 })}
                      className="w-full px-4 py-2 bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
                    />
                  </div>
                </div>
              </div>

              <div className="flex justify-end pt-4 border-t border-border">
                <button
                  onClick={() => handleSave('system')}
                  disabled={saving}
                  className="flex items-center gap-2 px-6 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-orange-500/50 text-white rounded-lg transition-colors"
                >
                  {saving ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      保存中...
                    </>
                  ) : (
                    <>
                      <Save className="w-4 h-4" />
                      保存设置
                    </>
                  )}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
