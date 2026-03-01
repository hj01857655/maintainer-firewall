// Web Vitals 性能监控
import type { Metric } from 'web-vitals'

// 跟踪事件函数
function trackEvent(eventName: string, data: Record<string, any>) {
  console.log(`Tracking event: ${eventName}`, data)
  // 这里可以集成实际的分析服务，如 Google Analytics
  if (typeof window !== 'undefined' && (window as any).gtag) {
    ;(window as any).gtag('event', eventName, data)
  }
}

export function initPerformanceMonitoring() {
  // Web Vitals 指标收集
  if (typeof window !== 'undefined' && 'web-vitals' in window) {
    import('web-vitals').then(({ onCLS, onFCP, onLCP, onTTFB }) => {
      onCLS((metric: Metric) => {
        console.log('CLS:', metric)
        trackEvent('web_vitals_cls', { value: metric.value })
      })
      onFCP((metric: Metric) => {
        console.log('FCP:', metric)
        trackEvent('web_vitals_fcp', { value: metric.value })
      })
      onLCP((metric: Metric) => {
        console.log('LCP:', metric)
        trackEvent('web_vitals_lcp', { value: metric.value })
      })
      onTTFB((metric: Metric) => {
        console.log('TTFB:', metric)
        trackEvent('web_vitals_ttfb', { value: metric.value })
      })
    })
  }

  // 页面加载时间监控
  window.addEventListener('load', () => {
    const loadTime = performance.now()
    console.log(`页面加载时间: ${loadTime}ms`)

    // 发送到监控服务（如果有的话）
    if (typeof window !== 'undefined' && (window as any).gtag) {
      ;(window as any).gtag('event', 'page_load_time', {
        event_category: 'performance',
        event_label: 'page_load',
        value: Math.round(loadTime)
      })
    }
  })

  // 错误监控
  window.addEventListener('error', (event) => {
    console.error('JavaScript错误:', event.error)

    // 发送到错误追踪服务
    if (typeof window !== 'undefined' && (window as any).gtag) {
      ;(window as any).gtag('event', 'exception', {
        description: event.error?.message || 'JavaScript Error',
        fatal: false
      })
    }
  })

  // 未捕获的Promise错误
  window.addEventListener('unhandledrejection', (event) => {
    console.error('未捕获的Promise错误:', event.reason)

    if (typeof window !== 'undefined' && (window as any).gtag) {
      ;(window as any).gtag('event', 'exception', {
        description: (event.reason as any)?.message || 'Unhandled Promise Rejection',
        fatal: false
      })
    }
  })
}

// 用户行为跟踪
export function trackUserAction(action: string, category: string = 'engagement', label?: string) {
  if (typeof window !== 'undefined' && (window as any).gtag) {
    ;(window as any).gtag('event', action, {
      event_category: category,
      event_label: label
    })
  }

  // 也可以发送到自定义监控服务
  console.log(`用户行为: ${category} - ${action}`, label ? { label } : {})
}

// 性能指标收集
export function collectPerformanceMetrics() {
  if (typeof window.performance !== 'undefined') {
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
    const paint = performance.getEntriesByType('paint')

    const metrics = {
      // DNS查询时间
      dnsTime: navigation.domainLookupEnd - navigation.domainLookupStart,
      // TCP连接时间
      tcpTime: navigation.connectEnd - navigation.connectStart,
      // 服务器响应时间
      serverTime: navigation.responseStart - navigation.requestStart,
      // 页面加载时间
      loadTime: navigation.loadEventEnd - (navigation.activationStart || navigation.fetchStart),
      // DOM构建时间
      domTime: navigation.domContentLoadedEventEnd - (navigation.activationStart || navigation.fetchStart),
      // 首次绘制
      firstPaint: paint.find(entry => entry.name === 'first-paint')?.startTime,
      // 首次内容绘制
      firstContentfulPaint: paint.find(entry => entry.name === 'first-contentful-paint')?.startTime
    }

    console.log('性能指标:', metrics)

    // 发送到监控服务
    if (typeof window !== 'undefined' && (window as any).gtag) {
      ;(window as any).gtag('event', 'performance_metrics', {
        event_category: 'performance',
        event_label: 'page_metrics',
        custom_map: metrics
      })
    }

    return metrics
  }
}

// 内存使用情况监控
export function monitorMemoryUsage() {
  if ('memory' in performance) {
    const memory = (performance as any).memory
    const memoryInfo = {
      used: Math.round(memory.usedJSHeapSize / 1024 / 1024),
      total: Math.round(memory.totalJSHeapSize / 1024 / 1024),
      limit: Math.round(memory.jsHeapSizeLimit / 1024 / 1024)
    }

    console.log('内存使用:', memoryInfo)

    if (typeof window !== 'undefined' && (window as any).gtag) {
      ;(window as any).gtag('event', 'memory_usage', {
        event_category: 'performance',
        event_label: 'heap_size',
        value: memoryInfo.used
      })
    }

    return memoryInfo
  }
}
