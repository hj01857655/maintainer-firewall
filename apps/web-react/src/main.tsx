import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { AppRouter } from './AppRouter'
import { ThemeProvider, initTheme } from './theme'
import { QueryProvider } from './components/QueryProvider'
import './i18n'
import './index.css'

initTheme()

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ThemeProvider>
      <QueryProvider>
        <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
          <AppRouter />
        </BrowserRouter>
      </QueryProvider>
    </ThemeProvider>
  </React.StrictMode>
)
