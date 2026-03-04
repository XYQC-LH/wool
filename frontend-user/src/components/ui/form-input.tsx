'use client'

import { useState } from 'react'
import { Input } from './input'
import { Label } from './label'
import { AlertCircle } from 'lucide-react'

export interface FormInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string | null
  touched?: boolean
  onTouched?: () => void
  helperText?: string
}

export function FormInput({
  label,
  error,
  touched,
  onTouched,
  helperText,
  className = '',
  onBlur,
  ...props
}: FormInputProps) {
  const [isFocused, setIsFocused] = useState(false)

  const handleBlur = (e: React.FocusEvent<HTMLInputElement>) => {
    setIsFocused(false)
    onTouched?.()
    onBlur?.(e)
  }

  const showError = touched && error

  return (
    <div className="space-y-2">
      {label && (
        <Label className={showError ? 'text-destructive' : ''}>
          {label}
          {props.required && <span className="text-destructive ml-1">*</span>}
        </Label>
      )}
      <div className="relative">
        <Input
          className={`${showError ? 'border-destructive focus:border-destructive' : ''} ${isFocused ? 'ring-2 ring-primary/20' : ''} ${className}`}
          onBlur={handleBlur}
          onFocus={() => setIsFocused(true)}
          {...props}
        />
        {showError && (
          <AlertCircle className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 text-destructive" />
        )}
      </div>
      {showError && (
        <p className="text-sm text-destructive flex items-center gap-1">
          <AlertCircle className="h-3 w-3" />
          {error}
        </p>
      )}
      {helperText && !showError && (
        <p className="text-sm text-muted-foreground">{helperText}</p>
      )}
    </div>
  )
}
