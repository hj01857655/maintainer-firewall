import { createContext, type ReactNode, useContext, useEffect, useMemo, useState } from 'react'

export type ThemeMode = 'light' | 'dark'
export type ResolvedTheme = 'light' | 'dark'

type ThemeContextValue = {
  mode: ThemeMode
  resolved: ResolvedTheme
  setMode: (next: ThemeMode) => void
}

const STORAGE_KEY = 'mf_theme_mode'

const ThemeContext = createContext<ThemeContextValue | null>(null)

function resolveTheme(mode: ThemeMode): ResolvedTheme {
  return mode
}

function applyTheme(mode: ThemeMode) {
  if (typeof document === 'undefined') return
  const resolved = resolveTheme(mode)
  const root = document.documentElement
  root.classList.toggle('dark', resolved === 'dark')
  root.style.colorScheme = resolved
}

function getInitialMode(): ThemeMode {
  if (typeof window === 'undefined') return 'light'
  const v = window.localStorage.getItem(STORAGE_KEY)
  if (v === 'light' || v === 'dark') return v
  return 'light'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>(() => getInitialMode())
  const resolved = useMemo(() => resolveTheme(mode), [mode])

  useEffect(() => {
    applyTheme(mode)
    try {
      window.localStorage.setItem(STORAGE_KEY, mode)
    } catch {
      // ignore storage errors
    }
  }, [mode])

  const value = useMemo<ThemeContextValue>(
    () => ({ mode, resolved, setMode: setModeState }),
    [mode, resolved],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme() {
  const ctx = useContext(ThemeContext)
  if (!ctx) {
    throw new Error('useTheme must be used within ThemeProvider')
  }
  return ctx
}

export function initTheme() {
  applyTheme(getInitialMode())
}
