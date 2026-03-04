/**
 * 数据导出工具
 * 支持 CSV 和 Excel 格式导出
 */

export type ExportFormat = 'csv' | 'excel'

export interface ExportOptions {
  filename?: string
  format?: ExportFormat
  headers?: string[]
  data: Record<string, unknown>[]
}

/**
 * 将数据导出为 CSV 文件
 */
export function exportToCSV(options: ExportOptions): void {
  const { filename = 'export', headers, data } = options

  if (!data || data.length === 0) {
    throw new Error('没有数据可导出')
  }

  // 获取所有列名
  const keys = headers || Object.keys(data[0])

  // 构建 CSV 内容
  const csvRows: string[] = []

  // 添加表头
  csvRows.push(keys.join(','))

  // 添加数据行
  for (const row of data) {
    const values = keys.map(key => {
      const value = row[key]
      // 处理包含逗号、引号或换行符的值
      const stringValue = String(value ?? '')
      if (stringValue.includes(',') || stringValue.includes('"') || stringValue.includes('\n')) {
        return `"${stringValue.replace(/"/g, '""')}"`
      }
      return stringValue
    })
    csvRows.push(values.join(','))
  }

  // 创建 Blob 并下载
  const csvContent = csvRows.join('\n')
  const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' })
  downloadBlob(blob, `${filename}.csv`)
}

/**
 * 将数据导出为 Excel 文件（使用 CSV 格式，Excel 可以打开）
 */
export function exportToExcel(options: ExportOptions): void {
  const { filename = 'export' } = options
  exportToCSV({ ...options, filename: `${filename}.xlsx` })
}

/**
 * 导出数据（根据格式自动选择）
 */
export function exportData(options: ExportOptions): void {
  const { format = 'csv' } = options

  switch (format) {
    case 'csv':
      exportToCSV(options)
      break
    case 'excel':
      exportToExcel(options)
      break
    default:
      throw new Error(`不支持的导出格式: ${format}`)
  }
}

/**
 * 下载 Blob 对象
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

/**
 * 格式化日期为导出友好的格式
 */
export function formatDateForExport(date: string | Date | null | undefined): string {
  if (!date) {
    return '-'
  }
  
  try {
    const d = typeof date === 'string' ? new Date(date) : date
    if (isNaN(d.getTime())) {
      return '-'
    }
    return d.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  } catch (error) {
    if (process.env.NODE_ENV !== 'production') {
      console.error('日期格式化失败:', error)
    }
    return '-'
  }
}

/**
 * 格式化金额为导出友好的格式
 */
export function formatCurrencyForExport(amount: number | string | null | undefined): string {
  if (amount === null || amount === undefined) {
    return '¥0.00'
  }
  
  // 转换为数字
  const numAmount = typeof amount === 'string' ? parseFloat(amount) : amount
  
  // 验证是否为有效数字
  if (isNaN(numAmount)) {
    return '¥0.00'
  }
  
  return `¥${numAmount.toFixed(2)}`
}

/**
 * 格式化状态为导出友好的格式
 */
export function formatStatusForExport(status: string): string {
  const statusMap: Record<string, string> = {
    active: '活跃',
    inactive: '未激活',
    pending: '待处理',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
    success: '成功',
    error: '错误',
    enabled: '已启用',
    disabled: '已禁用',
  }
  return statusMap[status] || status
}
