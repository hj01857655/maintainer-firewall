import type { ReactNode } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { clearAccessToken } from '../auth'

type Props = {
  children: ReactNode
}

export function AppLayout({ children }: Props) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()

  function onLogout() {
    clearAccessToken()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      <div className="mx-auto grid min-h-screen max-w-[1440px] grid-cols-1 lg:grid-cols-[280px_1fr]">
        <aside className="border-b border-slate-200 bg-white/95 px-4 py-4 backdrop-blur lg:sticky lg:top-0 lg:h-screen lg:border-b-0 lg:border-r lg:px-5 lg:py-6">
          <div className="flex items-center justify-between gap-3 lg:block">
            <h2 className="m-0 text-lg font-semibold tracking-tight text-slate-900">{t('brand')}</h2>
            <div className="inline-flex items-center gap-2 lg:mt-4 lg:w-full lg:justify-between">
              <label className="inline-flex items-center gap-2 text-xs text-slate-600 lg:text-sm">
                <span>{t('lang.label')}</span>
                <select
                  className="h-9 min-w-[96px] cursor-pointer rounded-lg border border-slate-300 bg-white px-2 text-slate-700 outline-none transition-colors duration-200 focus:ring-2 focus:ring-blue-500/30"
                  value={i18n.resolvedLanguage === 'en' ? 'en' : 'zh-CN'}
                  onChange={(e) => void i18n.changeLanguage(e.target.value)}
                >
                  <option value="zh-CN">{t('lang.zhCN')}</option>
                  <option value="en">{t('lang.en')}</option>
                </select>
              </label>
              <button
                type="button"
                onClick={onLogout}
                className="inline-flex h-9 min-w-[80px] cursor-pointer items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-xs font-medium text-slate-700 transition-colors duration-200 hover:border-red-200 hover:bg-red-50 hover:text-red-700 focus:outline-none focus:ring-2 focus:ring-red-500/30 lg:text-sm"
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
          'cursor-pointer rounded-xl border px-3 py-2.5 text-center text-sm font-medium transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 lg:text-left',
          isActive
            ? 'border-blue-200 bg-blue-50 text-blue-700 shadow-sm'
            : 'border-slate-200 bg-white text-slate-700 hover:border-blue-200 hover:bg-blue-50/60 hover:text-blue-700',
        ].join(' ')
      }
    >
      {label}
    </NavLink>
  )
}
