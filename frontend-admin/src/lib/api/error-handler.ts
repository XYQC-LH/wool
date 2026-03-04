// lib/api/error-handler.ts
// 错误处理工具函数

import axios from 'axios'

function extractErrorMessageFromResponseData(data: unknown): string | undefined {
  if (!data) return undefined

  if (typeof data === 'string') {
    const trimmed = data.trim()
    return trimmed ? trimmed : undefined
  }

  if (typeof data !== 'object') return undefined

  const record = data as Record<string, unknown>
  const message = record.message
  if (typeof message === 'string' && message.trim()) return message

  const error = record.error
  if (typeof error === 'string' && error.trim()) return error
  if (error && typeof error === 'object') {
    const nestedMessage = (error as Record<string, unknown>).message
    if (typeof nestedMessage === 'string' && nestedMessage.trim()) return nestedMessage
  }

  return undefined
}

export function getErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    return (
      extractErrorMessageFromResponseData(error.response?.data) ||
      (typeof error.message === 'string' && error.message.trim() ? error.message : undefined) ||
      '请求失败'
    )
  }

  if (error instanceof Error) {
    return error.message && error.message.trim() ? error.message : '请求失败'
  }

  if (typeof error === 'string') {
    return error.trim() ? error : '请求失败'
  }

  return '请求失败'
}
