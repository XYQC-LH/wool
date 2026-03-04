/**
 * 响应式设计工具
 * 提供移动端适配相关的工具函数
 */

/**
 * 检测是否为移动设备
 */
export function isMobile(): boolean {
  if (typeof window === 'undefined') return false
  return window.innerWidth < 768
}

/**
 * 检测是否为平板设备
 */
export function isTablet(): boolean {
  if (typeof window === 'undefined') return false
  return window.innerWidth >= 768 && window.innerWidth < 1024
}

/**
 * 检测是否为桌面设备
 */
export function isDesktop(): boolean {
  if (typeof window === 'undefined') return false
  return window.innerWidth >= 1024
}

/**
 * 获取当前断点
 */
export function getBreakpoint(): 'sm' | 'md' | 'lg' | 'xl' {
  if (typeof window === 'undefined') return 'md'
  const width = window.innerWidth
  if (width < 640) return 'sm'
  if (width < 1024) return 'md'
  if (width < 1280) return 'lg'
  return 'xl'
}

/**
 * 响应式 Hook 类型
 */
export interface UseResponsiveReturn {
  isMobile: boolean
  isTablet: boolean
  isDesktop: boolean
  breakpoint: 'sm' | 'md' | 'lg' | 'xl'
  width: number
}

/**
 * 响应式 Hook
 * 监听窗口大小变化
 */
export function useResponsive(): UseResponsiveReturn {
  const [state, setState] = React.useState<UseResponsiveReturn>({
    isMobile: false,
    isTablet: false,
    isDesktop: true,
    breakpoint: 'md',
    width: 1024,
  })

  React.useEffect(() => {
    const handleResize = () => {
      const width = window.innerWidth
      setState({
        isMobile: width < 768,
        isTablet: width >= 768 && width < 1024,
        isDesktop: width >= 1024,
        breakpoint: getBreakpoint(),
        width,
      })
    }

    handleResize()
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  return state
}

// 导入 React 以便在文件中使用
import React from 'react'