import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export function GuidePage() {
  const { t } = useTranslation()

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6">
        <h1 className="m-0 text-2xl font-semibold tracking-tight text-slate-900">{t('guide.title')}</h1>
        <p className="mt-2 text-sm leading-relaxed text-slate-600">{t('guide.subtitle')}</p>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm md:p-6">
        <h2 className="m-0 text-lg font-semibold text-slate-900">{t('guide.quickStart.title')}</h2>
        <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm leading-relaxed text-slate-700">
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
  return (
    <Link
      to={to}
      className="cursor-pointer rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition-colors duration-200 hover:border-blue-300 hover:bg-blue-50/40 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
    >
      <h3 className="m-0 text-base font-semibold text-slate-900">{title}</h3>
      <p className="mt-2 text-sm leading-relaxed text-slate-600">{desc}</p>
    </Link>
  )
}
