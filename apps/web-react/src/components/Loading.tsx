import { cn } from '../utils/cn'

interface LoadingProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export function Loading({ size = 'md', className }: LoadingProps) {
  const sizeClasses = {
    sm: 'h-4 w-4',
    md: 'h-8 w-8',
    lg: 'h-12 w-12',
  }

  return (
    <div className="flex items-center justify-center py-12">
      <div className={cn('flex items-center gap-3 text-slate-600 dark:text-slate-300', className)}>
        <div
          className={cn(
            'animate-spin rounded-full border-2 border-current border-t-transparent',
            sizeClasses[size]
          )}
        />
        <span>加载中...</span>
      </div>
    </div>
  )
}
