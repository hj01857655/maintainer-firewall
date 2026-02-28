import type { ReactElement } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppLayout } from './layout/AppLayout'
import { getAccessToken } from './auth'
import { DashboardPage } from './pages/DashboardPage'
import { EventsPage } from './pages/EventsPage'
import { AlertsPage } from './pages/AlertsPage'
import { LoginPage } from './pages/LoginPage'

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
                <Route path="/events" element={<EventsPage />} />
                <Route path="/alerts" element={<AlertsPage />} />
              </Routes>
            </AppLayout>
          </RequireAuth>
        }
      />
    </Routes>
  )
}
