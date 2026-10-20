import { useEffect, useState } from 'react'
import { api, type Department, type Employee } from '../api'
import OrgTree from '../components/OrgTree'

export default function PeopleView({ me }: { me: Employee }) {
  const [employees, setEmployees] = useState<Employee[]>([])
  const [departments, setDepartments] = useState<Department[]>([])
  const [error, setError] = useState<string | null>(null)

  const [deptName, setDeptName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [fullName, setFullName] = useState('')
  const [title, setTitle] = useState('')
  const [role, setRole] = useState('employee')
  const [departmentId, setDepartmentId] = useState('')
  const [managerId, setManagerId] = useState('')

  function refresh() {
    api.employees().then(setEmployees).catch((err) => setError(err.message))
    api.departments().then(setDepartments).catch((err) => setError(err.message))
  }

  useEffect(refresh, [])

  async function addDepartment(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.createDepartment(deptName)
      setDeptName('')
      refresh()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function addEmployee(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.createEmployee({
        email,
        password,
        full_name: fullName,
        title,
        role,
        department_id: departmentId ? Number(departmentId) : null,
        manager_id: managerId ? Number(managerId) : null,
      })
      setEmail('')
      setPassword('')
      setFullName('')
      setTitle('')
      refresh()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <div>
      <h2>People</h2>
      {error && <p className="form-error">{error}</p>}
      <p className="view-intro">
        {employees.length} {employees.length === 1 ? 'person' : 'people'} visible to you, arranged by who reports to
        whom.
      </p>

      <OrgTree employees={employees} departments={departments} meId={me.id} />

      {me.role === 'hr_admin' && (
        <div className="panel">
          <h3>Add a department</h3>
          <form onSubmit={addDepartment} className="form-row">
            <label>
              Name
              <input value={deptName} onChange={(e) => setDeptName(e.target.value)} required />
            </label>
            <button type="submit" className="btn-primary">
              Add department
            </button>
          </form>

          <h3>Add an employee</h3>
          <form onSubmit={addEmployee} className="form-grid">
            <label>
              Full name
              <input value={fullName} onChange={(e) => setFullName(e.target.value)} required />
            </label>
            <label>
              Email
              <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
            </label>
            <label>
              Job title
              <input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g. Engineering Manager"
                required
              />
            </label>
            <label>
              Password
              <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
            </label>
            <label>
              Role
              <select value={role} onChange={(e) => setRole(e.target.value)}>
                <option value="employee">Employee</option>
                <option value="manager">Manager</option>
                <option value="hr_admin">HR Admin</option>
              </select>
            </label>
            <label>
              Department
              <select value={departmentId} onChange={(e) => setDepartmentId(e.target.value)}>
                <option value="">None</option>
                {departments.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Manager
              <select value={managerId} onChange={(e) => setManagerId(e.target.value)}>
                <option value="">None</option>
                {employees
                  .filter((emp) => emp.role !== 'hr_admin')
                  .map((emp) => (
                    <option key={emp.id} value={emp.id}>
                      {emp.full_name}
                    </option>
                  ))}
              </select>
            </label>
            <button type="submit" className="btn-primary">
              Add employee
            </button>
          </form>
        </div>
      )}
    </div>
  )
}
