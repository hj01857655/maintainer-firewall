import type { CSSProperties } from 'react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { setAccessToken } from '../auth'

export function LoginPage() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const resp = await fetch('/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      if (!resp.ok) throw new Error(`login HTTP ${resp.status}`)
      const data: { ok: boolean; token?: string; message?: string } = await resp.json()
      if (!data.ok || !data.token) throw new Error(data.message || 'login failed')
      setAccessToken(data.token)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'login failed'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <section style={wrapStyle}>
      <form onSubmit={onSubmit} style={cardStyle}>
        <h1 style={{ marginTop: 0 }}>Login</h1>
        <p style={{ color: '#475569', marginTop: 0 }}>Sign in to access maintainer console.</p>

        <label style={labelStyle}>
          Username
          <input
            style={inputStyle}
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            required
          />
        </label>

        <label style={labelStyle}>
          Password
          <input
            style={inputStyle}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
          />
        </label>

        <button type="submit" style={buttonStyle} disabled={loading} aria-label="登录">
          {loading ? 'Signing in...' : 'Sign in'}
        </button>

        {error ? <p style={{ color: '#EF4444', marginBottom: 0 }}>error: {error}</p> : null}
      </form>
    </section>
  )
}

const wrapStyle: CSSProperties = {
  minHeight: '100vh',
  display: 'grid',
  placeItems: 'center',
  background: '#F8FAFC',
  padding: 16,
}

const cardStyle: CSSProperties = {
  width: '100%',
  maxWidth: 420,
  background: '#FFFFFF',
  border: '1px solid #E2E8F0',
  borderRadius: 12,
  padding: 20,
  display: 'flex',
  flexDirection: 'column',
  gap: 12,
}

const labelStyle: CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 6,
  color: '#334155',
  fontSize: 14,
}

const inputStyle: CSSProperties = {
  minHeight: 44,
  border: '1px solid #E2E8F0',
  borderRadius: 8,
  padding: '8px 10px',
  color: '#0F172A',
}

const buttonStyle: CSSProperties = {
  minHeight: 44,
  border: '1px solid #CBD5E1',
  borderRadius: 8,
  background: '#FFFFFF',
  color: '#0F172A',
  cursor: 'pointer',
  transition: 'background-color 0.2s ease',
}
