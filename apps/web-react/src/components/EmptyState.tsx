import { useTranslation } from 'react-i18next'

interface EmptyStateProps {
  title?: string
  description?: string
  icon?: React.ReactNode
  action?: React.ReactNode
}

export function EmptyState({
  title,
  description,
  icon,
  action
}: EmptyStateProps) {
  const { t } = useTranslation()

  const defaultIcon = (
    <svg className="h-12 w-12 text-slate-400 dark:text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
    </svg>
  )

  return (
    <div className="flex flex-col items-center justify-center py-16 px-4 text-center">
      <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-slate-100 dark:bg-slate-800">
        {icon || defaultIcon}
      </div>
      <h3 className="mb-2 text-lg font-semibold text-slate-900 dark:text-slate-100">
        {title || t('common.noData')}
      </h3>
      <p className="mb-6 max-w-sm text-sm text-slate-600 dark:text-slate-400">
        {description || t('common.noDataDescription')}
      </p>
      {action}
    </div>
  )
}
