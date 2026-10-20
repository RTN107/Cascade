import { useState } from 'react'
import { api, type Employee } from '../api'
import Logo from '../components/Logo'

export default function AuthView({
  needsSetup,
  onDone,
}: {
  needsSetup: boolean
  onDone: (emp: Employee) => void
}) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [fullName, setFullName] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      if (needsSetup) {
        await api.setup({ email, password, full_name: fullName })
      } else {
        await api.login({ email, password })
      }
      onDone(await api.me())
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <div className="auth-screen">
      <div className="auth-panel">
        <Logo size="large" />
        <p className="auth-tagline">See how your organization actually connects.</p>

        <form onSubmit={submit}>
          {needsSetup && (
            <label>
              Full name
              <input value={fullName} onChange={(e) => setFullName(e.target.value)} required />
            </label>
          )}
          <label>
            Email
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </label>
          <label>
            Password
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
          </label>
          {error && <p className="form-error">{error}</p>}
          <button type="submit" className="btn-primary">
            {needsSetup ? 'Create HR Admin account' : 'Log in'}
          </button>
        </form>
        {needsSetup && <p className="auth-note">No employees exist yet. This creates the first HR Admin.</p>}
      </div>
    </div>
  )
}
