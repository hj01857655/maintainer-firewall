import type { CSSProperties } from 'react'
import { useEffect, useState } from 'react'
import { authHeaders } from '../auth'

type Health = {
  status: string
  service: string
}

export function DashboardPage() {
  const [health, setHealth] = useState<Health | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    fetch('/health', { headers: authHeaders() })
      .then((r) => {

        return r.json()
      })
      .then((data: Health) => setHealth(data))
      .catch((e: Error) => setError(e.message))
  }, [])

  return (
    <section>
      <h1 style={{ marginTop: 0 }}>Dashboard</h1>
      <p style={{ color: '#475569' }}>Service overview and quick status.</p>

      <div style={cardStyle}>
        <h3 style={{ marginTop: 0 }}>API Health</h3>
        {health ? <pre style={{ margin: 0 }}>{JSON.stringify(health, null, 2)}</pre> : <p>{error || 'Loading...'}</p>}
      </div>
    </section>
  )
}

const cardStyle: CSSProperties = {
  background: '#FFFFFF',
  border: '1px solid #E2E8F0',
  borderRadius: 10,
  padding: 16,
}
