import type { CSSProperties } from 'react'
import { useEffect, useMemo, useState } from 'react'

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
  limit: number
  offset: number
  total: number
  event_type?: string
  action?: string
  suggestion_type?: string
}

export function AlertsPage() {
  const [alerts, setAlerts] = useState<AlertItem[]>([])
  const [error, setError] = useState<string>('')
  const [eventTypeFilter, setEventTypeFilter] = useState<string>('')
  const [actionFilter, setActionFilter] = useState<string>('')
  const [suggestionTypeFilter, setSuggestionTypeFilter] = useState<string>('')
  const [offset, setOffset] = useState<number>(0)
  const [total, setTotal] = useState<number>(0)
  const limit = 20

  useEffect(() => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (eventTypeFilter) params.set('event_type', eventTypeFilter)
    if (actionFilter) params.set('action', actionFilter)
    if (suggestionTypeFilter) params.set('suggestion_type', suggestionTypeFilter)

    fetch(`/alerts?${params.toString()}`)
      .then((r) => {
        if (!r.ok) throw new Error(`alerts HTTP ${r.status}`)
        return r.json()
      })
      .then((data: AlertsResponse) => {
        if (!data.ok) throw new Error('alerts response not ok')
        setAlerts(data.items)
        setTotal(data.total)
        setError('')
      })
      .catch((e: Error) => setError((prev) => (prev ? `${prev}; ${e.message}` : e.message)))
  }, [eventTypeFilter, actionFilter, suggestionTypeFilter, offset])

  const currentPage = useMemo(() => Math.floor(offset / limit) + 1, [offset])
  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total])

  return (
    <section>
      <h1 style={{ marginTop: 0 }}>Alerts</h1>

      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 12 }}>
        <label style={labelStyle}>
          Event Type
          <input style={inputStyle} value={eventTypeFilter} onChange={(e) => { setOffset(0); setEventTypeFilter(e.target.value) }} placeholder="issues / pull_request" />
        </label>

        <label style={labelStyle}>
          Action
          <input style={inputStyle} value={actionFilter} onChange={(e) => { setOffset(0); setActionFilter(e.target.value) }} placeholder="opened / edited / closed" />
        </label>

        <label style={labelStyle}>
          Suggestion Type
          <input style={inputStyle} value={suggestionTypeFilter} onChange={(e) => { setOffset(0); setSuggestionTypeFilter(e.target.value) }} placeholder="label / comment" />
        </label>

        <button style={buttonStyle} onClick={() => setOffset(0)} aria-label="应用筛选">Apply Filters</button>
      </div>

      {alerts.length === 0 ? (
        <p style={{ color: '#475569' }}>no alerts matched current filters</p>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 1100, background: '#FFFFFF' }}>
            <thead>
              <tr>
                <th style={thStyle}>Delivery</th>
                <th style={thStyle}>Type</th>
                <th style={thStyle}>Action</th>
                <th style={thStyle}>Suggestion Type</th>
                <th style={thStyle}>Suggestion Value</th>
                <th style={thStyle}>Rule Matched</th>
                <th style={thStyle}>Reason</th>
                <th style={thStyle}>Repository</th>
                <th style={thStyle}>Sender</th>
                <th style={thStyle}>Created At</th>
              </tr>
            </thead>
            <tbody>
              {alerts.map((item) => (
                <tr key={`${item.delivery_id}-${item.suggestion_type}-${item.suggestion_value}-${item.rule_matched}`}>
                  <td style={tdStyle}>{item.delivery_id}</td>
                  <td style={tdStyle}>{item.event_type}</td>
                  <td style={tdStyle}>{item.action || '-'}</td>
                  <td style={tdStyle}>{item.suggestion_type}</td>
                  <td style={tdStyle}>{item.suggestion_value}</td>
                  <td style={tdStyle}>{item.rule_matched}</td>
                  <td style={tdStyle}>{item.reason}</td>
                  <td style={tdStyle}>{item.repository_full_name}</td>
                  <td style={tdStyle}>{item.sender_login}</td>
                  <td style={tdStyle}>{new Date(item.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div style={{ display: 'flex', gap: 8, marginTop: 12, alignItems: 'center' }}>
        <button style={buttonStyle} onClick={() => setOffset((v) => Math.max(0, v - limit))} disabled={offset === 0} aria-label="上一页">Prev</button>
        <button style={buttonStyle} onClick={() => setOffset((v) => v + limit)} disabled={currentPage >= totalPages} aria-label="下一页">Next</button>
        <span style={{ color: '#475569' }}>page: {currentPage} / {totalPages} · total: {total}</span>
      </div>

      {error ? <p style={{ color: '#EF4444', marginTop: 16 }}>error: {error}</p> : null}
    </section>
  )
}

const thStyle: CSSProperties = { textAlign: 'left', padding: '10px 8px', borderBottom: '1px solid #E2E8F0', color: '#334155', fontWeight: 600 }
const tdStyle: CSSProperties = { padding: '10px 8px', borderBottom: '1px solid #E2E8F0', color: '#0F172A', fontSize: 14 }
const labelStyle: CSSProperties = { display: 'flex', flexDirection: 'column', gap: 6, color: '#334155', fontSize: 14 }
const inputStyle: CSSProperties = { minHeight: 40, minWidth: 220, border: '1px solid #E2E8F0', borderRadius: 8, padding: '8px 10px', color: '#0F172A' }
const buttonStyle: CSSProperties = { minHeight: 40, padding: '0 14px', border: '1px solid #CBD5E1', borderRadius: 8, background: '#FFFFFF', color: '#0F172A', cursor: 'pointer', transition: 'background-color 0.2s ease' }
