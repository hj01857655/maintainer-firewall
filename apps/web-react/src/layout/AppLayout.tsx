import { type ReactNode, useEffect, useRef, useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { clearAccessToken } from '../auth'
import { useTheme, type ThemeMode } from '../theme'

type Props = {
  children: ReactNode
}

export function AppLayout({ children }: Props) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { mode, setMode } = useTheme()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!menuOpen) return
    const onMouseDown = (ev: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(ev.target as Node)) {
        setMenuOpen(false)
      }
    }
    const onKeyDown = (ev: KeyboardEvent) => {
      if (ev.key === 'Escape') setMenuOpen(false)
    }
    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [menuOpen])

  function onToggleLanguage() {
    const current = i18n.resolvedLanguage === 'en' ? 'en' : 'zh-CN'
    const next = current === 'en' ? 'zh-CN' : 'en'
    void i18n.changeLanguage(next)
  }

  function onCycleTheme() {
    setMode(mode === 'light' ? 'dark' : 'light')
  }

  function onGoSystemConfig() {
    setMenuOpen(false)
    navigate('/system-config')
  }

  function onLogout() {
    setMenuOpen(false)
    if (!window.confirm(t('nav.logoutConfirm'))) return
    clearAccessToken()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100 transition-colors duration-200">
      <div className="mx-auto grid min-h-screen max-w-[1600px] grid-cols-1 lg:grid-cols-[320px_1fr]">
        <aside className="border-r border-slate-200/60 bg-white/90 backdrop-blur-xl px-6 py-6 shadow-xl dark:border-slate-700/50 dark:bg-gradient-to-b dark:from-slate-800 dark:to-slate-900 dark:shadow-2xl lg:sticky lg:top-0 lg:h-screen lg:border-b-0 lg:shadow-2xl transition-all duration-300">
          <div className="flex items-center space-x-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 text-white shadow-lg shadow-blue-500/30">
              <svg className="h-6 w-6" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <h2 className="m-0 text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">{t('brand')}</h2>
          </div>

          <nav className="mt-8 space-y-2" aria-label={t('nav.main')}>
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

        <main className="max-w-full overflow-x-hidden px-6 py-6 md:px-8 md:py-8 lg:px-8">
          <header className="mb-4 flex flex-wrap items-center justify-end gap-2 rounded-2xl border border-slate-200 bg-white/95 p-3 shadow-sm transition-colors duration-200 dark:border-slate-700 dark:bg-slate-900/95">
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={onToggleLanguage}
                className="inline-flex h-11 w-11 cursor-pointer items-center justify-center rounded-xl border border-slate-300 bg-white text-slate-700 transition-colors duration-200 hover:bg-slate-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/30 focus:ring-offset-2 dark:focus:ring-offset-slate-900"
                aria-label={t('lang.label')}
                title={`${t('lang.label')}: ${i18n.resolvedLanguage === 'en' ? t('lang.en') : t('lang.zhCN')}`}
              >
                <LanguageIcon />
              </button>

              <button
                type="button"
                onClick={onCycleTheme}
                className="inline-flex h-11 w-11 cursor-pointer items-center justify-center rounded-xl border border-slate-300 bg-white text-slate-700 transition-colors duration-200 hover:bg-slate-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/30 focus:ring-offset-2 dark:focus:ring-offset-slate-900"
                aria-label={t('theme.label')}
                title={`${t('theme.label')}: ${mode === 'dark' ? t('theme.dark') : t('theme.light')}`}
              >
                <ThemeIcon mode={mode} />
              </button>

              <div ref={menuRef} className="relative">
                <button
                  type="button"
                  onClick={() => setMenuOpen((v) => !v)}
                  className="inline-flex h-11 w-11 cursor-pointer items-center justify-center rounded-xl border border-slate-300 bg-white text-slate-700 transition-colors duration-200 hover:bg-slate-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/30 focus:ring-offset-2 dark:focus:ring-offset-slate-900"
                  aria-label={t('nav.accountMenu')}
                  aria-haspopup="menu"
                  aria-expanded={menuOpen}
                  title={t('nav.accountMenu')}
                >
                  <UserIcon />
                </button>

              {menuOpen ? (
                <div className="absolute right-0 z-20 mt-2 w-44 overflow-hidden rounded-xl border border-slate-200 bg-white/95 p-1 shadow-lg dark:border-slate-700 dark:bg-slate-900/95">
                  <button
                    type="button"
                    onClick={onGoSystemConfig}
                    className="flex h-11 w-full cursor-pointer items-center rounded-lg px-3 text-left text-sm text-slate-700 transition-colors duration-200 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-800"
                    role="menuitem"
                  >
<svg className="mr-3 h-4 w-4" fill="none" stroke="currentColor" strokeWidth="1.8" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                      <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                    {t('nav.systemConfig')}
                  </button>
                  <div className="my-1 h-px bg-slate-200 dark:bg-slate-700" />
                  <button
                    type="button"
                    onClick={onLogout}
                    className="flex h-11 w-full cursor-pointer items-center rounded-lg px-3 text-left text-sm text-red-600 transition-colors duration-200 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-500/10"
                    role="menuitem"
                  >
                    <svg className="mr-3 h-4 w-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                    </svg>
                    {t('nav.logout')}
                  </button>
                </div>
              ) : null}
              </div>
            </div>
          </header>
          {children}
        </main>
      </div>
    </div>
  )
}

function NavItem({ to, label }: { to: string; label: string }) {
  const getIcon = (path: string) => {
    const iconMap: Record<string, JSX.Element> = {
      '/dashboard': (
        <svg className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
        </svg>
      ),
      '/events': (
        <svg className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
      '/rules': (
        <svg className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
        </svg>
      ),
      '/alerts': (
        <svg className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
      ),
      '/failures': (
        <svg className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
      '/audit': (
        <svg className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      ),
      '/system-config': (
        <svg className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      ),
      '/guide': (
        <svg className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
        </svg>
      ),
    }
    return iconMap[path] || null
  }

  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        [
          'group flex items-center space-x-3 rounded-xl border px-4 py-3 text-sm font-medium transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500/30 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
          isActive
            ? 'border-blue-200/60 bg-gradient-to-r from-blue-50 to-blue-100/50 text-blue-700 shadow-lg shadow-blue-500/10 dark:border-blue-400/60 dark:from-blue-500/20 dark:to-blue-600/20 dark:text-blue-100 dark:shadow-blue-500/30'
            : 'border-slate-200/60 bg-white/60 text-slate-600 hover:border-blue-200/60 hover:bg-gradient-to-r hover:from-blue-50/50 hover:to-blue-100/30 hover:text-blue-700 hover:shadow-md hover:shadow-blue-500/10 dark:border-slate-700/60 dark:bg-slate-800/90 dark:text-slate-300 dark:hover:border-blue-500/40 dark:hover:from-blue-500/10 dark:hover:to-blue-500/5 dark:hover:text-blue-200 dark:hover:shadow-blue-500/10',
        ].join(' ')
      }
    >
      <span>{getIcon(to)}</span>
      <span>{label}</span>
    </NavLink>
  )
}

function LanguageIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-6 w-6" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M3 5h12M9 5v1a12 12 0 0 1-6 10M4 15h10M7 13c1.2 2 3.2 4.4 5 6M17 6h4m-2-2v4M14 19h8m-7-3h6" />
    </svg>
  )
}

function ThemeIcon({ mode }: { mode: ThemeMode }) {
  if (mode === 'dark') {
    return (
      <svg viewBox="0 0 24 24" className="h-6 w-6" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" d="M21 12.8A9 9 0 1 1 11.2 3 7 7 0 0 0 21 12.8Z" />
      </svg>
    )
  }

  return (
    <svg viewBox="0 0 24 24" className="h-6 w-6" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <circle cx="12" cy="12" r="4" />
      <path strokeLinecap="round" d="M12 3v2M12 19v2M3 12h2M19 12h2M5.6 5.6l1.4 1.4M17 17l1.4 1.4M18.4 5.6 17 7M7 17l-1.4 1.4" />
    </svg>
  )
}

function UserIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-6 w-6" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <circle cx="12" cy="8" r="3.5" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M5 19a7 7 0 0 1 14 0" />
    </svg>
  )
}
