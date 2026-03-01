// Global type declarations
declare global {
  interface Window {
    gtag?: (
      command: 'config' | 'set' | 'event',
      targetId: string | Record<string, any>,
      config?: Record<string, any>
    ) => void
  }
}
