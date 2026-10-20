import { useEffect, useState } from 'react'
import { api, type Balance, type CalendarEntry, type Employee, type LeaveRequestItem, type LeaveType } from '../api'
import StatusPill from '../components/StatusPill'
import TeamCalendar from '../components/TeamCalendar'

export default function LeaveView({ me }: { me: Employee }) {
  const [employees, setEmployees] = useState<Employee[]>([])
  const [types, setTypes] = useState<LeaveType[]>([])
  const [balances, setBalances] = useState<Balance[]>([])
  const [requests, setRequests] = useState<LeaveRequestItem[]>([])
  const [calendar, setCalendar] = useState<CalendarEntry[]>([])
  const [leaveTypeId, setLeaveTypeId] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [newTypeName, setNewTypeName] = useState('')
  const [newTypeDays, setNewTypeDays] = useState('')
  const [error, setError] = useState<string | null>(null)

  function refresh() {
    api.employees().then(setEmployees)
    api.leaveTypes().then(setTypes)
    api.balances().then(setBalances)
    api.leaveRequests().then(setRequests)
    api.leaveCalendar().then(setCalendar)
  }

  useEffect(refresh, [])

  async function submitRequest(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.createLeaveRequest({ leave_type_id: Number(leaveTypeId), start_date: startDate, end_date: endDate })
      setStartDate('')
      setEndDate('')
      refresh()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function addLeaveType(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.createLeaveType({ name: newTypeName, annual_balance_days: Number(newTypeDays) })
      setNewTypeName('')
      setNewTypeDays('')
      refresh()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function decide(id: number, approve: boolean) {
    try {
      await api.decideLeaveRequest(id, approve)
      refresh()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  function employeeName(id: number) {
    return employees.find((e) => e.id === id)?.full_name ?? `#${id}`
  }

  function typeName(id: number) {
    return types.find((t) => t.id === id)?.name ?? `#${id}`
  }

  const decidable = requests.filter(
    (r) => r.status === 'pending' && r.employee_id !== me.id && (me.role === 'manager' || me.role === 'hr_admin'),
  )

  return (
    <div>
      <h2>Leave</h2>
      {error && <p className="form-error">{error}</p>}

      {me.role !== 'viewer' && (
      <div className="balance-row">
        {balances.map((b) => {
          const pct =
            b.annual_balance_days > 0 ? Math.max(0, Math.min(100, (b.used_days / b.annual_balance_days) * 100)) : 0
          return (
            <div key={b.leave_type_id} className="balance-tile">
              <div className="balance-tile-top">
                <span>{b.leave_type_name}</span>
                <span>{b.remaining_days} left</span>
              </div>
              <div className="balance-bar">
                <div className="balance-bar-fill" style={{ width: `${pct}%` }} />
              </div>
            </div>
          )
        })}
      </div>
      )}

      <div className="panel">
        <h3>Team calendar</h3>
        <TeamCalendar entries={calendar} />
      </div>

      {me.role !== 'viewer' && (
      <div className="panel">
        <h3>Request leave</h3>
        <form onSubmit={submitRequest} className="form-row">
          <label>
            Type
            <select value={leaveTypeId} onChange={(e) => setLeaveTypeId(e.target.value)} required>
              <option value="">Choose</option>
              {types.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Start date
            <input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} required />
          </label>
          <label>
            End date
            <input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} required />
          </label>
          <button type="submit" className="btn-primary">
            Request leave
          </button>
        </form>
      </div>
      )}

      <div className="panel">
        <h3>Requests</h3>
        <table>
          <thead>
            <tr>
              <th>Employee</th>
              <th>Type</th>
              <th>Dates</th>
              <th>Days</th>
              <th>Status</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {requests.map((r) => (
              <tr key={r.id}>
                <td>{employeeName(r.employee_id)}</td>
                <td>{typeName(r.leave_type_id)}</td>
                <td>
                  {r.start_date.slice(0, 10)} to {r.end_date.slice(0, 10)}
                </td>
                <td>{r.days}</td>
                <td>
                  <StatusPill tone={r.status}>{r.status}</StatusPill>{' '}
                  {r.is_lop && (
                    <StatusPill tone="flag">
                      {r.lop_days} day{r.lop_days === 1 ? '' : 's'} loss of pay
                    </StatusPill>
                  )}
                </td>
                <td>
                  {decidable.some((d) => d.id === r.id) && (
                    <>
                      <button type="button" className="btn-text" onClick={() => decide(r.id, true)}>
                        Approve
                      </button>
                      <button type="button" className="btn-text" onClick={() => decide(r.id, false)}>
                        Reject
                      </button>
                    </>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {me.role === 'hr_admin' && (
        <div className="panel">
          <h3>Add a leave type</h3>
          <form onSubmit={addLeaveType} className="form-row">
            <label>
              Name
              <input value={newTypeName} onChange={(e) => setNewTypeName(e.target.value)} required />
            </label>
            <label>
              Annual balance (days)
              <input type="number" value={newTypeDays} onChange={(e) => setNewTypeDays(e.target.value)} required />
            </label>
            <button type="submit" className="btn-primary">
              Add leave type
            </button>
          </form>
        </div>
      )}
    </div>
  )
}
