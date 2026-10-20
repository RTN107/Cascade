import { AnimatePresence, motion } from 'motion/react'
import { useEffect, useState, type ReactNode } from 'react'
import type { Employee } from '../api'
import Logo from './Logo'

export type ViewKey = 'dashboard' | 'people' | 'leave' | 'payroll'

const navItems: { key: ViewKey; label: string }[] = [
  { key: 'dashboard', label: 'Dashboard' },
  { key: 'people', label: 'People' },
  { key: 'leave', label: 'Leave' },
  { key: 'payroll', label: 'Payroll' },
]

const roleLabel: Record<string, string> = {
  employee: 'Employee',
  manager: 'Manager',
  hr_admin: 'HR Admin',
  viewer: 'Viewer (read only)',
}

function ChevronIcon({ flipped }: { flipped: boolean }) {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      style={{ transform: flipped ? 'rotate(180deg)' : undefined }}
      aria-hidden="true"
    >
      <path
        d="M10 3L5 8l5 5"
        stroke="currentColor"
        strokeWidth="1.6"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function useCollapsed() {
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return localStorage.getItem('cascade-rail-collapsed') === '1'
    } catch {
      return false
    }
  })

  useEffect(() => {
    try {
      localStorage.setItem('cascade-rail-collapsed', collapsed ? '1' : '0')
    } catch {
      // ignore: this is just a remembered preference, not critical state
    }
  }, [collapsed])

  return [collapsed, setCollapsed] as const
}

export default function Shell({
  me,
  active,
  onNavigate,
  onLogout,
  children,
}: {
  me: Employee
  active: ViewKey
  onNavigate: (view: ViewKey) => void
  onLogout: () => void
  children: ReactNode
}) {
  const [collapsed, setCollapsed] = useCollapsed()
  const initial = me.full_name.charAt(0).toUpperCase()

  return (
    <div className="shell-layout">
      <nav className={collapsed ? 'rail rail-collapsed' : 'rail'}>
        <div className="rail-top">
          {!collapsed && <Logo />}
          <button
            type="button"
            className="rail-toggle"
            onClick={() => setCollapsed(!collapsed)}
            aria-label={collapsed ? 'Open sidebar' : 'Close sidebar'}
          >
            <ChevronIcon flipped={collapsed} />
          </button>
        </div>
        {!collapsed && (
          <>
            <div className="rail-nav">
              {navItems.map((item) => (
                <button
                  key={item.key}
                  type="button"
                  className={active === item.key ? 'rail-link rail-link-active' : 'rail-link'}
                  onClick={() => onNavigate(item.key)}
                >
                  {item.label}
                </button>
              ))}
            </div>
            <div className="rail-user">
              <span className="rail-user-avatar">{initial}</span>
              <span className="rail-user-info">
                <span className="rail-user-name">{me.full_name}</span>
                <span className="rail-user-role">{roleLabel[me.role] ?? me.role}</span>
              </span>
              <button type="button" className="rail-logout" onClick={onLogout}>
                Log out
              </button>
            </div>
          </>
        )}
      </nav>
      <main className="content">
        <div className="content-inner">
          <AnimatePresence mode="wait">
            <motion.div
              key={active}
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -6 }}
              transition={{ duration: 0.18, ease: [0.2, 0, 0, 1] }}
            >
              {children}
            </motion.div>
          </AnimatePresence>
        </div>
      </main>
    </div>
  )
}
