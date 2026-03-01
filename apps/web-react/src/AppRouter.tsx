import type { ReactElement } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppLayout } from './layout/AppLayout'
import { getAccessToken } from './auth'
import { DashboardPage } from './pages/DashboardPage'
import { EventsPage } from './pages/EventsPage'
import { AlertsPage } from './pages/AlertsPage'
import { LoginPage } from './pages/LoginPage'
import { RulesPage } from './pages/RulesPage'
import { FailuresPage } from './pages/FailuresPage'
import { AuditLogsPage } from './pages/AuditLogsPage'
import { GuidePage } from './pages/GuidePage'
import { UsersPage } from './pages/UsersPage'
import { SystemConfigPage } from './pages/SystemConfigPage'
import { AnalyticsPage } from './pages/AnalyticsPage'

function RequireAuth({ children }: { children: ReactElement }) {
  return getAccessToken() ? children : <Navigate to="/login" replace />
}

export function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/*"
        element={
          <RequireAuth>
            <AppLayout>
              <Routes>
                <Route path="/" element={<Navigate to="/dashboard" replace />} />
                <Route path="/dashboard" element={<DashboardPage />} />
                <Route path="/guide" element={<GuidePage />} />
                <Route path="/events" element={<EventsPage />} />
                <Route path="/alerts" element={<AlertsPage />} />
                <Route path="/rules" element={<RulesPage />} />
                <Route path="/failures" element={<FailuresPage />} />
                <Route path="/audit" element={<AuditLogsPage />} />
                <Route path="/system-config" element={<SystemConfigPage />} />
                <Route path="/users" element={<UsersPage />} />
                <Route path="/analytics" element={<AnalyticsPage />} />
              </Routes>
            </AppLayout>
          </RequireAuth>
        }
      />
    </Routes>
  )
}
