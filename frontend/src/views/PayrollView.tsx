import { Fragment, useEffect, useState } from 'react'
import {
  api,
  type ComponentLine,
  type DeductionLine,
  type Employee,
  type PayrollRun,
  type Payslip,
  type SalaryComponent,
  type StructureLine,
} from '../api'
import { money } from '../format'
import { downloadPayslipPdf, payslipPdfUrl } from '../payslipPdf'

type SubTab = 'payslips' | 'structure' | 'components' | 'run'

export default function PayrollView({ me }: { me: Employee }) {
  const canEdit = me.role === 'hr_admin'
  const canSeeAdminTabs = canEdit || me.role === 'viewer'
  const [subTab, setSubTab] = useState<SubTab>('payslips')

  const tabs: { key: SubTab; label: string }[] = [
    { key: 'payslips', label: 'Payslips' },
    { key: 'structure', label: 'Salary structure' },
    ...(canSeeAdminTabs
      ? ([{ key: 'components', label: 'Pay components' }, { key: 'run', label: 'Run payroll' }] as const)
      : []),
  ]

  return (
    <div>
      <h2>Payroll</h2>
      <div className="subtabs">
        {tabs.map((t) => (
          <button
            key={t.key}
            type="button"
            className={subTab === t.key ? 'subtab subtab-active' : 'subtab'}
            onClick={() => setSubTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {subTab === 'payslips' && <PayslipsTab me={me} />}
      {subTab === 'structure' && <StructureTab me={me} />}
      {subTab === 'components' && canSeeAdminTabs && <ComponentsTab editable={canEdit} />}
      {subTab === 'run' && canSeeAdminTabs && <RunTab editable={canEdit} />}
    </div>
  )
}

function PayslipsTab({ me }: { me: Employee }) {
  const [employees, setEmployees] = useState<Employee[]>([])
  const [payslips, setPayslips] = useState<Payslip[]>([])
  const [breakdownByPayslip, setBreakdownByPayslip] = useState<
    Record<number, { deductions: DeductionLine[]; components: ComponentLine[] }>
  >({})

  useEffect(() => {
    api.employees().then(setEmployees)
    api.payslips().then(setPayslips)
  }, [me.role])

  async function toggleBreakdown(payslipId: number) {
    if (breakdownByPayslip[payslipId]) {
      const copy = { ...breakdownByPayslip }
      delete copy[payslipId]
      setBreakdownByPayslip(copy)
      return
    }
    const [deductions, componentLines] = await Promise.all([
      api.payslipDeductions(payslipId),
      api.payslipComponents(payslipId),
    ])
    setBreakdownByPayslip({ ...breakdownByPayslip, [payslipId]: { deductions, components: componentLines } })
  }

  function employeeName(id: number) {
    return employees.find((e) => e.id === id)?.full_name ?? `#${id}`
  }

  function periodLabel(month: number, year: number) {
    const name = new Date(year, month - 1).toLocaleString('en-US', { month: 'long' })
    return `${name} ${year}`
  }

  async function loadBreakdown(p: Payslip) {
    return (
      breakdownByPayslip[p.id] ??
      (await (async () => {
        const [deductions, components] = await Promise.all([api.payslipDeductions(p.id), api.payslipComponents(p.id)])
        return { deductions, components }
      })())
    )
  }

  async function downloadPayslip(p: Payslip) {
    const breakdown = await loadBreakdown(p)
    downloadPayslipPdf(p, employeeName(p.employee_id), periodLabel(p.period_month, p.period_year), breakdown.components, breakdown.deductions)
  }

  async function viewPayslip(p: Payslip) {
    // open the tab synchronously, in direct response to the click, so the browser
    // never treats it as a popup; the PDF URL is dropped in once it is ready
    const tab = window.open('', '_blank')
    const breakdown = await loadBreakdown(p)
    const url = payslipPdfUrl(p, employeeName(p.employee_id), periodLabel(p.period_month, p.period_year), breakdown.components, breakdown.deductions)
    if (tab) tab.location.href = url
  }

  if (payslips.length === 0) {
    return <p className="view-intro">No payslips yet. They appear here once HR Admin runs payroll.</p>
  }

  return (
    <table>
      <thead>
        <tr>
          <th>Employee</th>
          <th>Period</th>
          <th>Gross earnings</th>
          <th>Deductions</th>
          <th>Loss of pay</th>
          <th>Net pay</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {payslips.map((p, i) => {
          const breakdown = breakdownByPayslip[p.id]
          const isNewRun = i > 0 && payslips[i - 1].payroll_run_id !== p.payroll_run_id
          return (
            <Fragment key={p.id}>
              <tr className={isNewRun ? 'payrun-start' : undefined}>
                <td>{employeeName(p.employee_id)}</td>
                <td>{periodLabel(p.period_month, p.period_year)}</td>
                <td>{money(p.gross_earnings)}</td>
                <td>{money(p.total_deductions)}</td>
                <td>{money(p.lop_deduction)}</td>
                <td>{money(p.net_pay)}</td>
                <td>
                  <button type="button" className="btn-text" onClick={() => toggleBreakdown(p.id)}>
                    {breakdown ? 'Hide' : 'Details'}
                  </button>{' '}
                  <button type="button" className="btn-text" onClick={() => viewPayslip(p)}>
                    View
                  </button>{' '}
                  <button type="button" className="btn-text" onClick={() => downloadPayslip(p)}>
                    Download
                  </button>
                </td>
              </tr>
              {breakdown && (
                <tr className="deduction-row">
                  <td colSpan={7}>
                    {breakdown.components.map((c) => (
                      <div key={`c-${c.id}`} className="deduction-line">
                        {c.name} ({c.kind === 'earning' ? 'added' : 'deducted'}): {money(c.amount)}
                      </div>
                    ))}
                    {breakdown.deductions.map((d) => (
                      <div key={`d-${d.id}`} className="deduction-line">
                        {d.description}, amount {money(d.amount)}.
                      </div>
                    ))}
                  </td>
                </tr>
              )}
            </Fragment>
          )
        })}
      </tbody>
    </table>
  )
}

function StructureTab({ me }: { me: Employee }) {
  const canEdit = me.role === 'hr_admin'
  const canBrowseAnyone = canEdit || me.role === 'viewer'
  const [employees, setEmployees] = useState<Employee[]>([])
  const [selectedId, setSelectedId] = useState<number>(me.id)
  const [lines, setLines] = useState<StructureLine[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (canBrowseAnyone) api.employees().then(setEmployees)
  }, [canBrowseAnyone])

  function refresh() {
    api.employeeStructure(selectedId).then(setLines).catch((err) => setError(err.message))
  }

  useEffect(refresh, [selectedId])

  async function save(componentId: number, amount: number) {
    try {
      await api.setEmployeeSalaryComponent({ employee_id: selectedId, component_id: componentId, amount })
      refresh()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  const earnings = lines.filter((l) => l.kind === 'earning')
  const deductions = lines.filter((l) => l.kind === 'deduction')
  const totalEarnings = earnings.reduce((sum, l) => sum + (l.amount ?? 0), 0)
  const totalDeductions = deductions.reduce((sum, l) => sum + (l.amount ?? 0), 0)

  return (
    <div>
      {canBrowseAnyone && (
        <label className="structure-employee-picker">
          Employee
          <select value={selectedId} onChange={(e) => setSelectedId(Number(e.target.value))}>
            {employees.map((emp) => (
              <option key={emp.id} value={emp.id}>
                {emp.full_name}
              </option>
            ))}
          </select>
        </label>
      )}
      {error && <p className="form-error">{error}</p>}

      <StructureGroup key={`${selectedId}-earnings`} title="Earnings" lines={earnings} editable={canEdit} onSave={save} />
      <StructureGroup key={`${selectedId}-deductions`} title="Deductions" lines={deductions} editable={canEdit} onSave={save} />

      {lines.length > 0 && (
        <p className="form-note">
          Gross {money(totalEarnings)} minus deductions {money(totalDeductions)} puts net pay at{' '}
          {money(totalEarnings - totalDeductions)} before any Loss of Pay for the period.
        </p>
      )}
    </div>
  )
}

function StructureGroup({
  title,
  lines,
  editable,
  onSave,
}: {
  title: string
  lines: StructureLine[]
  editable: boolean
  onSave: (componentId: number, amount: number) => void
}) {
  return (
    <div className="panel">
      <h3>{title}</h3>
      {lines.length === 0 ? (
        <p className="view-intro">No {title.toLowerCase()} components defined yet.</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Component</th>
              <th>Amount</th>
              {editable && <th></th>}
            </tr>
          </thead>
          <tbody>
            {lines.map((line) => (
              <StructureRow key={line.component_id} line={line} editable={editable} onSave={onSave} />
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

function StructureRow({
  line,
  editable,
  onSave,
}: {
  line: StructureLine
  editable: boolean
  onSave: (componentId: number, amount: number) => void
}) {
  const [value, setValue] = useState(line.amount !== null ? String(line.amount) : '')

  // line.amount can change out from under this row (switching employee, or after a
  // save completes); re-sync the field instead of trusting the value it mounted with.
  useEffect(() => {
    setValue(line.amount !== null ? String(line.amount) : '')
  }, [line.amount])

  if (!editable) {
    return (
      <tr>
        <td>
          {line.name}
          {line.is_base && ' (required)'}
        </td>
        <td>{line.amount !== null ? money(line.amount) : 'Not set'}</td>
      </tr>
    )
  }

  return (
    <tr>
      <td>
        {line.name}
        {line.is_base && ' (required)'}
      </td>
      <td>
        <input type="number" value={value} onChange={(e) => setValue(e.target.value)} />
      </td>
      <td>
        <button
          type="button"
          className="btn-text"
          disabled={value === ''}
          onClick={() => onSave(line.component_id, Number(value))}
        >
          {line.amount !== null ? 'Update' : 'Assign'}
        </button>
      </td>
    </tr>
  )
}

function ComponentsTab({ editable }: { editable: boolean }) {
  const [components, setComponents] = useState<SalaryComponent[]>([])
  const [name, setName] = useState('')
  const [kind, setKind] = useState<'earning' | 'deduction'>('deduction')
  const [error, setError] = useState<string | null>(null)

  function refresh() {
    api.salaryComponents().then(setComponents)
  }

  useEffect(refresh, [])

  async function addComponent(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.createSalaryComponent({ name, kind })
      setName('')
      refresh()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function removeComponent(c: SalaryComponent) {
    if (!window.confirm(`Remove ${c.name}? This takes it off everyone it's currently assigned to.`)) return
    try {
      await api.deleteSalaryComponent(c.id)
      refresh()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  const earnings = components.filter((c) => c.kind === 'earning')
  const deductions = components.filter((c) => c.kind === 'deduction')

  return (
    <div>
      <p className="view-intro">
        These are the pay items available to assign to anyone, Basic Pay always exists, add anything else your
        organization needs: HRA, Provident Fund, a bonus, insurance.
      </p>
      {error && <p className="form-error">{error}</p>}

      <div className="panel">
        <h3>Earnings</h3>
        <ul className="plain-list">
          {earnings.map((c) => (
            <li key={c.id}>
              {c.name}
              {c.is_base && ' (required for every employee)'}
              {editable && !c.is_base && (
                <button type="button" className="btn-text" onClick={() => removeComponent(c)}>
                  Remove
                </button>
              )}
            </li>
          ))}
        </ul>
        <h3>Deductions</h3>
        <ul className="plain-list">
          {deductions.map((c) => (
            <li key={c.id}>
              {c.name}{' '}
              {editable && (
                <button type="button" className="btn-text" onClick={() => removeComponent(c)}>
                  Remove
                </button>
              )}
            </li>
          ))}
        </ul>
      </div>

      {editable && (
      <div className="panel">
        <h3>Add a pay component</h3>
        <form onSubmit={addComponent} className="form-row">
          <label>
            Name
            <input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Provident Fund" required />
          </label>
          <label>
            Kind
            <select value={kind} onChange={(e) => setKind(e.target.value as 'earning' | 'deduction')}>
              <option value="deduction">Deduction</option>
              <option value="earning">Earning</option>
            </select>
          </label>
          <button type="submit" className="btn-primary">
            Add component
          </button>
        </form>
      </div>
      )}
    </div>
  )
}

function RunTab({ editable }: { editable: boolean }) {
  const [runs, setRuns] = useState<PayrollRun[]>([])
  const [periodMonth, setPeriodMonth] = useState('')
  const [periodYear, setPeriodYear] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [runResult, setRunResult] = useState<string | null>(null)

  function refresh() {
    api.payrollRuns().then(setRuns)
  }

  useEffect(refresh, [])

  async function runPayroll(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setRunResult(null)
    try {
      const result = await api.runPayroll({ period_month: Number(periodMonth), period_year: Number(periodYear) })
      setRunResult(`Run complete. ${result.payslip_count} payslips generated.`)
      refresh()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <div>
      {editable && (
      <div className="panel">
        <h3>Run a new period</h3>
        <p className="view-intro">
          Processes every active employee at once. If anyone is missing Basic Pay, nothing is saved and you can fix it
          and retry.
        </p>
        {error && <p className="form-error">{error}</p>}
        {runResult && <p className="form-note">{runResult}</p>}
        <form onSubmit={runPayroll} className="form-row">
          <label>
            Period month
            <input
              type="number"
              min="1"
              max="12"
              value={periodMonth}
              onChange={(e) => setPeriodMonth(e.target.value)}
              required
            />
          </label>
          <label>
            Period year
            <input type="number" value={periodYear} onChange={(e) => setPeriodYear(e.target.value)} required />
          </label>
          <button type="submit" className="btn-primary">
            Run payroll
          </button>
        </form>
      </div>
      )}

      <div className="panel">
        <h3>Past runs</h3>
        <ul className="plain-list">
          {runs.map((r) => (
            <li key={r.id}>
              {r.period_month}/{r.period_year}, run on {new Date(r.run_at).toLocaleDateString()}
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
