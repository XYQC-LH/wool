import { cn } from '@/lib/utils'

interface LoadingProps {
  size?: 'sm' | 'md' | 'lg'
  text?: string
  className?: string
}

export function Loading({ size = 'md', text, className }: LoadingProps) {
  const sizeClasses = {
    sm: 'h-4 w-4 border-2',
    md: 'h-8 w-8 border-t-2 border-b-2',
    lg: 'h-12 w-12 border-t-3 border-b-3',
  }

  return (
    <div className={cn('flex flex-col items-center justify-center', className)}>
      <div
        className={cn(
          'animate-spin rounded-full border-orange-500 border-transparent',
          sizeClasses[size]
        )}
      />
      {text && (
        <p className="mt-4 text-sm text-muted-foreground">{text}</p>
      )}
    </div>
  )
}

export function PageLoading() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Loading size="lg" text="加载中..." />
    </div>
  )
}

export function TableLoading({ colSpan = 1 }: { colSpan?: number }) {
  return (
    <tr>
      <td colSpan={colSpan} className="px-6 py-12">
        <div className="flex items-center justify-center">
          <Loading size="md" />
        </div>
      </td>
    </tr>
  )
}