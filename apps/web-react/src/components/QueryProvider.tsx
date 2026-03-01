import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { ReactNode } from 'react'

interface QueryProviderProps {
  children: ReactNode
}

// 创建QueryClient实例，配置缓存策略
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // 缓存5分钟
      staleTime: 5 * 60 * 1000, // 5 minutes
      // 缓存时间30分钟 (新版本使用gcTime替代cacheTime)
      gcTime: 30 * 60 * 1000, // 30 minutes
      // 重试2次
      retry: 2,
      // 重试延迟
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
      // 网络恢复时重新获取
      refetchOnWindowFocus: false,
      refetchOnReconnect: true,
    },
    mutations: {
      // mutation重试1次
      retry: 1,
    },
  },
})

export function QueryProvider({ children }: QueryProviderProps) {
  // 简单的开发环境检查
  const isDev = process.env.NODE_ENV === 'development'

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      {/* 开发环境显示调试工具 */}
      {isDev && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  )
}

// 导出queryClient供其他地方使用
export { queryClient }
