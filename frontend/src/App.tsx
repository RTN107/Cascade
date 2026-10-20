import { useEffect, useState } from 'react'
import { api, type Employee } from './api'
import Shell, { type ViewKey } from './components/Shell'
import AuthView from './views/AuthView'
import DashboardView from './views/DashboardView'
import PeopleView from './views/PeopleView'
import LeaveView from './views/LeaveView'
import PayrollView from './views/PayrollView'
import './App.css'

type Phase = 'loading' | 'needs-setup' | 'needs-login' | 'ready'

function App() {
  const [phase, setPhase] = useState<Phase>('loading')
  const [me, setMe] = useState<Employee | null>(null)
  const [activeView, setActiveView] = useState<ViewKey>('dashboard')

  useEffect(() => {
    api
      .setupNeeded()
      .then(({ needed }) => {
        if (needed) {
          setPhase('needs-setup')
          return
        }
        return api
          .me()
          .then((emp) => {
            setMe(emp)
            setPhase('ready')
          })
          .catch(() => setPhase('needs-login'))
      })
      .catch(() => setPhase('needs-login'))
  }, [])

  function handleAuthed(emp: Employee) {
    setMe(emp)
    setPhase('ready')
  }

  async function handleLogout() {
    await api.logout()
    setMe(null)
    setActiveView('dashboard')
    setPhase('needs-login')
  }

  if (phase === 'loading') return null
  if (phase === 'needs-setup') return <AuthView needsSetup onDone={handleAuthed} />
  if (phase === 'needs-login') return <AuthView needsSetup={false} onDone={handleAuthed} />
  if (!me) return null

  return (
    <Shell me={me} active={activeView} onNavigate={setActiveView} onLogout={handleLogout}>
      {activeView === 'dashboard' && <DashboardView me={me} />}
      {activeView === 'people' && <PeopleView me={me} />}
      {activeView === 'leave' && <LeaveView me={me} />}
      {activeView === 'payroll' && <PayrollView me={me} />}
    </Shell>
  )
}

export default App
