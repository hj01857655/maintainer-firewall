import type { ReactNode } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { clearAccessToken } from '../auth'
import { useTheme } from '../theme'

type Props = {
  children: ReactNode
}

export function AppLayout({ children }: Props) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { mode, setMode } = useTheme()

  function onLogout() {
    clearAccessToken()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 dark:bg-slate-900 dark:text-slate-100 transition-colors duration-200">
      <div className="mx-auto grid min-h-screen max-w-[1440px] grid-cols-1 lg:grid-cols-[280px_1fr]">
        <aside className="border-b border-slate-200 bg-white/95 px-4 py-4 backdrop-blur dark:border-slate-700 dark:bg-slate-900/95 lg:sticky lg:top-0 lg:h-screen lg:border-b-0 lg:border-r lg:px-5 lg:py-6 transition-colors duration-200">
          <div className="flex items-center justify-between gap-3 lg:block">
            <h2 className="m-0 text-lg font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t('brand')}</h2>
            <div className="mt-3 grid grid-cols-1 gap-2 lg:mt-4">
              <label className="inline-flex items-center justify-between gap-2 text-xs text-slate-600 dark:text-slate-300 lg:text-sm">
                <span>{t('lang.label')}</span>
                <select
                  className="h-9 min-w-[110px] cursor-pointer rounded-lg border border-slate-300 bg-white px-2 text-slate-700 outline-none transition-colors duration-200 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 focus:ring-2 focus:ring-blue-500/30"
                  value={i18n.resolvedLanguage === 'en' ? 'en' : 'zh-CN'}
                  onChange={(e) => void i18n.changeLanguage(e.target.value)}
                >
                  <option value="zh-CN">{t('lang.zhCN')}</option>
                  <option value="en">{t('lang.en')}</option>
                </select>
              </label>
              <label className="inline-flex items-center justify-between gap-2 text-xs text-slate-600 dark:text-slate-300 lg:text-sm">
                <span>{t('theme.label')}</span>
                <select
                  className="h-9 min-w-[110px] cursor-pointer rounded-lg border border-slate-300 bg-white px-2 text-slate-700 outline-none transition-colors duration-200 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 focus:ring-2 focus:ring-blue-500/30"
                  value={mode}
                  onChange={(e) => setMode(e.target.value as 'light' | 'dark' | 'system')}
                  aria-label={t('theme.label')}
                >
                  <option value="light">{t('theme.light')}</option>
                  <option value="dark">{t('theme.dark')}</option>
                  <option value="system">{t('theme.system')}</option>
                </select>
              </label>
              <button
                type="button"
                onClick={onLogout}
                className="inline-flex h-9 min-w-[80px] cursor-pointer items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-xs font-medium text-slate-700 transition-colors duration-200 hover:border-red-200 hover:bg-red-50 hover:text-red-700 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:hover:border-red-400/40 dark:hover:bg-red-500/10 dark:hover:text-red-300 focus:outline-none focus:ring-2 focus:ring-red-500/30 lg:text-sm"
              >
                {t('nav.logout')}
              </button>
            </div>
          </div>

          <nav className="mt-4 grid grid-cols-2 gap-2 lg:mt-6 lg:grid-cols-1" aria-label={t('nav.main')}>
            <NavItem to="/dashboard" label={t('nav.dashboard')} />
            <NavItem to="/events" label={t('nav.events')} />
            <NavItem to="/rules" label={t('nav.rules')} />
            <NavItem to="/alerts" label={t('nav.alerts')} />
            <NavItem to="/failures" label={t('nav.failures')} />
            <NavItem to="/audit" label={t('nav.audit')} />
            <NavItem to="/system-config" label={t('nav.systemConfig')} />
            <NavItem to="/guide" label={t('nav.guide')} />
          </nav>
        </aside>

        <main className="max-w-full overflow-x-hidden px-4 py-5 md:px-6 md:py-6 lg:px-8">{children}</main>
      </div>
    </div>
  )
}

function NavItem({ to, label }: { to: string; label: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        [
          'cursor-pointer rounded-xl border px-3 py-2.5 text-center text-sm font-medium transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900 lg:text-left',
          isActive
            ? 'border-blue-200 bg-blue-50 text-blue-700 shadow-sm dark:border-blue-500/40 dark:bg-blue-500/15 dark:text-blue-200'
            : 'border-slate-200 bg-white text-slate-700 hover:border-blue-200 hover:bg-blue-50/60 hover:text-blue-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-blue-500/40 dark:hover:bg-blue-500/10 dark:hover:text-blue-200',
        ].join(' ')
      }
    >
      {label}
    </NavLink>
  )
}
