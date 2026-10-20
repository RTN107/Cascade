import { useEffect, useState } from 'react'
import { api, type DashboardData, type Employee } from '../api'
import { rupees } from '../format'

function Stat({ value, label, sub }: { value: React.ReactNode; label: string; sub?: string }) {
  return (
    <div className="stat-tile">
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
      {sub && <div className="stat-sub">{sub}</div>}
    </div>
  )
}

function monthYear(month?: number | null, year?: number | null) {
  if (!month || !year) return undefined
  return new Date(year, month - 1).toLocaleString('en-US', { month: 'long', year: 'numeric' })
}

type Tone = 'accent' | 'ink-light' | 'ink-dark'

function StackedBar({
  title,
  segments,
}: {
  title: string
  segments: { label: string; value: number; tone: Tone }[]
}) {
  const total = segments.reduce((sum, s) => sum + s.value, 0)
  if (total === 0) return null

  return (
    <div className="panel">
      <h3>{title}</h3>
      <div className="stacked-bar">
        {segments.map(
          (s) =>
            s.value > 0 && (
              <div
                key={s.label}
                className={`stacked-bar-segment stacked-bar-${s.tone}`}
                style={{ width: `${(s.value / total) * 100}%` }}
              />
            ),
        )}
      </div>
      <div className="stacked-bar-legend">
        {segments.map((s) => (
          <span key={s.label} className="stacked-bar-legend-item">
            <span className={`legend-dot legend-dot-${s.tone}`} />
            {s.label} ({s.value})
          </span>
        ))}
      </div>
    </div>
  )
}

export default function DashboardView({ me }: { me: Employee }) {
  const [data, setData] = useState<DashboardData | null>(null)

  useEffect(() => {
    api.dashboard().then(setData)
  }, [])

  if (!data) return null

  return (
    <div>
      <h2>Dashboard</h2>
      <div className="stat-row">
        {(me.role === 'hr_admin' || me.role === 'viewer') && (
          <>
            <Stat value={data.total_employees} label="Employees" />
            <Stat value={data.total_departments} label="Departments" />
            <Stat value={data.outstanding_lop_flags} label="Loss of pay not yet billed" />
            <Stat
              value={data.last_run ? rupees(data.last_run.total_net_pay) : 'None yet'}
              label="Last payroll run, total net pay"
              sub={data.last_run ? monthYear(data.last_run.period_month, data.last_run.period_year) : 'No payroll run yet'}
            />
          </>
        )}
        {me.role === 'manager' && (
          <>
            <Stat value={data.team_size} label="People on your team" />
            <Stat value={data.pending_approvals} label="Waiting on your approval" />
          </>
        )}
        {me.role === 'employee' && (
          <>
            <Stat value={data.pending_leave_requests} label="Your pending requests" />
            <Stat value={data.total_remaining_days} label="Leave days remaining" />
            <Stat
              value={data.latest_net_pay != null ? rupees(data.latest_net_pay) : 'None yet'}
              label="Latest payslip, net pay"
              sub={monthYear(data.latest_net_pay_month, data.latest_net_pay_year)}
            />
          </>
        )}
      </div>

      {(me.role === 'hr_admin' || me.role === 'viewer') && data.leave_status_breakdown && (
        <StackedBar
          title="Leave requests, company-wide"
          segments={[
            { label: 'Pending', value: data.leave_status_breakdown.pending, tone: 'ink-light' },
            { label: 'Approved', value: data.leave_status_breakdown.approved, tone: 'accent' },
            { label: 'Rejected', value: data.leave_status_breakdown.rejected, tone: 'ink-dark' },
          ]}
        />
      )}

      {(me.role === 'hr_admin' || me.role === 'viewer') && data.department_breakdown && data.department_breakdown.length > 1 && (
        <StackedBar
          title="Headcount by department"
          segments={data.department_breakdown.map((d, i) => ({
            label: d.name,
            value: d.count,
            tone: (['accent', 'ink-light', 'ink-dark'] as Tone[])[i % 3],
          }))}
        />
      )}
    </div>
  )
}
