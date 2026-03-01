import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { apiFetch } from '../api'
import { EmptyState } from '../components/EmptyState'

type Health = {
  status: string
  service: string
}

type MetricsOverview = {
  events_24h: number
  alerts_24h: number
  failures_24h: number
  success_rate_24h: number
  p95_latency_ms_24h: number
}

type MetricsPoint = {
  bucket_start: string
  events: number
  alerts: number
  failures: number
}

export function DashboardPage() {
  const { t } = useTranslation()
  const [health, setHealth] = useState<Health | null>(null)
  const [overview, setOverview] = useState<MetricsOverview | null>(null)
  const [series, setSeries] = useState<MetricsPoint[]>([])
  const [error, setError] = useState('')
  const [windowValue, setWindowValue] = useState<'6h' | '12h' | '24h'>('24h')

  useEffect(() => {
    Promise.all([
      apiFetch('/api/health').then((r) => r.json() as Promise<Health>),
      apiFetch(`/api/metrics/overview?window=${windowValue}`).then(async (r) => {

        return r.json() as Promise<{ ok: boolean; overview: MetricsOverview }>
      }),
      apiFetch(`/api/metrics/timeseries?window=${windowValue}&interval_minutes=60`).then(async (r) => {

        return r.json() as Promise<{ ok: boolean; items: MetricsPoint[] }>
      }),
    ])
      .then(([healthData, overviewResp, seriesResp]) => {
        setHealth(healthData)
        setOverview(overviewResp.overview)
        setSeries(seriesResp.items || [])
      })
      .catch((e: Error) => setError(e.message))
  }, [windowValue])

  const maxY = useMemo(() => {
    const max = series.reduce((acc, item) => Math.max(acc, item.events, item.alerts, item.failures), 0)
    return Math.max(1, max)
  }, [series])

  function linePath(values: number[]) {
    if (values.length === 0) return ''
    return values
      .map((v, i) => {
        const x = (i / Math.max(1, values.length - 1)) * 100
        const y = 100 - (v / maxY) * 100
        return `${x},${y}`
      })
      .join(' ')
  }

  const eventsPath = linePath(series.map((s) => s.events))
  const alertsPath = linePath(series.map((s) => s.alerts))
  const failuresPath = linePath(series.map((s) => s.failures))

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white/95 p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t('dashboard.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">{t('dashboard.subtitle')}</p>
      </div>

      <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5">
        <MetricCard label={t('dashboard.cards.events24h')} value={overview?.events_24h ?? 0} to="/events" />
        <MetricCard label={t('dashboard.cards.alerts24h')} value={overview?.alerts_24h ?? 0} to="/alerts" />
        <MetricCard label={t('dashboard.cards.failures24h')} value={overview?.failures_24h ?? 0} to="/failures" />
        <MetricCard label={t('dashboard.cards.successRate24h')} value={`${(overview?.success_rate_24h ?? 0).toFixed(2)}%`} to="/audit" />
        <MetricCard label={t('dashboard.cards.p95Latency24h')} value={`${Math.round(overview?.p95_latency_ms_24h ?? 0)}`} to="/audit" />
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="m-0 text-lg font-semibold text-slate-900 dark:text-slate-100">{t('dashboard.trendTitle')}</h2>
          <div className="inline-flex rounded-lg border border-slate-300 bg-white p-1 dark:border-slate-600 dark:bg-slate-800">
            {(['6h', '12h', '24h'] as const).map((w) => (
              <button
                key={w}
                type="button"
                className={[
                  'h-8 min-w-[52px] cursor-pointer rounded-md px-3 text-xs font-semibold transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500',
                  windowValue === w ? 'bg-blue-600 text-white' : 'text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700',
                ].join(' ')}
                onClick={() => setWindowValue(w)}
              >
                {w}
              </button>
            ))}
          </div>
        </div>
        <div className="mt-3 h-56 w-full rounded-xl border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
          {series.length === 0 ? (
            <EmptyState
              title={t('dashboard.noTrendData')}
              description={t('dashboard.noTrendDataDescription')}
              icon={
                <svg className="h-8 w-8 text-slate-400 dark:text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              }
            />
          ) : (
            <svg viewBox="0 0 100 100" preserveAspectRatio="none" className="h-full w-full">
              <polyline points={eventsPath} fill="none" stroke="#3B82F6" strokeWidth="1.8" />
              <polyline points={alertsPath} fill="none" stroke="#F59E0B" strokeWidth="1.8" />
              <polyline points={failuresPath} fill="none" stroke="#EF4444" strokeWidth="1.8" />
            </svg>
          )}
        </div>
        <div className="mt-2 flex flex-wrap gap-3 text-xs text-slate-500 dark:text-slate-400">
          <span className="inline-flex items-center gap-1"><i className="inline-block h-2 w-2 rounded-full bg-blue-500" />{t('dashboard.legend.events')}</span>
          <span className="inline-flex items-center gap-1"><i className="inline-block h-2 w-2 rounded-full bg-amber-500" />{t('dashboard.legend.alerts')}</span>
          <span className="inline-flex items-center gap-1"><i className="inline-block h-2 w-2 rounded-full bg-red-500" />{t('dashboard.legend.failures')}</span>
        </div>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <h2 className="m-0 text-lg font-semibold text-slate-900 dark:text-slate-100">{t('dashboard.apiHealth')}</h2>
        {health ? (
          <pre className="mt-3 overflow-x-auto rounded-xl bg-slate-900 p-4 text-sm text-slate-100">
            {JSON.stringify(health, null, 2)}
          </pre>
        ) : (
          <p className="mt-3 text-sm text-slate-600 dark:text-slate-300">{error || t('common.loading')}</p>
        )}
      </div>

      {error ? <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600 dark:border-red-500/40 dark:bg-red-500/10 dark:text-red-300">{t('common.error', { message: error })}</p> : null}
    </section>
  )
}

function MetricCard({ label, value, to }: { label: string; value: number | string; to?: string }) {
  const body = (
    <div className="rounded-2xl border border-slate-200 bg-white/95 p-4 shadow-sm transition-colors duration-200 hover:border-blue-300 hover:bg-blue-50/40 dark:border-slate-700 dark:bg-slate-900/80 dark:hover:border-blue-500/40 dark:hover:bg-blue-500/10">
      <p className="text-xs text-slate-500 dark:text-slate-400">{label}</p>
      <p className="mt-1 text-2xl font-semibold text-slate-900 dark:text-slate-100">{value}</p>
    </div>
  )

  if (!to) return body

  return (
    <Link to={to} className="cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-2xl">
      {body}
    </Link>
  )
}
