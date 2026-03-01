import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { apiFetch } from '../api'

type AuditItem = {
  id: number
  actor: string
  action: string
  target: string
  target_id: string
  payload: string
  created_at: string
}

type AuditResponse = {
  ok: boolean
  items: AuditItem[]
  total: number
  message?: string
}

function formatJSON(v: string): string {
  try {
    return JSON.stringify(JSON.parse(v), null, 2)
  } catch {
    return v
  }
}

export function AuditLogsPage() {
  const { t } = useTranslation()
  const [items, setItems] = useState<AuditItem[]>([])
  const [error, setError] = useState('')
  const [offset, setOffset] = useState(0)
  const [total, setTotal] = useState(0)
  const [actor, setActor] = useState('')
  const [action, setAction] = useState('')
  const [sinceWindow, setSinceWindow] = useState<'all' | '24h' | '7d' | '30d'>('all')
  const limit = 20

  useEffect(() => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (actor) params.set('actor', actor)
    if (action) params.set('action', action)
    if (sinceWindow !== 'all') {
      const now = new Date()
      const since = new Date(now)
      if (sinceWindow === '24h') since.setHours(now.getHours() - 24)
      if (sinceWindow === '7d') since.setDate(now.getDate() - 7)
      if (sinceWindow === '30d') since.setDate(now.getDate() - 30)
      params.set('since', since.toISOString())
    }

    apiFetch(`/api/audit-logs?${params.toString()}`)
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text())
        return r.json() as Promise<AuditResponse>
      })
      .then((data) => {
        if (!data.ok) throw new Error(data.message || 'load audit logs failed')
        setItems(data.items)
        setTotal(data.total)
      })
      .catch((e: Error) => setError(e.message))
  }, [offset, actor, action, sinceWindow])

  const currentPage = useMemo(() => Math.floor(offset / limit) + 1, [offset])
  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total])

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white/95 p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t('audit.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">{t('audit.subtitle')}</p>

        <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-3">
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            <span>{t('audit.filters.actor')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={actor}
              onChange={(e) => {
                setOffset(0)
                setActor(e.target.value)
              }}
              placeholder={t('audit.filters.actorPlaceholder')}
            />
          </label>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            <span>{t('audit.filters.action')}</span>
            <select
              className="mt-2 h-11 w-full cursor-pointer rounded-xl border border-slate-300 bg-white px-3 text-base text-slate-900 outline-none transition-colors duration-200 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={action}
              onChange={(e) => {
                setOffset(0)
                setAction(e.target.value)
              }}
            >
              <option value="">{t('audit.filters.all')}</option>
              <option value="rule.create">rule.create</option>
              <option value="rule.update_active">rule.update_active</option>
              <option value="failure.retry.success">failure.retry.success</option>
              <option value="failure.retry.failed">failure.retry.failed</option>
            </select>
          </label>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            <span>{t('audit.filters.since')}</span>
            <select
              className="mt-2 h-11 w-full cursor-pointer rounded-xl border border-slate-300 bg-white px-3 text-base text-slate-900 outline-none transition-colors duration-200 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={sinceWindow}
              onChange={(e) => {
                setOffset(0)
                setSinceWindow(e.target.value as 'all' | '24h' | '7d' | '30d')
              }}
            >
              <option value="all">{t('audit.filters.all')}</option>
              <option value="24h">{t('audit.filters.last24h')}</option>
              <option value="7d">{t('audit.filters.last7d')}</option>
              <option value="30d">{t('audit.filters.last30d')}</option>
            </select>
          </label>
        </div>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white/95 shadow-sm dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <table className="min-w-[1200px] w-full text-sm">
          <thead className="bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300">
            <tr>
              <th className="px-3 py-2 text-left">{t('audit.table.id')}</th>
              <th className="px-3 py-2 text-left">{t('audit.table.actor')}</th>
              <th className="px-3 py-2 text-left">{t('audit.table.action')}</th>
              <th className="px-3 py-2 text-left">{t('audit.table.target')}</th>
              <th className="px-3 py-2 text-left">{t('audit.table.targetId')}</th>
              <th className="px-3 py-2 text-left">{t('audit.table.payload')}</th>
              <th className="px-3 py-2 text-left">{t('audit.table.createdAt')}</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-t border-slate-200 hover:bg-slate-50/70 dark:border-slate-700 dark:hover:bg-slate-800/50">
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.id}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.actor}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.action}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.target}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.target_id}</td>
                <td className="px-3 py-2 max-w-[420px] break-words text-slate-900 dark:text-slate-100">
                  <details>
                    <summary className="cursor-pointer text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300">{t('common.viewJson')}</summary>
                    <pre className="mt-2 overflow-x-auto rounded-lg bg-slate-900 p-2 text-xs leading-relaxed text-slate-100 dark:bg-slate-800">
                      {formatJSON(item.payload)}
                    </pre>
                  </details>
                </td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{new Date(item.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className="h-11 min-w-[88px] cursor-pointer rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
          onClick={() => setOffset((v) => Math.max(0, v - limit))}
          disabled={offset === 0}
        >
          {t('common.prev')}
        </button>
        <button
          type="button"
          className="h-11 min-w-[88px] cursor-pointer rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
          onClick={() => setOffset((v) => v + limit)}
          disabled={currentPage >= totalPages}
        >
          {t('common.next')}
        </button>
        <span className="text-sm text-slate-600 dark:text-slate-300">{t('common.pageSummary', { current: currentPage, total: totalPages, count: total })}</span>
      </div>

      {items.length === 0 ? <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.empty')}</p> : null}
      {error ? <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600 dark:border-red-500/40 dark:bg-red-500/10 dark:text-red-300">{t('common.error', { message: error })}</p> : null}
    </section>
  )
}
