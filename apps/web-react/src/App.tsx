import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppLayout } from './layout/AppLayout'
import { ErrorBoundary } from './components/ErrorBoundary'

// 懒加载页面组件
const DashboardPage = lazy(() => import('./pages/DashboardPage').then(module => ({ default: module.DashboardPage })))
const EventsPage = lazy(() => import('./pages/EventsPage').then(module => ({ default: module.EventsPage })))
const AlertsPage = lazy(() => import('./pages/AlertsPage').then(module => ({ default: module.AlertsPage })))
const RulesPage = lazy(() => import('./pages/RulesPage').then(module => ({ default: module.RulesPage })))
const FailuresPage = lazy(() => import('./pages/FailuresPage').then(module => ({ default: module.FailuresPage })))
const AuditLogsPage = lazy(() => import('./pages/AuditLogsPage').then(module => ({ default: module.AuditLogsPage })))
const SystemConfigPage = lazy(() => import('./pages/SystemConfigPage').then(module => ({ default: module.SystemConfigPage })))
const GuidePage = lazy(() => import('./pages/GuidePage').then(module => ({ default: module.GuidePage })))

// 加载占位符组件
function LoadingFallback() {
  return (
    <div className="flex items-center justify-center py-12">
      <div className="flex items-center gap-3 text-slate-600 dark:text-slate-300">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-current border-t-transparent" />
        <span>加载中...</span>
      </div>
    </div>
  )
}

export function App() {
  return (
    <ErrorBoundary>
      <AppLayout>
        <Suspense fallback={<LoadingFallback />}>
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<DashboardPage />} />
            <Route path="/events" element={<EventsPage />} />
            <Route path="/alerts" element={<AlertsPage />} />
            <Route path="/rules" element={<RulesPage />} />
            <Route path="/failures" element={<FailuresPage />} />
            <Route path="/audit" element={<AuditLogsPage />} />
            <Route path="/system-config" element={<SystemConfigPage />} />
            <Route path="/guide" element={<GuidePage />} />
          </Routes>
        </Suspense>
      </AppLayout>
    </ErrorBoundary>
  )
}
