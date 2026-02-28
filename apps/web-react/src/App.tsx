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
}

export function App() {
  const [health, setHealth] = useState<Health | null>(null)
  const [events, setEvents] = useState<EventItem[]>([])
  const [error, setError] = useState<string>('')

  useEffect(() => {
    fetch('/health')
      .then((r) => {
        if (!r.ok) throw new Error(`health HTTP ${r.status}`)
        return r.json()
      })
      .then((data: Health) => setHealth(data))
      .catch((e: Error) => setError(e.message))

    fetch('/events?limit=20&offset=0')
      .then((r) => {
        if (!r.ok) throw new Error(`events HTTP ${r.status}`)
        return r.json()
      })
      .then((data: EventsResponse) => {
        if (!data.ok) throw new Error('events response not ok')
        setEvents(data.items)
      })
      .catch((e: Error) => setError((prev) => (prev ? `${prev}; ${e.message}` : e.message)))
  }, [])

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
        {events.length === 0 ? (
          <p style={{ color: '#475569' }}>no events yet</p>
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
