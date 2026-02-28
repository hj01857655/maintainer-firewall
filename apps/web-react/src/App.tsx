import { useEffect, useState } from 'react'

type Health = {
  status: string
  service: string
}

export function App() {
  const [health, setHealth] = useState<Health | null>(null)
  const [error, setError] = useState<string>('')

  useEffect(() => {
    fetch('/health')
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then((data: Health) => setHealth(data))
      .catch((e: Error) => setError(e.message))
  }, [])

  return (
    <main style={{ padding: 24, fontFamily: 'Inter, system-ui, sans-serif' }}>
      <h1>Maintainer Firewall</h1>
      <p>Go API + React Console</p>
      {health ? (
        <pre>{JSON.stringify(health, null, 2)}</pre>
      ) : error ? (
        <p style={{ color: 'crimson' }}>health check failed: {error}</p>
      ) : (
        <p>loading health...</p>
      )}
    </main>
  )
}
