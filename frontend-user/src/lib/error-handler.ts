import { toast } from '@/components/ui/use-toast'

// API 错误类型
export interface ApiError {
  code: string
  message: string
  details?: unknown
  status?: number
}

// 错误码映射
const errorMessages: Record<string, string> = {
  INVALID_REQUEST: '请求参数错误',
  UNAUTHORIZED: '未授权，请重新登录',
  FORBIDDEN: '无权限访问',
  NOT_FOUND: '资源不存在',
  CONFLICT: '资源冲突',
  INTERNAL_ERROR: '服务器内部错误',
  RATE_LIMITED: '请求过于频繁，请稍后再试',
  INSUFFICIENT_FUND: '余额不足，请充值',
  QUOTA_EXCEEDED: '配额已用尽',
  CHANNEL_ERROR: '渠道错误',
  MODEL_NOT_FOUND: '模型不存在',
  NETWORK_ERROR: '网络错误，请检查网络连接',
  TIMEOUT_ERROR: '请求超时，请重试',
}

// HTTP 状态码映射
const statusMessages: Record<number, string> = {
  400: '请求参数错误',
  401: '未授权，请重新登录',
  403: '无权限访问',
  404: '资源不存在',
  429: '请求过于频繁',
  500: '服务器内部错误',
  502: '网关错误',
  503: '服务不可用',
  504: '请求超时',
}

// API 错误类
export class ApiErrorException extends Error {
  code: string
  status?: number
  details?: unknown

  constructor(error: ApiError) {
    super(error.message)
    this.name = 'ApiErrorException'
    this.code = error.code
    this.status = error.status
    this.details = error.details
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

// 解析 API 错误
export function parseApiError(error: unknown): ApiError {
  // 如果已经是 ApiError 格式
  if (isRecord(error) && typeof error.code === 'string' && typeof error.message === 'string') {
    return {
      code: error.code,
      message: error.message,
      details: error.details,
      status: typeof error.status === 'number' ? error.status : undefined,
    }
  }

  // 如果是 Axios 错误
  if (isRecord(error) && isRecord(error.response)) {
    const response = error.response
    const status = typeof response.status === 'number' ? response.status : undefined
    const data = response.data
    
    // 尝试从响应数据中提取错误信息
    if (isRecord(data) && data.error) {
      const nested = data.error
      if (isRecord(nested)) {
      return {
        code: typeof nested.code === 'string' && nested.code ? nested.code : 'UNKNOWN_ERROR',
        message:
          (typeof nested.message === 'string' && nested.message ? nested.message : undefined) ||
          (status !== undefined ? statusMessages[status] : undefined) ||
          '未知错误',
        status,
        details: nested.details,
      }
    }
    }

    // 如果响应有 message 字段
    if (isRecord(data) && typeof data.message === 'string' && data.message) {
      return {
        code: 'API_ERROR',
        message: data.message,
        status,
      }
    }

    // 使用状态码映射
    return {
      code: 'HTTP_ERROR',
      message: (status !== undefined ? statusMessages[status] : undefined) || `HTTP 错误: ${status ?? 'UNKNOWN'}`,
      status,
    }
  }

  // 如果是网络错误
  if (isRecord(error) && error.code === 'ECONNABORTED') {
    return {
      code: 'TIMEOUT_ERROR',
      message: '请求超时，请重试',
    }
  }

  if (isRecord(error) && error.code === 'ERR_NETWORK') {
    return {
      code: 'NETWORK_ERROR',
      message: '网络错误，请检查网络连接',
    }
  }

  if (typeof error === 'string' && error.trim()) {
    return {
      code: 'UNKNOWN_ERROR',
      message: error,
    }
  }

  // 默认错误
  return {
    code: 'UNKNOWN_ERROR',
    message: error instanceof Error && error.message ? error.message : '未知错误',
  }
}

// 获取用户友好的错误消息
export function getErrorMessage(error: ApiError): string {
  // 优先使用错误码映射
  if (errorMessages[error.code]) {
    return errorMessages[error.code]
  }

  // 其次使用状态码映射
  if (error.status && statusMessages[error.status]) {
    return statusMessages[error.status]
  }

  // 最后使用原始消息
  return error.message
}

// 统一错误处理函数
export function handleApiError(error: unknown, options?: {
  showToast?: boolean
  logError?: boolean
  customMessage?: string
}): ApiError {
  const {
    showToast = true,
    logError = true,
    customMessage,
  } = options || {}

  // 解析错误
  const apiError = parseApiError(error)

  // 记录错误
  if (logError && process.env.NODE_ENV !== 'production') {
    console.error('API Error:', {
      code: apiError.code,
      message: apiError.message,
      status: apiError.status,
      details: apiError.details,
      originalError: error,
    })
  }

  // 显示 Toast 通知
  if (showToast) {
    const message = customMessage || getErrorMessage(apiError)
    
    // 根据错误类型选择不同的 Toast 样式
    const isError = apiError.status && apiError.status >= 400 && apiError.status < 500
    const isServerError = apiError.status && apiError.status >= 500

    toast({
      title: isError ? '操作失败' : isServerError ? '服务器错误' : '提示',
      description: message,
      variant: isError ? 'destructive' : 'default',
    })
  }

  return apiError
}

// 异步错误处理包装器
export function withErrorHandler<TArgs extends unknown[], TResult>(
  fn: (...args: TArgs) => Promise<TResult>,
  options?: {
    showToast?: boolean
    logError?: boolean
    customMessage?: string
  }
) {
  return async (...args: TArgs): Promise<TResult> => {
    try {
      return await fn(...args)
    } catch (error) {
      handleApiError(error, options)
      throw error
    }
  }
}

// React Hook 错误处理
export function useErrorHandler() {
  const handleError = (error: unknown, options?: {
    showToast?: boolean
    logError?: boolean
    customMessage?: string
  }) => {
    return handleApiError(error, options)
  }

  return { handleError }
}
