import type { CSSProperties } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { authHeaders } from '../auth'

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
  limit: number
  offset: number
  total: number
  event_type?: string
  action?: string
}

export function EventsPage() {
  const [events, setEvents] = useState<EventItem[]>([])
  const [error, setError] = useState<string>('')
  const [eventTypeFilter, setEventTypeFilter] = useState<string>('')
  const [actionFilter, setActionFilter] = useState<string>('')
  const [offset, setOffset] = useState<number>(0)
  const [total, setTotal] = useState<number>(0)
  const limit = 20

  useEffect(() => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (eventTypeFilter) params.set('event_type', eventTypeFilter)
    if (actionFilter) params.set('action', actionFilter)

    fetch(`/events?${params.toString()}`, { headers: authHeaders() })
      .then((r) => {

        return r.json()
      })
      .then((data: EventsResponse) => {
        if (!data.ok) throw new Error('events response not ok')
        setEvents(data.items)
        setTotal(data.total)
      })
      .catch((e: Error) => setError((prev) => (prev ? `${prev}; ${e.message}` : e.message)))
  }, [eventTypeFilter, actionFilter, offset])

  const currentPage = useMemo(() => Math.floor(offset / limit) + 1, [offset])
  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / limit)), [total])

  return (
    <section>
      <h1 style={{ marginTop: 0 }}>Events</h1>

      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 12 }}>
        <label style={labelStyle}>
          Event Type
          <input style={inputStyle} value={eventTypeFilter} onChange={(e) => { setOffset(0); setEventTypeFilter(e.target.value) }} placeholder="issues / pull_request" />
        </label>

        <label style={labelStyle}>
          Action
          <input style={inputStyle} value={actionFilter} onChange={(e) => { setOffset(0); setActionFilter(e.target.value) }} placeholder="opened / edited / closed" />
        </label>

        <button style={buttonStyle} onClick={() => setOffset(0)} aria-label="应用筛选">Apply Filters</button>
      </div>

      {events.length === 0 ? (
        <p style={{ color: '#475569' }}>no events matched current filters</p>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 760, background: '#FFFFFF' }}>
            <thead>
              <tr>
                <th style={thStyle}>ID</th>
                <th style={thStyle}>Type</th>
                <th style={thStyle}>Action</th>
                <th style={thStyle}>Repository</th>
                <th style={thStyle}>Sender</th>
                <th style={thStyle}>Received At</th>
              </tr>
            </thead>
            <tbody>
              {events.map((evt) => (
                <tr key={evt.id}>
                  <td style={tdStyle}>{evt.id}</td>
                  <td style={tdStyle}>{evt.event_type}</td>
                  <td style={tdStyle}>{evt.action || '-'}</td>
                  <td style={tdStyle}>{evt.repository_full_name}</td>
                  <td style={tdStyle}>{evt.sender_login}</td>
                  <td style={tdStyle}>{new Date(evt.received_at).toLocaleString()}</td>
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
