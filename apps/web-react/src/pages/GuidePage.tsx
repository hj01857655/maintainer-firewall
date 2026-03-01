import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export function GuidePage() {
  const { t } = useTranslation()

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t('guide.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">{t('guide.subtitle')}</p>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl">
        <h2 className="m-0 text-lg font-semibold text-slate-900 dark:text-slate-100">{t('guide.quickStart.title')}</h2>
        <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm leading-relaxed text-slate-700 dark:text-slate-300">
          <li>{t('guide.quickStart.step1')}</li>
          <li>{t('guide.quickStart.step2')}</li>
          <li>{t('guide.quickStart.step3')}</li>
          <li>{t('guide.quickStart.step4')}</li>
          <li>{t('guide.quickStart.step5')}</li>
        </ol>
      </div>

      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        <GuideCard title={t('guide.cards.rules.title')} desc={t('guide.cards.rules.desc')} to="/rules" />
        <GuideCard title={t('guide.cards.events.title')} desc={t('guide.cards.events.desc')} to="/events" />
        <GuideCard title={t('guide.cards.alerts.title')} desc={t('guide.cards.alerts.desc')} to="/alerts" />
        <GuideCard title={t('guide.cards.failures.title')} desc={t('guide.cards.failures.desc')} to="/failures" />
      </div>
    </section>
  )
}

function GuideCard({ title, desc, to }: { title: string; desc: string; to: string }) {
  const getIcon = (path: string) => {
    const iconMap: Record<string, JSX.Element> = {
      '/rules': (
        <svg className="h-6 w-6 text-blue-500 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
        </svg>
      ),
      '/events': (
        <svg className="h-6 w-6 text-green-500 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
      '/alerts': (
        <svg className="h-6 w-6 text-orange-500 dark:text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
      ),
      '/failures': (
        <svg className="h-6 w-6 text-red-500 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
    }
    return iconMap[path] || (
      <svg className="h-6 w-6 text-slate-400 dark:text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
    )
  }

  return (
    <Link
      to={to}
      className="cursor-pointer rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition-colors duration-200 hover:border-blue-300 hover:bg-blue-50/40 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-xl dark:hover:border-blue-500/40 dark:hover:bg-blue-500/10"
    >
      <div className="flex items-start space-x-3">
        <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-slate-100 dark:bg-slate-800">
          {getIcon(to)}
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="m-0 text-base font-semibold text-slate-900 dark:text-slate-100">{title}</h3>
          <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">{desc}</p>
        </div>
      </div>
    </Link>
  )
}
