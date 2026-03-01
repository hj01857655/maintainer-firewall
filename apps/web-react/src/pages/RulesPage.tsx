import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { authHeaders } from '../auth'

type RuleItem = {
  id: number
  event_type: 'issues' | 'pull_request'
  keyword: string
  suggestion_type: 'label' | 'comment'
  suggestion_value: string
  reason: string
  is_active: boolean
  created_at: string
}

type RulesResponse = {
  ok: boolean
  items: RuleItem[]
  total: number
  message?: string
}

type RuleFilterOptions = {
  event_types: string[]
  suggestion_types: string[]
  active_states: string[]
}

type RuleFilterOptionsResponse = {
  ok: boolean
  options: RuleFilterOptions
  message?: string
}

export function RulesPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const [rules, setRules] = useState<RuleItem[]>([])
  const [error, setError] = useState('')
  const [keywordFilter, setKeywordFilter] = useState(searchParams.get('keyword') || '')
  const [eventTypeFilter, setEventTypeFilter] = useState(searchParams.get('event_type') || '')
  const [activeOnly, setActiveOnly] = useState((searchParams.get('active_only') || 'true') === 'true')
  const [offset, setOffset] = useState(0)
  const [total, setTotal] = useState(0)
  const [filterOptions, setFilterOptions] = useState<RuleFilterOptions>({ event_types: [], suggestion_types: [], active_states: [] })
  const [creating, setCreating] = useState(false)
  const [togglingId, setTogglingId] = useState<number | null>(null)

  const [formEventType, setFormEventType] = useState<'issues' | 'pull_request'>('issues')
  const [formKeyword, setFormKeyword] = useState('')
  const [formSuggestionType, setFormSuggestionType] = useState<'label' | 'comment'>('label')
  const [formSuggestionValue, setFormSuggestionValue] = useState('')
  const [formReason, setFormReason] = useState('')
  const [formIsActive, setFormIsActive] = useState(true)

  const limit = 20

  function reloadRules(nextOffset = offset) {
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String(nextOffset),
      active_only: String(activeOnly),
    })
    if (eventTypeFilter) params.set('event_type', eventTypeFilter)
    if (keywordFilter) params.set('keyword', keywordFilter)

    fetch(`/api/rules?${params.toString()}`, { headers: authHeaders() })
      .then(async (r) => {
        if (!r.ok) {
          const body = await r.text()
          throw new Error(`rules HTTP ${r.status} ${body}`.trim())
        }
        return r.json() as Promise<RulesResponse>
      })
      .then((data) => {
        if (!data.ok) throw new Error(data.message || 'rules response not ok')
        setRules(data.items)
        setTotal(data.total)
        setError('')
      })
      .catch((e: Error) => setError(e.message))
  }

  useEffect(() => {
    fetch('/api/rules/filter-options', { headers: authHeaders() })
      .then(async (r) => {
        if (!r.ok) {
          const body = await r.text()
          throw new Error(`rules filter-options HTTP ${r.status} ${body}`.trim())
        }
        return r.json() as Promise<RuleFilterOptionsResponse>
      })
      .then((data) => {
        if (!data.ok) throw new Error(data.message || 'rules filter options response not ok')
        setFilterOptions(data.options)
      })
      .catch(() => {
        // 不阻塞主列表加载，失败时回退到当前页去重选项
      })
  }, [])

  useEffect(() => {
    reloadRules(offset)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [offset, eventTypeFilter, keywordFilter, activeOnly])

  const eventTypeOptions = useMemo(() => {
    const base = filterOptions.event_types.length > 0
      ? [...filterOptions.event_types]
      : Array.from(new Set(rules.map((item) => item.event_type?.trim()).filter(Boolean) as string[])).sort((a, b) => a.localeCompare(b))
    if (eventTypeFilter && !base.includes(eventTypeFilter)) {
      base.unshift(eventTypeFilter)
    }
    return base
  }, [filterOptions.event_types, rules, eventTypeFilter])

  const currentPage = useMemo(() => Math.floor(offset / limit) + 1, [offset])
  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total])

  async function onCreateRule(e: React.FormEvent) {
    e.preventDefault()
    setCreating(true)
    setError('')
    try {
      const resp = await fetch('/api/rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify({
          event_type: formEventType,
          keyword: formKeyword.trim(),
          suggestion_type: formSuggestionType,
          suggestion_value: formSuggestionValue.trim(),
          reason: formReason.trim(),
          is_active: formIsActive,
        }),
      })
      if (!resp.ok) {
        const body = await resp.text()
        throw new Error(`create rule HTTP ${resp.status} ${body}`.trim())
      }
      const data: { ok: boolean; message?: string } = await resp.json()
      if (!data.ok) throw new Error(data.message || 'create rule failed')

      setFormKeyword('')
      setFormSuggestionValue('')
      setFormReason('')
      setOffset(0)
      reloadRules(0)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'create rule failed'
      setError(msg)
    } finally {
      setCreating(false)
    }
  }

  async function onToggleActive(rule: RuleItem) {
    setTogglingId(rule.id)
    setError('')
    try {
      const resp = await fetch(`/api/rules/${rule.id}/active`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify({ is_active: !rule.is_active }),
      })
      if (!resp.ok) {
        const body = await resp.text()
        throw new Error(`toggle rule HTTP ${resp.status} ${body}`.trim())
      }
      const data: { ok: boolean; message?: string } = await resp.json()
      if (!data.ok) throw new Error(data.message || 'toggle rule failed')
      reloadRules(offset)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'toggle rule failed'
      setError(msg)
    } finally {
      setTogglingId(null)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white/95 p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t('rules.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">{t('rules.subtitle')}</p>
      </div>

      <form onSubmit={onCreateRule} className="rounded-2xl border border-slate-200 bg-white/95 p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <h2 className="m-0 text-lg font-semibold text-slate-900 dark:text-slate-100">{t('rules.create.title')}</h2>
        {(searchParams.get('keyword') || searchParams.get('event_type')) ? (
          <p className="mt-2 rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-700 dark:border-blue-500/40 dark:bg-blue-500/10 dark:text-blue-200">
            {t('rules.filters.prefilledHint')}
          </p>
        ) : null}
        <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            <span>{t('rules.create.eventType')}</span>
            <select
              className="mt-2 h-11 w-full cursor-pointer rounded-xl border border-slate-300 bg-white px-3 text-base text-slate-900 outline-none transition-colors duration-200 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={formEventType}
              onChange={(e) => setFormEventType(e.target.value as 'issues' | 'pull_request')}
            >
              <option value="issues">issues</option>
              <option value="pull_request">pull_request</option>
            </select>
          </label>

          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            <span>{t('rules.create.keyword')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={formKeyword}
              onChange={(e) => setFormKeyword(e.target.value)}
              placeholder={t('rules.create.keywordPlaceholder')}
              required
            />
          </label>

          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            <span>{t('rules.create.suggestionType')}</span>
            <select
              className="mt-2 h-11 w-full cursor-pointer rounded-xl border border-slate-300 bg-white px-3 text-base text-slate-900 outline-none transition-colors duration-200 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={formSuggestionType}
              onChange={(e) => setFormSuggestionType(e.target.value as 'label' | 'comment')}
            >
              <option value="label">label</option>
              <option value="comment">comment</option>
            </select>
          </label>

          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            <span>{t('rules.create.suggestionValue')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={formSuggestionValue}
              onChange={(e) => setFormSuggestionValue(e.target.value)}
              placeholder={t('rules.create.suggestionValuePlaceholder')}
              required
            />
          </label>

          <label className="block text-sm font-medium text-slate-700 md:col-span-2 dark:text-slate-300">
            <span>{t('rules.create.reason')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={formReason}
              onChange={(e) => setFormReason(e.target.value)}
              placeholder={t('rules.create.reasonPlaceholder')}
              required
            />
          </label>
        </div>

        <label className="mt-4 inline-flex min-h-11 cursor-pointer items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
          <input
            type="checkbox"
            checked={formIsActive}
            onChange={(e) => setFormIsActive(e.target.checked)}
            className="h-4 w-4"
          />
          <span>{t('rules.create.active')}</span>
        </label>

        <div className="mt-4">
          <button
            type="submit"
            disabled={creating}
            className="h-11 min-w-[128px] cursor-pointer rounded-xl bg-blue-600 px-4 text-sm font-semibold text-white transition-colors duration-200 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {creating ? t('rules.create.creating') : t('rules.create.submit')}
          </button>
        </div>
      </form>

      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <div className="grid grid-cols-1 gap-3 md:grid-cols-4">
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            <span>{t('rules.filters.eventType')}</span>
            <select
              className="mt-2 h-11 w-full cursor-pointer rounded-xl border border-slate-300 bg-white px-3 text-base text-slate-900 outline-none transition-colors duration-200 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={eventTypeFilter}
              onChange={(e) => {
                setOffset(0)
                setEventTypeFilter(e.target.value)
              }}
              aria-label={t('rules.filters.eventType')}
            >
              <option value="">{t('common.all')}</option>
              {eventTypeOptions.map((value) => (
                <option key={value} value={value}>{value}</option>
              ))}
            </select>
          </label>

          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            <span>{t('rules.filters.keyword')}</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
              value={keywordFilter}
              onChange={(e) => {
                setOffset(0)
                setKeywordFilter(e.target.value)
              }}
              placeholder={t('rules.filters.keywordPlaceholder')}
            />
          </label>

          <label className="inline-flex min-h-11 cursor-pointer items-center gap-2 text-sm text-slate-700 md:pt-8 dark:text-slate-300">
            <input
              type="checkbox"
              checked={activeOnly}
              onChange={(e) => {
                setOffset(0)
                setActiveOnly(e.target.checked)
              }}
              className="h-4 w-4"
            />
            <span>{t('rules.filters.activeOnly')}</span>
          </label>

          <div className="flex items-end">
            <button
              type="button"
              className="h-11 w-full cursor-pointer rounded-xl bg-blue-600 px-4 text-sm font-semibold text-white transition-colors duration-200 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              onClick={() => {
                setOffset(0)
                reloadRules(0)
              }}
            >
              {t('common.applyFilters')}
            </button>
          </div>
        </div>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white/95 shadow-sm dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <table className="min-w-[1200px] w-full text-sm">
          <thead className="bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300">
            <tr>
              <th className="px-3 py-2 text-left">ID</th>
              <th className="px-3 py-2 text-left">{t('rules.table.eventType')}</th>
              <th className="px-3 py-2 text-left">{t('rules.table.keyword')}</th>
              <th className="px-3 py-2 text-left">{t('rules.table.suggestionType')}</th>
              <th className="px-3 py-2 text-left">{t('rules.table.suggestionValue')}</th>
              <th className="px-3 py-2 text-left">{t('rules.table.reason')}</th>
              <th className="px-3 py-2 text-left">{t('rules.table.active')}</th>
              <th className="px-3 py-2 text-left">{t('rules.table.createdAt')}</th>
              <th className="px-3 py-2 text-left">{t('rules.table.action')}</th>
            </tr>
          </thead>
          <tbody>
            {rules.map((item) => (
              <tr key={item.id} className="border-t border-slate-200 hover:bg-slate-50/70 dark:border-slate-700 dark:hover:bg-slate-800/50">
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.id}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.event_type}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.keyword}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.suggestion_type}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.suggestion_value}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.reason}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{item.is_active ? t('rules.status.active') : t('rules.status.inactive')}</td>
                <td className="px-3 py-2 text-slate-900 dark:text-slate-100">{new Date(item.created_at).toLocaleString()}</td>
                <td className="px-3 py-2">
                  <button
                    type="button"
                    disabled={togglingId === item.id}
                    onClick={() => void onToggleActive(item)}
                    className="h-9 cursor-pointer rounded-lg border border-slate-300 bg-white px-3 text-xs font-semibold text-slate-700 transition-colors duration-200 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
                  >
                    {togglingId === item.id
                      ? t('rules.table.updating')
                      : item.is_active
                        ? t('rules.table.disable')
                        : t('rules.table.enable')}
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

      {rules.length === 0 ? <p className="text-sm text-slate-500 dark:text-slate-400">{t('common.empty')}</p> : null}
      {error ? <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600 dark:border-red-500/40 dark:bg-red-500/10 dark:text-red-300">{t('common.error', { message: error })}</p> : null}
    </section>
  )
}
