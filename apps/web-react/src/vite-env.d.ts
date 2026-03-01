/// <reference types="vite/client" />

// Global type declarations for Google Analytics
declare global {
  interface Window {
    gtag?: (
      command: 'config' | 'set' | 'event',
      targetId: string | Record<string, any>,
      config?: Record<string, any>
    ) => void
  }
}
