import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { apiFetch } from '../api'

type AlertItem = {
  delivery_id: string
  event_type: string
  action: string
  repository_full_name: string
  sender_login: string
  rule_matched: string
  suggestion_type: string
  suggestion_value: string
  reason: string
  created_at: string
}

type AlertsResponse = {
  ok: boolean
  items: AlertItem[]
  total: number
  message?: string
}

export function AlertsPage() {
  const { t } = useTranslation()
  const [alerts, setAlerts] = useState<AlertItem[]>([])
  const [error, setError] = useState('')
  const [eventTypeFilter, setEventTypeFilter] = useState('')
  const [actionFilter, setActionFilter] = useState('')
  const [suggestionTypeFilter, setSuggestionTypeFilter] = useState('')
  const [offset, setOffset] = useState(0)
  const [total, setTotal] = useState(0)
  const limit = 20

  useEffect(() => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (eventTypeFilter) params.set('event_type', eventTypeFilter)
    if (actionFilter) params.set('action', actionFilter)
    if (suggestionTypeFilter) params.set('suggestion_type', suggestionTypeFilter)

    apiFetch(`/api/alerts?${params.toString()}`)
      .then(async (r) => {
        if (!r.ok) {
          const body = await r.text()
          throw new Error(`alerts HTTP ${r.status} ${body}`.trim())
        }
        return r.json() as Promise<AlertsResponse>
      })
      .then((data: AlertsResponse) => {
        if (!data.ok) throw new Error(data.message || 'alerts response not ok')
        setAlerts(data.items)
        setTotal(data.total)
        setError('')
      })
      .catch((e: Error) => setError(e.message))
  }, [eventTypeFilter, actionFilter, suggestionTypeFilter, offset])



  const currentPage = useMemo(() => Math.floor(offset / limit) + 1, [offset])
  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total])

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900">{t('alerts.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600">{t('alerts.subtitle')}</p>

        <div className="mt-4 grid gap-3 md:grid-cols-4">
          <label className="block text-sm font-medium text-slate-700">
            <span>{t('alerts.filters.eventType')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              value={eventTypeFilter}
              onChange={(e) => {
                setOffset(0)
                setEventTypeFilter(e.target.value)
              }}
              placeholder={t('alerts.filters.eventTypePlaceholder')}
            />
          </label>

          <label className="block text-sm font-medium text-slate-700">
            <span>{t('alerts.filters.action')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              value={actionFilter}
              onChange={(e) => {
                setOffset(0)
                setActionFilter(e.target.value)
              }}
              placeholder={t('alerts.filters.actionPlaceholder')}
            />
          </label>

          <label className="block text-sm font-medium text-slate-700">
            <span>{t('alerts.filters.suggestionType')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              value={suggestionTypeFilter}
              onChange={(e) => {
                setOffset(0)
                setSuggestionTypeFilter(e.target.value)
              }}
              placeholder={t('alerts.filters.suggestionTypePlaceholder')}
            />
          </label>

          <div className="flex items-end">
            <button
              className="h-11 w-full cursor-pointer rounded-xl bg-blue-600 px-4 text-sm font-semibold text-white transition-colors duration-200 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              onClick={() => setOffset(0)}
              aria-label={t('common.applyFilters')}
            >
              {t('common.applyFilters')}
            </button>
          </div>
        </div>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white/95 shadow-sm">
        <table className="min-w-[1120px] w-full text-sm">
          <thead className="bg-slate-50 text-slate-700">
            <tr>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.delivery')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.type')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.action')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.suggestionType')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.suggestionValue')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.ruleMatched')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.reason')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.repository')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.sender')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.createdAt')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('alerts.table.manageRule')}</th>
            </tr>
          </thead>
          <tbody>
            {alerts.map((item) => (
              <tr
                key={`${item.delivery_id}-${item.suggestion_type}-${item.suggestion_value}-${item.rule_matched}`}
                className="border-t border-slate-200 hover:bg-slate-50/70"
              >
                <td className="px-3 py-3 text-slate-900">{item.delivery_id}</td>
                <td className="px-3 py-3 text-slate-900">{item.event_type}</td>
                <td className="px-3 py-3 text-slate-900">{item.action || '-'}</td>
                <td className="px-3 py-3 text-slate-900">{item.suggestion_type}</td>
                <td className="px-3 py-3 text-slate-900">{item.suggestion_value}</td>
                <td className="px-3 py-3 text-slate-900">{item.rule_matched}</td>
                <td className="px-3 py-3 text-slate-900">{item.reason}</td>
                <td className="px-3 py-3 text-slate-900">{item.repository_full_name}</td>
                <td className="px-3 py-3 text-slate-900">{item.sender_login}</td>
                <td className="px-3 py-3 text-slate-900">{new Date(item.created_at).toLocaleString()}</td>
                <td className="px-3 py-3 text-slate-900">
                  <Link
                    to={`/rules?${new URLSearchParams({
                      event_type: item.event_type,
                      keyword: item.rule_matched,
                      active_only: 'false',
                    }).toString()}`}
                    className="inline-flex min-h-9 cursor-pointer items-center rounded-lg border border-slate-300 bg-white px-3 text-xs font-semibold text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                  >
                    {t('alerts.table.goRules')}
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {alerts.length === 0 ? <p className="text-sm text-slate-600">{t('alerts.empty')}</p> : null}

      <div className="flex flex-wrap items-center gap-2">
        <button
          className="h-11 min-w-[88px] cursor-pointer rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          onClick={() => setOffset((v) => Math.max(0, v - limit))}
          disabled={offset === 0}
          aria-label={t('common.prev')}
        >
          {t('common.prev')}
        </button>
        <button
          className="h-11 min-w-[88px] cursor-pointer rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          onClick={() => setOffset((v) => v + limit)}
          disabled={currentPage >= totalPages}
          aria-label={t('common.next')}
        >
          {t('common.next')}
        </button>
        <span className="text-sm text-slate-600">{t('common.pageSummary', { current: currentPage, total: totalPages, count: total })}</span>
      </div>

      {error ? <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600">{t('common.error', { message: error })}</p> : null}
    </section>
  )
}
