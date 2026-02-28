import type { CSSProperties } from 'react'
import { useEffect, useState } from 'react'

type Health = {
  status: string
  service: string
}

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
  event_type?: string
  action?: string
}

export function App() {
  const [health, setHealth] = useState<Health | null>(null)
  const [events, setEvents] = useState<EventItem[]>([])
  const [error, setError] = useState<string>('')
  const [eventTypeFilter, setEventTypeFilter] = useState<string>('')
  const [actionFilter, setActionFilter] = useState<string>('')
  const [offset, setOffset] = useState<number>(0)
  const limit = 20

  useEffect(() => {
    fetch('/health')
      .then((r) => {
        if (!r.ok) throw new Error(`health HTTP ${r.status}`)
        return r.json()
      })
      .then((data: Health) => setHealth(data))
      .catch((e: Error) => setError(e.message))
  }, [])

  useEffect(() => {
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String(offset),
    })
    if (eventTypeFilter) params.set('event_type', eventTypeFilter)
    if (actionFilter) params.set('action', actionFilter)

    fetch(`/events?${params.toString()}`)
      .then((r) => {
        if (!r.ok) throw new Error(`events HTTP ${r.status}`)
        return r.json()
      })
      .then((data: EventsResponse) => {
        if (!data.ok) throw new Error('events response not ok')
        setEvents(data.items)
      })
      .catch((e: Error) => setError((prev) => (prev ? `${prev}; ${e.message}` : e.message)))
  }, [eventTypeFilter, actionFilter, offset])

  return (
    <main
      style={{
        padding: 24,
        fontFamily: 'Inter, system-ui, sans-serif',
        maxWidth: '100%',
        overflowX: 'hidden',
      }}
    >
      <h1 style={{ color: '#0F172A' }}>Maintainer Firewall</h1>
      <p style={{ color: '#475569' }}>Go API + React Console</p>

      <section style={{ marginTop: 16 }}>
        <h2 style={{ color: '#0F172A', fontSize: 20 }}>Health</h2>
        {health ? (
          <pre style={{ background: '#F8FAFC', border: '1px solid #E2E8F0', padding: 12 }}>
            {JSON.stringify(health, null, 2)}
          </pre>
        ) : (
          <p>loading health...</p>
        )}
      </section>

      <section style={{ marginTop: 24 }}>
        <h2 style={{ color: '#0F172A', fontSize: 20 }}>Latest Events</h2>

        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 12 }}>
          <label style={labelStyle}>
            Event Type
            <input
              style={inputStyle}
              value={eventTypeFilter}
              onChange={(e) => {
                setOffset(0)
                setEventTypeFilter(e.target.value)
              }}
              placeholder="issues / pull_request"
            />
          </label>

          <label style={labelStyle}>
            Action
            <input
              style={inputStyle}
              value={actionFilter}
              onChange={(e) => {
                setOffset(0)
                setActionFilter(e.target.value)
              }}
              placeholder="opened / edited / closed"
            />
          </label>

          <button
            style={buttonStyle}
            onClick={() => setOffset(0)}
            aria-label="应用筛选"
          >
            Apply Filters
          </button>
        </div>

        {events.length === 0 ? (
          <p style={{ color: '#475569' }}>no events matched current filters</p>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 760 }}>
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

        <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
          <button
            style={buttonStyle}
            onClick={() => setOffset((v) => Math.max(0, v - limit))}
            disabled={offset === 0}
            aria-label="上一页"
          >
            Prev
          </button>
          <button
            style={buttonStyle}
            onClick={() => setOffset((v) => v + limit)}
            disabled={events.length < limit}
            aria-label="下一页"
          >
            Next
          </button>
          <span style={{ color: '#475569', alignSelf: 'center' }}>offset: {offset}</span>
        </div>
      </section>

      {error ? <p style={{ color: '#EF4444', marginTop: 16 }}>error: {error}</p> : null}
    </main>
  )
}

const thStyle: CSSProperties = {
  textAlign: 'left',
  padding: '10px 8px',
  borderBottom: '1px solid #E2E8F0',
  color: '#334155',
  fontWeight: 600,
}

const tdStyle: CSSProperties = {
  padding: '10px 8px',
  borderBottom: '1px solid #E2E8F0',
  color: '#0F172A',
  fontSize: 14,
}

const labelStyle: CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 6,
  color: '#334155',
  fontSize: 14,
}

const inputStyle: CSSProperties = {
  minHeight: 40,
  minWidth: 220,
  border: '1px solid #E2E8F0',
  borderRadius: 8,
  padding: '8px 10px',
  color: '#0F172A',
}

const buttonStyle: CSSProperties = {
  minHeight: 40,
  padding: '0 14px',
  border: '1px solid #CBD5E1',
  borderRadius: 8,
  background: '#FFFFFF',
  color: '#0F172A',
  cursor: 'pointer',
  transition: 'background-color 0.2s ease',
}
