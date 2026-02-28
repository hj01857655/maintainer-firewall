import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { authHeaders } from '../auth'

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
  const limit = 20

  useEffect(() => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (actor) params.set('actor', actor)
    if (action) params.set('action', action)

    fetch(`/api/audit-logs?${params.toString()}`, { headers: authHeaders() })
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
  }, [offset, actor, action])

  const currentPage = useMemo(() => Math.floor(offset / limit) + 1, [offset])
  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total])

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900">{t('audit.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600">{t('audit.subtitle')}</p>

        <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-3">
          <label className="block text-sm font-medium text-slate-700">
            <span>Actor</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              value={actor}
              onChange={(e) => {
                setOffset(0)
                setActor(e.target.value)
              }}
              placeholder="admin"
            />
          </label>
          <label className="block text-sm font-medium text-slate-700">
            <span>Action</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              value={action}
              onChange={(e) => {
                setOffset(0)
                setAction(e.target.value)
              }}
              placeholder="rule.create"
            />
          </label>
        </div>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm">
        <table className="min-w-[1080px] w-full text-sm">
          <thead className="bg-slate-100 text-slate-700">
            <tr>
              <th className="px-3 py-2 text-left">ID</th>
              <th className="px-3 py-2 text-left">Actor</th>
              <th className="px-3 py-2 text-left">Action</th>
              <th className="px-3 py-2 text-left">Target</th>
              <th className="px-3 py-2 text-left">Target ID</th>
              <th className="px-3 py-2 text-left">Payload</th>
              <th className="px-3 py-2 text-left">Created At</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-t border-slate-200">
                <td className="px-3 py-2">{item.id}</td>
                <td className="px-3 py-2">{item.actor}</td>
                <td className="px-3 py-2">{item.action}</td>
                <td className="px-3 py-2">{item.target}</td>
                <td className="px-3 py-2">{item.target_id}</td>
                <td className="px-3 py-2 max-w-[420px] break-words">
                  <details>
                    <summary className="cursor-pointer text-blue-600">View JSON</summary>
                    <pre className="mt-2 overflow-x-auto rounded-lg bg-slate-900 p-2 text-xs text-slate-100">
                      {formatJSON(item.payload)}
                    </pre>
                  </details>
                </td>
                <td className="px-3 py-2">{new Date(item.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className="h-11 min-w-[88px] cursor-pointer rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          onClick={() => setOffset((v) => Math.max(0, v - limit))}
          disabled={offset === 0}
        >
          {t('common.prev')}
        </button>
        <button
          type="button"
          className="h-11 min-w-[88px] cursor-pointer rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          onClick={() => setOffset((v) => v + limit)}
          disabled={currentPage >= totalPages}
        >
          {t('common.next')}
        </button>
        <span className="text-sm text-slate-600">{t('common.pageSummary', { current: currentPage, total: totalPages, count: total })}</span>
      </div>

      {items.length === 0 ? <p className="text-sm text-slate-600">{t('common.empty')}</p> : null}
      {error ? <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600">{t('common.error', { message: error })}</p> : null}
    </section>
  )
}
