// 项目工具函数 - 重新导出共享库 + 项目特定功能

import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

// ===== 项目特定的 cn 函数（使用 clsx + tailwind-merge）=====
/**
 * 合并 Tailwind CSS 类名
 * 使用 clsx 进行条件类名合并，然后使用 tailwind-merge 解决冲突
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// ===== 从共享库重新导出所有工具函数 =====
// 通用工具
export {
  debounce,
  throttle,
  copyToClipboard,
  generateApiKey,
  truncateString,
  getStatusColor,
  getStatusText,
  sleep,
  deepClone,
} from '@nexus/shared-utils'

// 格式化工具
export {
  formatNumber,
  formatCurrency,
  formatDate,
  formatRelativeTime,
  formatFileSize,
  formatDuration,
  formatPercent,
  formatPhone,
  formatCardNo,
  maskSensitive,
} from '@nexus/shared-utils'

// 表单验证工具
export {
  Validator,
  createValidator,
  commonValidators,
  validateForm,
  isFormValid,
  getFirstError,
} from '@nexus/shared-utils'

// 验证类型
export type {
  ValidationRule,
  ValidationResult,
  FormValidation,
} from '@nexus/shared-utils'
