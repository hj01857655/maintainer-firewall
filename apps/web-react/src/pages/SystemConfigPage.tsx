import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { apiFetch } from '../api'

type ConfigViewResponse = {
  ok: boolean
  database_url: string
  admin_username: string
  admin_password_masked: string
  jwt_secret_masked: string
  github_webhook_secret_masked: string
  github_token_masked: string
}

type ConfigUpdatePayload = {
  database_url: string
  admin_username: string
  admin_password: string
  jwt_secret: string
  github_webhook_secret: string
  github_token: string
}

export function SystemConfigPage() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')

  const [databaseURL, setDatabaseURL] = useState('')
  const [adminUsername, setAdminUsername] = useState('')
  const [adminPassword, setAdminPassword] = useState('')
  const [jwtSecret, setJwtSecret] = useState('')
  const [webhookSecret, setWebhookSecret] = useState('')
  const [githubToken, setGithubToken] = useState('')

  const [adminPasswordMasked, setAdminPasswordMasked] = useState('')
  const [jwtSecretMasked, setJwtSecretMasked] = useState('')
  const [webhookSecretMasked, setWebhookSecretMasked] = useState('')
  const [githubTokenMasked, setGithubTokenMasked] = useState('')

  useEffect(() => {
    apiFetch('/api/config-view')
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text())
        return r.json() as Promise<ConfigViewResponse>
      })
      .then((data) => {
        if (!data.ok) throw new Error('config view response not ok')
        setDatabaseURL(data.database_url || '')
        setAdminUsername(data.admin_username || '')
        setAdminPasswordMasked(data.admin_password_masked || '')
        setJwtSecretMasked(data.jwt_secret_masked || '')
        setWebhookSecretMasked(data.github_webhook_secret_masked || '')
        setGithubTokenMasked(data.github_token_masked || '')
        setError('')
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  async function onSave(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError('')
    setMessage('')
    try {
      const payload: ConfigUpdatePayload = {
        database_url: databaseURL.trim(),
        admin_username: adminUsername.trim(),
        admin_password: adminPassword.trim(),
        jwt_secret: jwtSecret.trim(),
        github_webhook_secret: webhookSecret.trim(),
        github_token: githubToken.trim(),
      }
      const resp = await apiFetch('/api/config-update', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!resp.ok) throw new Error(await resp.text())
      const data: { ok: boolean; message?: string; restart_required?: boolean } = await resp.json()
      if (!data.ok) throw new Error(data.message || 'save config failed')
      setAdminPassword('')
      setJwtSecret('')
      setWebhookSecret('')
      setGithubToken('')
      setMessage(data.restart_required ? t('config.restartRequired') : t('config.saved'))
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'save config failed'
      setError(msg)
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t('config.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">{t('config.subtitle')}</p>
      </div>

      <form onSubmit={onSave} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        {loading ? <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p> : null}

        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <Field label="DATABASE_URL" value={databaseURL} onChange={setDatabaseURL} placeholder="mysql://..." required />
          <Field label="ADMIN_USERNAME" value={adminUsername} onChange={setAdminUsername} placeholder="admin" required />
          <SecretField label="ADMIN_PASSWORD" value={adminPassword} onChange={setAdminPassword} masked={adminPasswordMasked} />
          <SecretField label="JWT_SECRET" value={jwtSecret} onChange={setJwtSecret} masked={jwtSecretMasked} />
          <SecretField label="GITHUB_WEBHOOK_SECRET" value={webhookSecret} onChange={setWebhookSecret} masked={webhookSecretMasked} />
          <SecretField label="GITHUB_TOKEN" value={githubToken} onChange={setGithubToken} masked={githubTokenMasked} />
        </div>

        <button
          type="submit"
          disabled={saving}
          className="mt-4 h-11 min-w-[128px] cursor-pointer rounded-xl bg-blue-600 px-4 text-sm font-semibold text-white transition-colors duration-200 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saving ? t('config.saving') : t('config.save')}
        </button>
      </form>

      {message ? <p className="rounded-lg border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700 dark:border-green-500/40 dark:bg-green-500/10 dark:text-green-200">{message}</p> : null}
      {error ? <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600 dark:border-red-500/40 dark:bg-red-500/10 dark:text-red-300">{t('common.error', { message: error })}</p> : null}
    </section>
  )
}

type FieldProps = {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder: string
  required?: boolean
}

function Field({ label, value, onChange, placeholder, required }: FieldProps) {
  return (
    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
      <span>{label}</span>
      <input
        className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        required={required}
      />
    </label>
  )
}

type SecretFieldProps = {
  label: string
  value: string
  onChange: (v: string) => void
  masked: string
}

function SecretField({ label, value, onChange, masked }: SecretFieldProps) {
  return (
    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
      <span>{label}</span>
      <input
        type="password"
        className="mt-2 h-11 w-full rounded-xl border border-slate-300 px-3 text-base text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={masked || '******'}
      />
    </label>
  )
}
