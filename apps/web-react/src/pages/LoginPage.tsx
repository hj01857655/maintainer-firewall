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
      const resp = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      if (!resp.ok) {
        const body = await resp.text()
        throw new Error(`login HTTP ${resp.status} ${body}`.trim())
      }

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
    <section className="grid min-h-screen place-items-center bg-gradient-to-b from-slate-50 to-blue-50 px-4 py-10">
      <form
        onSubmit={onSubmit}
        className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-6 shadow-sm md:p-8"
      >
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900">登录控制台</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600">使用管理员账号登录 Maintainer Firewall。</p>

        <div className="mt-6 space-y-4">
          <label className="block text-sm font-medium text-slate-700">
            <span>用户名</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              placeholder="请输入用户名"
              required
            />
          </label>

          <label className="block text-sm font-medium text-slate-700">
            <span>密码</span>
            <input
              className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              placeholder="请输入密码"
              required
            />
          </label>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="mt-6 inline-flex h-11 w-full cursor-pointer items-center justify-center rounded-xl bg-blue-600 px-4 text-sm font-semibold text-white transition-colors duration-200 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:bg-blue-300"
          aria-label="登录"
        >
          {loading ? (
            <span className="inline-flex items-center gap-2">
              <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white" />
              登录中...
            </span>
          ) : (
            '登录'
          )}
        </button>

        {error ? (
          <p className="mt-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600">登录失败：{error}</p>
        ) : null}
      </form>
    </section>
  )
}
