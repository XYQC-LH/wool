// 表单验证 - 从共享库重新导出

// 所有验证工具都已迁移到 @nexus/shared-utils
// 从 @/lib/utils 导入以保持一致性

export {
  Validator,
  createValidator,
  commonValidators,
  validateForm,
  isFormValid,
  getFirstError,
} from '@/lib/utils'

export type {
  ValidationRule,
  ValidationResult,
  FormValidation,
} from '@/lib/utils'
