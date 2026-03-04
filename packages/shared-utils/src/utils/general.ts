// 通用工具函数

/**
 * 合并 Tailwind CSS 类名
 * 使用 clsx 进行条件类名合并，然后使用 tailwind-merge 解决冲突
 * 
 * @example
 * cn('px-2', 'py-1', condition && 'bg-blue-500')
 * cn({ 'px-2': true, 'py-1': false }) // => 'px-2'
 */
export function cn(...inputs: (string | undefined | null | false | Record<string, boolean>)[]): string {
  // 简单的类名合并实现（不依赖外部库）
  const classes: string[] = []
  
  for (const input of inputs) {
    if (!input) continue
    
    if (typeof input === 'string') {
      classes.push(input)
    } else if (typeof input === 'object') {
      for (const [key, value] of Object.entries(input)) {
        if (value) classes.push(key)
      }
    }
  }
  
  return classes.join(' ')
}

/**
 * 防抖函数
 * @param func 要执行的函数
 * @param wait 等待时间（毫秒）
 * @returns 防抖后的函数
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: ReturnType<typeof setTimeout> | null = null
  return (...args: Parameters<T>) => {
    if (timeout) clearTimeout(timeout)
    timeout = setTimeout(() => func(...args), wait)
  }
}

/**
 * 节流函数
 * @param func 要执行的函数
 * @param limit 时间限制（毫秒）
 * @returns 节流后的函数
 */
export function throttle<T extends (...args: unknown[]) => unknown>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle = false
  return (...args: Parameters<T>) => {
    if (!inThrottle) {
      func(...args)
      inThrottle = true
      setTimeout(() => (inThrottle = false), limit)
    }
  }
}

/**
 * 复制文本到剪贴板
 * @param text 要复制的文本
 */
export function copyToClipboard(text: string): Promise<void> {
  return navigator.clipboard.writeText(text)
}

/**
 * 生成随机 API Key
 * @returns 格式为 sk-xxxxxxxx... 的随机字符串
 */
export function generateApiKey(): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = 'sk-'
  for (let i = 0; i < 48; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

/**
 * 截断字符串
 * @param str 原字符串
 * @param maxLength 最大长度
 * @returns 截断后的字符串（带...）
 */
export function truncateString(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str
  return str.slice(0, maxLength) + '...'
}

/**
 * 获取状态对应的颜色类名
 * @param status 状态字符串
 * @returns Tailwind 颜色类名
 */
export function getStatusColor(status: string): string {
  const successStatuses = ['active', 'healthy', 'success', 'paid', 'completed', 'enabled']
  const errorStatuses = ['inactive', 'disabled', 'down', 'error', 'failed', 'banned']
  const warningStatuses = ['pending', 'processing', 'warning']
  
  if (successStatuses.includes(status)) {
    return 'text-green-500 bg-green-500/10'
  }
  if (errorStatuses.includes(status)) {
    return 'text-red-500 bg-red-500/10'
  }
  if (warningStatuses.includes(status)) {
    return 'text-yellow-500 bg-yellow-500/10'
  }
  return 'text-gray-500 bg-gray-500/10'
}

/**
 * 获取状态的中文文本
 * @param status 状态字符串
 * @returns 中文状态文本
 */
export function getStatusText(status: string): string {
  const statusMap: Record<string, string> = {
    active: '正常',
    inactive: '禁用',
    healthy: '健康',
    down: '故障',
    pending: '待处理',
    processing: '处理中',
    paid: '已支付',
    cancelled: '已取消',
    success: '成功',
    error: '失败',
    failed: '失败',
    completed: '已完成',
    enabled: '已启用',
    disabled: '已禁用',
    banned: '已封禁',
  }
  return statusMap[status] || status
}

/**
 * 睡眠函数
 * @param ms 毫秒数
 * @returns Promise
 */
export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * 深拷贝对象
 * @param obj 要拷贝的对象
 * @returns 拷贝后的新对象
 */
export function deepClone<T>(obj: T): T {
  return JSON.parse(JSON.stringify(obj))
}
