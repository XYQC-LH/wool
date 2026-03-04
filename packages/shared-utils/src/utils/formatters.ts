// 格式化工具函数

/**
 * 格式化数字（添加 K/M 后缀）
 * @param num 数字
 * @returns 格式化后的字符串
 * @example
 * formatNumber(1500) // => '1.5K'
 * formatNumber(1500000) // => '1.5M'
 */
export function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M'
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

/**
 * 格式化货币金额
 * @param amount 金额
 * @param currency 货币代码（默认 CNY）
 * @returns 格式化后的货币字符串
 */
export function formatCurrency(amount: number, currency: string = 'CNY'): string {
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}

/**
 * 格式化日期时间
 * @param date 日期字符串或 Date 对象
 * @returns 格式化的日期字符串（YYYY-MM-DD HH:mm）
 */
export function formatDate(date: string | Date): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(date))
}

/**
 * 格式化相对时间（多久前）
 * @param date 日期字符串或 Date 对象
 * @returns 相对时间字符串
 * @example
 * formatRelativeTime(new Date()) // => '刚刚'
 * formatRelativeTime('2024-01-01') // => 'X天前'
 */
export function formatRelativeTime(date: string | Date): string {
  const now = new Date()
  const target = new Date(date)
  const diff = now.getTime() - target.getTime()
  
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  if (days > 365) {
    return `${Math.floor(days / 365)}年前`
  }
  if (days > 30) {
    return `${Math.floor(days / 30)}个月前`
  }
  if (days > 0) {
    return `${days}天前`
  }
  if (hours > 0) {
    return `${hours}小时前`
  }
  if (minutes > 0) {
    return `${minutes}分钟前`
  }
  return '刚刚'
}

/**
 * 格式化文件大小
 * @param bytes 字节数
 * @returns 格式化后的文件大小字符串
 * @example
 * formatFileSize(1024) // => '1 KB'
 * formatFileSize(1024 * 1024) // => '1 MB'
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const unitIndex = Math.floor(Math.log(bytes) / Math.log(1024))
  const size = bytes / Math.pow(1024, unitIndex)
  
  return `${size.toFixed(2)} ${units[unitIndex]}`
}

/**
 * 格式化持续时间（毫秒转可读格式）
 * @param ms 毫秒数
 * @returns 格式化后的时间字符串
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`
  }
  
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) {
    return `${seconds}s`
  }
  
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  if (minutes < 60) {
    return `${minutes}m ${remainingSeconds}s`
  }
  
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}h ${remainingMinutes}m`
}

/**
 * 格式化百分比
 * @param value 0-1 之间的小数
 * @param digits 小数位数（默认 1）
 * @returns 百分比字符串
 * @example
 * formatPercent(0.1567) // => '15.7%'
 */
export function formatPercent(value: number, digits: number = 1): string {
  return `${(value * 100).toFixed(digits)}%`
}

/**
 * 格式化手机号码（添加空格）
 * @param phone 手机号码
 * @returns 格式化后的手机号
 * @example
 * formatPhone('13812345678') // => '138 1234 5678'
 */
export function formatPhone(phone: string): string {
  if (phone.length !== 11) return phone
  return `${phone.slice(0, 3)} ${phone.slice(3, 7)} ${phone.slice(7)}`
}

/**
 * 格式化银行卡号（每4位添加空格）
 * @param cardNo 银行卡号
 * @returns 格式化后的卡号
 */
export function formatCardNo(cardNo: string): string {
  return cardNo.replace(/(\d{4})(?=\d)/g, '$1 ')
}

/**
 * 隐藏敏感信息（中间部分用*替代）
 * @param str 原字符串
 * @param visibleStart 前面显示的字符数
 * @param visibleEnd 后面显示的字符数
 * @returns 隐藏后的字符串
 * @example
 * maskSensitive('13812345678', 3, 4) // => '138****5678'
 */
export function maskSensitive(str: string, visibleStart: number = 3, visibleEnd: number = 4): string {
  if (str.length <= visibleStart + visibleEnd) return str
  
  const start = str.slice(0, visibleStart)
  const end = str.slice(-visibleEnd)
  const middleLength = str.length - visibleStart - visibleEnd
  
  return `${start}${'*'.repeat(middleLength)}${end}`
}
