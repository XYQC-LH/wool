// 表单验证工具

/**
 * 验证规则类型
 */
export interface ValidationRule<T = unknown> {
  required?: boolean
  minLength?: number
  maxLength?: number
  pattern?: RegExp
  min?: number
  max?: number
  email?: boolean
  url?: boolean
  custom?: (value: T) => string | null
}

/**
 * 验证结果类型
 */
export interface ValidationResult {
  isValid: boolean
  error: string | null
}

/**
 * 验证器类 - 支持链式调用
 * 
 * @example
 * const validator = new Validator<string>()
 *   .required()
 *   .minLength(3)
 *   .email()
 * 
 * const result = validator.validate('test@example.com')
 */
export class Validator<T = unknown> {
  private rules: ValidationRule<T>[] = []

  /**
   * 添加自定义规则
   */
  addRule(rule: ValidationRule<T>): this {
    this.rules.push(rule)
    return this
  }

  /**
   * 必填验证
   * @param message 错误提示消息
   */
  required(message = '此字段为必填项'): this {
    return this.addRule({ required: true, custom: () => message })
  }

  /**
   * 最小长度验证
   * @param min 最小长度
   * @param message 错误提示消息（可选）
   */
  minLength(min: number, message?: string): this {
    return this.addRule({
      minLength: min,
      custom: (value) => {
        if (typeof value !== 'string') return null
        if (value.length < min) {
          return message || `最少需要 ${min} 个字符`
        }
        return null
      },
    })
  }

  /**
   * 最大长度验证
   * @param max 最大长度
   * @param message 错误提示消息（可选）
   */
  maxLength(max: number, message?: string): this {
    return this.addRule({
      maxLength: max,
      custom: (value) => {
        if (typeof value !== 'string') return null
        if (value.length > max) {
          return message || `最多允许 ${max} 个字符`
        }
        return null
      },
    })
  }

  /**
   * 正则表达式验证
   * @param regex 正则表达式
   * @param message 错误提示消息
   */
  pattern(regex: RegExp, message = '格式不正确'): this {
    return this.addRule({
      pattern: regex,
      custom: (value) => {
        if (typeof value !== 'string') return null
        if (!regex.test(value)) {
          return message
        }
        return null
      },
    })
  }

  /**
   * 邮箱格式验证
   * @param message 错误提示消息
   */
  email(message = '请输入有效的邮箱地址'): this {
    return this.addRule({
      email: true,
      custom: (value) => {
        if (typeof value !== 'string') return null
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
        if (!emailRegex.test(value)) {
          return message
        }
        return null
      },
    })
  }

  /**
   * URL 格式验证
   * @param message 错误提示消息
   */
  url(message = '请输入有效的URL'): this {
    return this.addRule({
      url: true,
      custom: (value) => {
        if (typeof value !== 'string') return null
        try {
          new URL(value)
          return null
        } catch {
          return message
        }
      },
    })
  }

  /**
   * 最小值验证（用于数字）
   * @param min 最小值
   * @param message 错误提示消息（可选）
   */
  min(min: number, message?: string): this {
    return this.addRule({
      min,
      custom: (value) => {
        if (typeof value !== 'number') return null
        if (value < min) {
          return message || `最小值为 ${min}`
        }
        return null
      },
    })
  }

  /**
   * 最大值验证（用于数字）
   * @param max 最大值
   * @param message 错误提示消息（可选）
   */
  max(max: number, message?: string): this {
    return this.addRule({
      max,
      custom: (value) => {
        if (typeof value !== 'number') return null
        if (value > max) {
          return message || `最大值为 ${max}`
        }
        return null
      },
    })
  }

  /**
   * 自定义验证函数
   * @param fn 验证函数，返回错误消息或 null
   */
  custom(fn: (value: T) => string | null): this {
    return this.addRule({ custom: fn })
  }

  /**
   * 执行验证
   * @param value 要验证的值
   * @returns 验证结果
   */
  validate(value: T): ValidationResult {
    // 检查必填
    const requiredRule = this.rules.find(r => r.required)
    if (requiredRule) {
      if (value === null || value === undefined || value === '') {
        return {
          isValid: false,
          error: requiredRule.custom?.(value) || '此字段为必填项',
        }
      }
    }

    // 如果值为空且非必填，则跳过其他验证
    if (value === null || value === undefined || value === '') {
      return { isValid: true, error: null }
    }

    // 执行所有验证规则
    for (const rule of this.rules) {
      if (rule.custom) {
        const error = rule.custom(value)
        if (error) {
          return { isValid: false, error }
        }
      }
    }

    return { isValid: true, error: null }
  }
}

/**
 * 创建验证器的便捷函数
 * @returns 新的 Validator 实例
 */
export function createValidator<T = unknown>(): Validator<T> {
  return new Validator<T>()
}

/**
 * 常用验证规则预设
 */
export const commonValidators = {
  /**
   * 用户名验证
   * - 必填
   * - 3-50 个字符
   * - 只能包含字母、数字和下划线
   */
  username: createValidator<string>()
    .required()
    .minLength(3)
    .maxLength(50)
    .pattern(/^[a-zA-Z0-9_]+$/, '只能包含字母、数字和下划线'),

  /**
   * 邮箱验证
   * - 必填
   * - 邮箱格式
   */
  email: createValidator<string>()
    .required()
    .email(),

  /**
   * 密码验证
   * - 必填
   * - 8-100 个字符
   * - 必须包含大小写字母和数字
   */
  password: createValidator<string>()
    .required()
    .minLength(8)
    .maxLength(100)
    .pattern(
      /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/,
      '密码必须包含大小写字母和数字'
    ),

  /**
   * Token 名称验证
   * - 必填
   * - 1-100 个字符
   */
  tokenName: createValidator<string>()
    .required()
    .minLength(1)
    .maxLength(100),

  /**
   * 金额验证
   * - 必填
   * - 最小 0.01
   * - 最大 100000
   */
  amount: createValidator<number>()
    .required()
    .min(0.01)
    .max(100000),

  /**
   * URL 验证
   * - 可选
   * - URL 格式
   */
  url: createValidator<string>()
    .url(),

  /**
   * 手机号验证（中国大陆）
   * - 必填
   * - 11位数字
   * - 以1开头
   */
  phone: createValidator<string>()
    .required()
    .pattern(/^1[3-9]\d{9}$/, '请输入有效的手机号码'),
}

/**
 * 表单验证结果类型
 */
export type FormValidation<T extends Record<string, unknown>> = {
  [K in keyof T]: ValidationResult
}

/**
 * 验证整个表单
 * @param data 表单数据
 * @param validators 验证器对象
 * @returns 每个字段的验证结果
 */
export function validateForm<T extends Record<string, unknown>>(
  data: T,
  validators: { [K in keyof T]?: Validator<T[K]> }
): FormValidation<T> {
  const result = {} as FormValidation<T>
  
  for (const key in validators) {
    const validator = validators[key]
    if (validator) {
      result[key] = validator.validate(data[key])
    }
  }
  
  return result
}

/**
 * 检查表单是否全部有效
 * @param validation 表单验证结果
 * @returns 是否全部有效
 */
export function isFormValid<T extends Record<string, unknown>>(
  validation: FormValidation<T>
): boolean {
  return Object.values(validation).every(v => v.isValid)
}

/**
 * 获取表单中的第一个错误消息
 * @param validation 表单验证结果
 * @returns 第一个错误消息，如果没有错误则返回 null
 */
export function getFirstError<T extends Record<string, unknown>>(
  validation: FormValidation<T>
): string | null {
  for (const key in validation) {
    if (!validation[key].isValid && validation[key].error) {
      return validation[key].error
    }
  }
  return null
}
