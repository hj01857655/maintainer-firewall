import type { CSSProperties, ReactNode } from 'react'
import { NavLink } from 'react-router-dom'

type Props = {
  children: ReactNode
}

export function AppLayout({ children }: Props) {
  return (
    <div style={rootStyle}>
      <aside style={sidebarStyle}>
        <h2 style={{ margin: 0, color: '#0F172A' }}>Maintainer Firewall</h2>
        <nav style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 8 }}>
          <NavItem to="/dashboard" label="Dashboard" />
          <NavItem to="/events" label="Events" />
          <NavItem to="/alerts" label="Alerts" />
        </nav>
      </aside>

      <main style={contentStyle}>{children}</main>
    </div>
  )
}

function NavItem({ to, label }: { to: string; label: string }) {
  return (
    <NavLink
      to={to}
      style={({ isActive }) => ({
        display: 'block',
        minHeight: 44,
        lineHeight: '44px',
        padding: '0 12px',
        borderRadius: 8,
        textDecoration: 'none',
        color: isActive ? '#0F172A' : '#334155',
        background: isActive ? '#E2E8F0' : 'transparent',
        border: '1px solid #E2E8F0',
        cursor: 'pointer',
        transition: 'background-color 0.2s ease, color 0.2s ease',
      })}
    >
      {label}
    </NavLink>
  )
}

const rootStyle: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: '240px 1fr',
  minHeight: '100vh',
  maxWidth: '100%',
  overflowX: 'hidden',
  background: '#F8FAFC',
}

const sidebarStyle: CSSProperties = {
  borderRight: '1px solid #E2E8F0',
  background: '#FFFFFF',
  padding: 16,
}

const contentStyle: CSSProperties = {
  padding: 24,
  color: '#0F172A',
}
