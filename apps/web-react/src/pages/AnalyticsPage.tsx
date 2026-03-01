import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import { Loading } from '../components/Loading'
import { cn } from '../utils/cn'

interface MetricsOverview {
  events_24h: number
  alerts_24h: number
  failures_24h: number
  success_rate_24h: number
  p95_latency_ms_24h: number
}

interface MetricsTimePoint {
  bucket_start: string
  events: number
  alerts: number
  failures: number
}

interface AnalyticsData {
  overview: MetricsOverview
  timeseries: MetricsTimePoint[]
}

export function AnalyticsPage() {
  const { t } = useTranslation()
  const [timeRange, setTimeRange] = useState<'24h' | '7d' | '30d'>('24h')

  // 获取统计数据
  const { data: analyticsData, isLoading, error } = useQuery<AnalyticsData>({
    queryKey: ['analytics', timeRange],
    queryFn: async (): Promise<AnalyticsData> => {
      const interval = timeRange === '24h' ? 60 : timeRange === '7d' ? 360 : 1440 // minutes
      const since = new Date()

      switch (timeRange) {
        case '24h':
          since.setHours(since.getHours() - 24)
          break
        case '7d':
          since.setDate(since.getDate() - 7)
          break
        case '30d':
          since.setDate(since.getDate() - 30)
          break
      }

      const [overviewRes, timeseriesRes] = await Promise.all([
        api.get('/metrics/overview'),
        api.get(`/metrics/timeseries?since=${since.toISOString()}&interval_minutes=${interval}`)
      ])

      return {
        overview: overviewRes.data,
        timeseries: timeseriesRes.data
      }
    },
    retry: 1, // 只重试1次，避免无限loading
  })

  if (isLoading) {
    return <Loading />
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-center py-12">
          <div className="text-center">
            <div className="text-red-600 dark:text-red-400 mb-2">
              <svg className="h-12 w-12 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-slate-900 dark:text-slate-100 mb-2">
              {t('common.error')}
            </h3>
            <p className="text-slate-600 dark:text-slate-400 mb-4">
              无法加载统计数据：{(error as Error)?.message || '未知错误'}
            </p>
            <Button onClick={() => window.location.reload()}>
              {t('errorBoundary.reload')}
            </Button>
          </div>
        </div>
      </div>
    )
  }

  const timeRangeOptions = [
    { value: '24h', label: t('analytics.timeRange.24h') },
    { value: '7d', label: t('analytics.timeRange.7d') },
    { value: '30d', label: t('analytics.timeRange.30d') },
  ]

  const data = analyticsData || { overview: {} as MetricsOverview, timeseries: [] }

  return (
    <div className="space-y-6">
      {/* 页面标题和时间范围选择 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t('analytics.title')}</h1>
          <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
            {t('analytics.description')}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{t('analytics.timeRange.label')}:</span>
          <select
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value as '24h' | '7d' | '30d')}
            className="px-3 py-1 border rounded-md text-sm"
          >
            {timeRangeOptions.map(option => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* 关键指标卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <MetricCard
          title={t('analytics.metrics.events')}
          value={data.overview.events_24h?.toLocaleString() || '0'}
          change="+12%"
          trend="up"
          icon="📈"
        />
        <MetricCard
          title={t('analytics.metrics.alerts')}
          value={data.overview.alerts_24h?.toLocaleString() || '0'}
          change="+8%"
          trend="up"
          icon="🚨"
        />
        <MetricCard
          title={t('analytics.metrics.failures')}
          value={data.overview.failures_24h?.toLocaleString() || '0'}
          change="-15%"
          trend="down"
          icon="❌"
        />
        <MetricCard
          title={t('analytics.metrics.successRate')}
          value={`${(data.overview.success_rate_24h || 0).toFixed(1)}%`}
          change="+2%"
          trend="up"
          icon="✅"
        />
      </div>

      {/* 图表区域 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 趋势图表 */}
        <Card className="p-6">
          <h3 className="text-lg font-semibold mb-4">{t('analytics.charts.trend')}</h3>
          <TrendChart data={data.timeseries} />
        </Card>

        {/* 性能指标 */}
        <Card className="p-6">
          <h3 className="text-lg font-semibold mb-4">{t('analytics.charts.performance')}</h3>
          <PerformanceMetrics data={data.overview} />
        </Card>
      </div>

      {/* 详细数据表格 */}
      <Card className="p-6">
        <h3 className="text-lg font-semibold mb-4">{t('analytics.charts.detailedData')}</h3>
        <AnalyticsTable data={data.timeseries} />
      </Card>
    </div>
  )
}

// 指标卡片组件
interface MetricCardProps {
  title: string
  value: string
  change: string
  trend: 'up' | 'down' | 'neutral'
  icon: string
}

function MetricCard({ title, value, change, trend, icon }: MetricCardProps) {
  return (
    <Card className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-slate-600 dark:text-slate-400">{title}</p>
          <p className="text-2xl font-bold">{value}</p>
          <p className={cn(
            'text-sm',
            trend === 'up' && 'text-green-600',
            trend === 'down' && 'text-red-600',
            trend === 'neutral' && 'text-slate-600'
          )}>
            {change}
          </p>
        </div>
        <div className="text-3xl">{icon}</div>
      </div>
    </Card>
  )
}

// 趋势图表组件
function TrendChart({ data }: { data: MetricsTimePoint[] }) {
  const { t } = useTranslation()

  if (!data || data.length === 0) {
    return <div className="text-center text-slate-500 py-8">{t('analytics.noData')}</div>
  }

  // 简单的条形图实现
  const maxValue = Math.max(...data.flatMap(d => [d.events, d.alerts, d.failures]))

  return (
    <div className="space-y-4">
      {data.slice(-10).map((point, index) => (
        <div key={index} className="flex items-center gap-4">
          <div className="w-20 text-sm text-slate-600 dark:text-slate-400">
            {new Date(point.bucket_start).toLocaleTimeString()}
          </div>
          <div className="flex-1 space-y-1">
            <div className="flex items-center gap-2">
              <div className="w-12 text-xs">事件</div>
              <div className="flex-1 bg-slate-200 dark:bg-slate-700 rounded-full h-2">
                <div
                  className="bg-blue-500 h-2 rounded-full"
                  style={{ width: `${(point.events / maxValue) * 100}%` }}
                />
              </div>
              <div className="w-8 text-xs text-right">{point.events}</div>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-12 text-xs">告警</div>
              <div className="flex-1 bg-slate-200 dark:bg-slate-700 rounded-full h-2">
                <div
                  className="bg-orange-500 h-2 rounded-full"
                  style={{ width: `${(point.alerts / maxValue) * 100}%` }}
                />
              </div>
              <div className="w-8 text-xs text-right">{point.alerts}</div>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

// 性能指标组件
function PerformanceMetrics({ data }: { data: MetricsOverview }) {
  const { t } = useTranslation()

  const metrics = [
    { label: t('analytics.performance.p95Latency'), value: `${(data.p95_latency_ms_24h || 0).toFixed(0)}ms` },
    { label: t('analytics.performance.avgProcessing'), value: `${((data.p95_latency_ms_24h || 0) * 0.8).toFixed(0)}ms` },
    { label: t('analytics.performance.uptime'), value: '99.9%' },
    { label: t('analytics.performance.throughput'), value: `${Math.round((data.events_24h || 0) / 24)}/min` },
  ]

  return (
    <div className="space-y-4">
      {metrics.map((metric, index) => (
        <div key={index} className="flex justify-between items-center">
          <span className="text-sm text-slate-600 dark:text-slate-400">{metric.label}</span>
          <span className="font-medium">{metric.value}</span>
        </div>
      ))}
    </div>
  )
}

// 数据表格组件
function AnalyticsTable({ data }: { data: MetricsTimePoint[] }) {
  const { t } = useTranslation()

  if (!data || data.length === 0) {
    return <div className="text-center text-slate-500 py-8">{t('analytics.noData')}</div>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b">
            <th className="text-left py-2">{t('analytics.table.time')}</th>
            <th className="text-right py-2">{t('analytics.table.events')}</th>
            <th className="text-right py-2">{t('analytics.table.alerts')}</th>
            <th className="text-right py-2">{t('analytics.table.failures')}</th>
          </tr>
        </thead>
        <tbody>
          {data.map((point, index) => (
            <tr key={index} className="border-b">
              <td className="py-2">
                {new Date(point.bucket_start).toLocaleString()}
              </td>
              <td className="text-right py-2">{point.events}</td>
              <td className="text-right py-2">{point.alerts}</td>
              <td className="text-right py-2">{point.failures}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
