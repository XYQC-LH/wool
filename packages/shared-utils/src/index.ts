// @nexus/shared-utils - 共享工具库

// ===== 通用工具函数 =====
export {
  cn,
  debounce,
  throttle,
  copyToClipboard,
  generateApiKey,
  truncateString,
  getStatusColor,
  getStatusText,
  sleep,
  deepClone,
} from './utils/general'

// ===== 格式化工具 =====
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
} from './utils/formatters'

// ===== 表单验证工具 =====
export {
  Validator,
  createValidator,
  commonValidators,
  validateForm,
  isFormValid,
  getFirstError,
} from './validation'

export type {
  ValidationRule,
  ValidationResult,
  FormValidation,
} from './validation'
