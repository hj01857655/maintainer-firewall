import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { authHeaders } from '../auth'

type FailureItem = {
  id: number
  delivery_id: string
  event_type: string
  action: string
  repository_full_name: string
  suggestion_type: string
  suggestion_value: string
  error_message: string
  attempt_count: number
  occurred_at: string
}

type FailuresResponse = {
  ok: boolean
  items: FailureItem[]
  total: number
  message?: string
}

export function FailuresPage() {
  const { t } = useTranslation()
  const [items, setItems] = useState<FailureItem[]>([])
  const [error, setError] = useState('')
  const [offset, setOffset] = useState(0)
  const [total, setTotal] = useState(0)
  const [retryingId, setRetryingId] = useState<number | null>(null)
  const limit = 20

  function loadFailures(nextOffset = offset) {
    fetch(`/api/action-failures?limit=${limit}&offset=${nextOffset}`, { headers: authHeaders() })
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text())
        return r.json() as Promise<FailuresResponse>
      })
      .then((data) => {
        if (!data.ok) throw new Error(data.message || 'load failures failed')
        setItems(data.items)
        setTotal(data.total)
        setError('')
      })
      .catch((e: Error) => setError(e.message))
  }

  useEffect(() => {
    loadFailures(offset)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [offset])

  const currentPage = useMemo(() => Math.floor(offset / limit) + 1, [offset])
  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total])

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900">{t('failures.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600">{t('failures.subtitle')}</p>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm">
        <table className="min-w-[1200px] w-full text-sm">
          <thead className="bg-slate-100 text-slate-700">
            <tr>
              <th className="px-3 py-2 text-left">ID</th>
              <th className="px-3 py-2 text-left">Delivery</th>
              <th className="px-3 py-2 text-left">Type</th>
              <th className="px-3 py-2 text-left">Action</th>
              <th className="px-3 py-2 text-left">Repository</th>
              <th className="px-3 py-2 text-left">Suggestion</th>
              <th className="px-3 py-2 text-left">Attempts</th>
              <th className="px-3 py-2 text-left">Error</th>
              <th className="px-3 py-2 text-left">Occurred At</th>
              <th className="px-3 py-2 text-left">Operate</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-t border-slate-200">
                <td className="px-3 py-2">{item.id}</td>
                <td className="px-3 py-2">{item.delivery_id}</td>
                <td className="px-3 py-2">{item.event_type}</td>
                <td className="px-3 py-2">{item.action}</td>
                <td className="px-3 py-2">{item.repository_full_name}</td>
                <td className="px-3 py-2">{item.suggestion_type}:{item.suggestion_value}</td>
                <td className="px-3 py-2">{item.attempt_count}</td>
                <td className="px-3 py-2 max-w-[420px] break-words">{item.error_message}</td>
                <td className="px-3 py-2">{new Date(item.occurred_at).toLocaleString()}</td>
                <td className="px-3 py-2">
                  <button
                    type="button"
                    disabled={retryingId === item.id}
                    className="h-9 min-w-[96px] cursor-pointer rounded-lg border border-slate-300 bg-white px-3 text-xs font-semibold text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                    onClick={async () => {
                      setRetryingId(item.id)
                      setError('')
                      try {
                        const resp = await fetch(`/api/action-failures/${item.id}/retry`, {
                          method: 'POST',
                          headers: authHeaders(),
                        })
                        if (!resp.ok) {
                          throw new Error(await resp.text())
                        }
                        const data: { ok: boolean; message?: string } = await resp.json()
                        if (!data.ok) throw new Error(data.message || 'retry failed')
                        loadFailures(offset)
                      } catch (err) {
                        const msg = err instanceof Error ? err.message : 'retry failed'
                        setError(msg)
                      } finally {
                        setRetryingId(null)
                      }
                    }}
                  >
                    {retryingId === item.id ? 'Retrying...' : 'Retry'}
                  </button>
                </td>
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
