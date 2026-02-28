import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { apiFetch } from '../api'

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
  retry_count: number
  last_retry_status: string
  last_retry_message: string
  last_retry_at?: string
  is_resolved: boolean
  occurred_at: string
}

type FailuresResponse = {
  ok: boolean
  items: FailureItem[]
  total: number
  message?: string
}

type ConfigStatusResponse = {
  ok: boolean
  github_token_configured: boolean
}

export function FailuresPage() {
  const { t } = useTranslation()
  const [items, setItems] = useState<FailureItem[]>([])
  const [error, setError] = useState('')
  const [offset, setOffset] = useState(0)
  const [total, setTotal] = useState(0)
  const [retryingId, setRetryingId] = useState<number | null>(null)
  const [includeResolved, setIncludeResolved] = useState(false)
  const [githubTokenConfigured, setGithubTokenConfigured] = useState(false)
  const limit = 20

  function loadFailures(nextOffset = offset) {
    apiFetch(`/api/action-failures?limit=${limit}&offset=${nextOffset}&include_resolved=${includeResolved}`)
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

  function loadConfigStatus() {
    apiFetch('/api/config-status')
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text())
        return r.json() as Promise<ConfigStatusResponse>
      })
      .then((data) => {
        if (!data.ok) throw new Error('load config status failed')
        setGithubTokenConfigured(Boolean(data.github_token_configured))
      })
      .catch(() => setGithubTokenConfigured(false))
  }

  useEffect(() => {
    loadFailures(offset)
    loadConfigStatus()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [offset, includeResolved])

  const currentPage = useMemo(() => Math.floor(offset / limit) + 1, [offset])
  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total])

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900">{t('failures.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600">{t('failures.subtitle')}</p>
        <label className="mt-3 inline-flex min-h-11 cursor-pointer items-center gap-2 text-sm text-slate-700">
          <input
            type="checkbox"
            checked={includeResolved}
            onChange={(e) => {
              setOffset(0)
              setIncludeResolved(e.target.checked)
            }}
            className="h-4 w-4"
          />
          <span>{t('failures.includeResolved')}</span>
        </label>
        {!githubTokenConfigured ? (
          <p className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700">
            {t('failures.tokenMissing')}
          </p>
        ) : null}
      </div>

      <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white/95 shadow-sm">
        <table className="min-w-[1200px] w-full text-sm">
          <thead className="bg-slate-100 text-slate-700">
            <tr>
              <th className="px-3 py-2 text-left">{t('failures.table.id')}</th>
              <th className="px-3 py-2 text-left">{t('failures.table.delivery')}</th>
              <th className="px-3 py-2 text-left">{t('failures.table.type')}</th>
              <th className="px-3 py-2 text-left">{t('failures.table.action')}</th>
              <th className="px-3 py-2 text-left">{t('failures.table.repository')}</th>
              <th className="px-3 py-2 text-left">{t('failures.table.suggestion')}</th>
              <th className="px-3 py-2 text-left">{t('failures.table.attempts')}</th>
              <th className="px-3 py-2 text-left">{t('failures.table.error')}</th>
              <th className="px-3 py-2 text-left">{t('failures.table.occurredAt')}</th>
              <th className="px-3 py-2 text-left">{t('failures.table.operation')}</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-t border-slate-200 hover:bg-slate-50/70">
                 <td className="px-3 py-2 text-slate-900">{item.id}</td>
                 <td className="px-3 py-2 text-slate-900">{item.delivery_id}</td>
                 <td className="px-3 py-2 text-slate-900">{item.event_type}</td>
                 <td className="px-3 py-2 text-slate-900">{item.action}</td>
                 <td className="px-3 py-2 text-slate-900">{item.repository_full_name}</td>
                 <td className="px-3 py-2 text-slate-900">{item.suggestion_type}:{item.suggestion_value}</td>
                 <td className="px-3 py-2 text-slate-900">{item.attempt_count} / retry {item.retry_count}</td>
                 <td className="px-3 py-2 max-w-[420px] break-words text-slate-900">
                  <div>{item.error_message}</div>
                  <div className="mt-1 text-xs text-slate-500">
                    {t('failures.table.last')}: {item.last_retry_status}
                    {item.last_retry_message ? ` · ${item.last_retry_message}` : ''}
                  </div>
                </td>
                <td className="px-3 py-2">{new Date(item.occurred_at).toLocaleString()}</td>
                <td className="px-3 py-2">
                  {item.is_resolved ? (
                    <span className="inline-flex rounded-lg border border-green-200 bg-green-50 px-2 py-1 text-xs font-semibold text-green-700">{t('failures.status.resolved')}</span>
                  ) : (
                    <button
                      type="button"
                      disabled={retryingId === item.id || !githubTokenConfigured}
                      className="h-9 min-w-[96px] cursor-pointer rounded-lg border border-slate-300 bg-white px-3 text-xs font-semibold text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                      onClick={async () => {
                        setRetryingId(item.id)
                        setError('')
                        try {
                          const resp = await apiFetch(`/api/action-failures/${item.id}/retry`, {
                            method: 'POST',
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
                      {!githubTokenConfigured ? t('failures.retry.required') : retryingId === item.id ? t('failures.retry.retrying') : t('failures.retry.action')}
                    </button>
                  )}
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
