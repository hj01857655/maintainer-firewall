import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { apiFetch } from '../api'

type EventItem = {
  id: number
  delivery_id: string
  event_type: string
  action: string
  repository_full_name: string
  sender_login: string
  received_at: string
}

type EventsResponse = {
  ok: boolean
  items: EventItem[]
  total: number
  message?: string
}

type SyncStatus = {
  running: boolean
  last_started_at?: string
  last_finished_at?: string
  last_success_at?: string
  last_saved: number
  last_total: number
  last_error?: string
  success_count: number
  failure_count: number
}

type SyncStatusResponse = {
  ok: boolean
  source: string
  status: SyncStatus
  message?: string
}

type SyncTriggerResponse = {
  ok: boolean
  source: string
  sync: boolean
  saved: number
  total: number
  message?: string
}

export function EventsPage() {
  const { t } = useTranslation()
  const [events, setEvents] = useState<EventItem[]>([])
  const [error, setError] = useState('')
  const [eventTypeFilter, setEventTypeFilter] = useState('')
  const [actionFilter, setActionFilter] = useState('')
  const [offset, setOffset] = useState(0)
  const [total, setTotal] = useState(0)
  const [syncStatus, setSyncStatus] = useState<SyncStatus | null>(null)
  const [syncing, setSyncing] = useState(false)
  const [syncMessage, setSyncMessage] = useState('')
  const limit = 20

  function reloadEvents(nextOffset = offset) {
    const params = new URLSearchParams({ limit: String(limit), offset: String(nextOffset) })
    if (eventTypeFilter) params.set('event_type', eventTypeFilter)
    if (actionFilter) params.set('action', actionFilter)

    apiFetch(`/api/events?${params.toString()}`)
      .then(async (r) => {
        if (!r.ok) {
          const body = await r.text()
          throw new Error(`events HTTP ${r.status} ${body}`.trim())
        }
        return r.json() as Promise<EventsResponse>
      })
      .then((data: EventsResponse) => {
        if (!data.ok) throw new Error(data.message || 'events response not ok')
        setEvents(data.items)
        setTotal(data.total)
        setError('')
      })
      .catch((e: Error) => setError(e.message))
  }

  function loadSyncStatus() {
    apiFetch('/api/events/sync-status')
      .then(async (r) => {
        if (!r.ok) {
          const body = await r.text()
          throw new Error(`sync-status HTTP ${r.status} ${body}`.trim())
        }
        return r.json() as Promise<SyncStatusResponse>
      })
      .then((data) => {
        if (!data.ok) throw new Error(data.message || 'sync status response not ok')
        setSyncStatus(data.status)
      })
      .catch((e: Error) => setError(e.message))
  }

  useEffect(() => {
    reloadEvents(offset)
  }, [eventTypeFilter, actionFilter, offset])

  useEffect(() => {
    loadSyncStatus()
    const timer = window.setInterval(loadSyncStatus, 10000)
    return () => window.clearInterval(timer)
  }, [])

  const currentPage = useMemo(() => Math.floor(offset / limit) + 1, [offset])
  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total])

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white/95 p-5 shadow-sm md:p-6">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900">{t('events.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600">{t('events.subtitle')}</p>

        <div className="mt-4 grid gap-3 md:grid-cols-3">
          <label className="block text-sm font-medium text-slate-700">
            <span>{t('events.filters.eventType')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              value={eventTypeFilter}
              onChange={(e) => {
                setOffset(0)
                setEventTypeFilter(e.target.value)
              }}
              placeholder={t('events.filters.eventTypePlaceholder')}
            />
          </label>

          <label className="block text-sm font-medium text-slate-700">
            <span>{t('events.filters.action')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              value={actionFilter}
              onChange={(e) => {
                setOffset(0)
                setActionFilter(e.target.value)
              }}
              placeholder={t('events.filters.actionPlaceholder')}
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

        <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50 p-4">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="space-y-1 text-sm text-slate-700">
              <p className="font-semibold text-slate-900">{t('events.sync.title')}</p>
              <p>{t('events.sync.running')}: {syncStatus?.running ? t('events.sync.yes') : t('events.sync.no')}</p>
              <p>{t('events.sync.lastSuccess')}: {syncStatus?.last_success_at ? new Date(syncStatus.last_success_at).toLocaleString() : '-'}</p>
              <p>{t('events.sync.lastResult')}: {syncStatus ? `${syncStatus.last_saved} / ${syncStatus.last_total}` : '-'}</p>
              <p>{t('events.sync.stats')}: {syncStatus ? `${syncStatus.success_count} / ${syncStatus.failure_count}` : '-'}</p>
              {syncStatus?.last_error ? <p className="text-red-600">{t('events.sync.lastError')}: {syncStatus.last_error}</p> : null}
            </div>
            <div className="flex w-full gap-2 md:w-auto">
              <button
                className="h-11 min-w-[120px] cursor-pointer rounded-xl border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                onClick={() => loadSyncStatus()}
                aria-label={t('events.sync.refreshStatus')}
              >
                {t('events.sync.refreshStatus')}
              </button>
              <button
                className="h-11 min-w-[140px] cursor-pointer rounded-xl bg-orange-500 px-4 text-sm font-semibold text-white transition-colors duration-200 hover:bg-orange-600 focus:outline-none focus:ring-2 focus:ring-orange-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
                onClick={async () => {
                  try {
                    setSyncing(true)
                    setSyncMessage('')
                    const r = await apiFetch('/api/events?source=github&sync=true')
                    if (!r.ok) {
                      const body = await r.text()
                      throw new Error(`sync HTTP ${r.status} ${body}`.trim())
                    }
                    const data = (await r.json()) as SyncTriggerResponse
                    if (!data.ok) throw new Error(data.message || 'sync response not ok')
                    setSyncMessage(t('events.sync.synced', { saved: data.saved, total: data.total }))
                    setOffset(0)
                    reloadEvents(0)
                    loadSyncStatus()
                  } catch (e) {
                    setError((e as Error).message)
                  } finally {
                    setSyncing(false)
                  }
                }}
                disabled={syncing || !!syncStatus?.running}
                aria-label={t('events.sync.syncNow')}
              >
                {syncing ? t('events.sync.syncing') : t('events.sync.syncNow')}
              </button>
            </div>
          </div>
          {syncMessage ? <p className="mt-2 text-sm text-emerald-700">{syncMessage}</p> : null}
        </div>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white/95 shadow-sm">
        <table className="min-w-[760px] w-full text-sm">
          <thead className="bg-slate-50 text-slate-700">
            <tr>
              <th className="px-3 py-3 text-left font-semibold">{t('events.table.id')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('events.table.type')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('events.table.action')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('events.table.repository')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('events.table.sender')}</th>
              <th className="px-3 py-3 text-left font-semibold">{t('events.table.receivedAt')}</th>
            </tr>
          </thead>
          <tbody>
            {events.map((evt) => (
              <tr key={evt.id} className="border-t border-slate-200 hover:bg-slate-50/80">
                <td className="px-3 py-3 text-slate-900">{evt.id}</td>
                <td className="px-3 py-3 text-slate-900">{evt.event_type}</td>
                <td className="px-3 py-3 text-slate-900">{evt.action || '-'}</td>
                <td className="px-3 py-3 text-slate-900">{evt.repository_full_name}</td>
                <td className="px-3 py-3 text-slate-900">{evt.sender_login}</td>
                <td className="px-3 py-3 text-slate-900">{new Date(evt.received_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {events.length === 0 ? <p className="text-sm text-slate-600">{t('events.empty')}</p> : null}

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
