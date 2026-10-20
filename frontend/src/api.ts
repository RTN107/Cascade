const API_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    throw new Error((await res.text()) || res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export interface Employee {
  id: number
  email: string
  full_name: string
  title: string | null
  role: 'employee' | 'manager' | 'hr_admin' | 'viewer'
  department_id: number | null
  manager_id: number | null
}

export interface Department {
  id: number
  name: string
}

export interface LeaveType {
  id: number
  name: string
  annual_balance_days: number
}

export interface LeaveRequestItem {
  id: number
  employee_id: number
  leave_type_id: number
  start_date: string
  end_date: string
  days: number
  status: 'pending' | 'approved' | 'rejected'
  is_lop: boolean
  lop_days: number
  decided_by: number | null
  decided_at: string | null
  created_at: string
}

export interface CalendarEntry {
  employee_id: number
  employee_name: string
  leave_type_name: string
  start_date: string
  end_date: string
  days: number
}

export interface Balance {
  leave_type_id: number
  leave_type_name: string
  annual_balance_days: number
  used_days: number
  remaining_days: number
}

export interface PayrollRun {
  id: number
  period_month: number
  period_year: number
  run_by: number
  run_at: string
}

export interface Payslip {
  id: number
  payroll_run_id: number
  period_month: number
  period_year: number
  employee_id: number
  gross_earnings: number
  total_deductions: number
  lop_deduction: number
  net_pay: number
  created_at: string
}

export interface DeductionLine {
  id: number
  payslip_id: number
  leave_request_id: number | null
  amount: number
  description: string
  leave_start_date: string | null
  leave_end_date: string | null
}

export interface SalaryComponent {
  id: number
  name: string
  kind: 'earning' | 'deduction'
  is_base: boolean
}

export interface StructureLine {
  component_id: number
  name: string
  kind: 'earning' | 'deduction'
  is_base: boolean
  amount: number | null
}

export interface ComponentLine {
  id: number
  payslip_id: number
  component_id: number | null
  name: string
  kind: 'earning' | 'deduction'
  amount: number
}

export interface DashboardData {
  role: string
  total_employees?: number
  total_departments?: number
  pending_leave_requests?: number
  outstanding_lop_flags?: number
  last_run?: { period_month: number; period_year: number; total_net_pay: number } | null
  leave_status_breakdown?: { pending: number; approved: number; rejected: number }
  department_breakdown?: { name: string; count: number }[]
  team_size?: number
  pending_approvals?: number
  total_remaining_days?: number
  latest_net_pay?: number | null
  latest_net_pay_month?: number | null
  latest_net_pay_year?: number | null
}

export const api = {
  setupNeeded: () => request<{ needed: boolean }>('/api/setup/needed'),
  setup: (body: { email: string; password: string; full_name: string }) =>
    request<{ id: number }>('/api/setup', { method: 'POST', body: JSON.stringify(body) }),
  login: (body: { email: string; password: string }) =>
    request<Employee>('/api/login', { method: 'POST', body: JSON.stringify(body) }),
  logout: () => request<void>('/api/logout', { method: 'POST' }),
  me: () => request<Employee>('/api/me'),
  employees: () => request<Employee[]>('/api/employees'),
  departments: () => request<Department[]>('/api/departments'),
  createDepartment: (name: string) =>
    request<{ id: number }>('/api/departments', { method: 'POST', body: JSON.stringify({ name }) }),
  createEmployee: (body: {
    email: string
    password: string
    full_name: string
    title: string
    role: string
    department_id: number | null
    manager_id: number | null
  }) => request<{ id: number }>('/api/employees', { method: 'POST', body: JSON.stringify(body) }),
  leaveTypes: () => request<LeaveType[]>('/api/leave-types'),
  createLeaveType: (body: { name: string; annual_balance_days: number }) =>
    request<{ id: number }>('/api/leave-types', { method: 'POST', body: JSON.stringify(body) }),
  leaveRequests: () => request<LeaveRequestItem[]>('/api/leave-requests'),
  createLeaveRequest: (body: { leave_type_id: number; start_date: string; end_date: string }) =>
    request<{ id: number; days: number }>('/api/leave-requests', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  decideLeaveRequest: (id: number, approve: boolean) =>
    request<{ status: string; is_lop: boolean }>(`/api/leave-requests/${id}/decide`, {
      method: 'POST',
      body: JSON.stringify({ approve }),
    }),
  balances: () => request<Balance[]>('/api/leave-balances'),
  leaveCalendar: () => request<CalendarEntry[]>('/api/leave-calendar'),
  employeeStructure: (employeeId: number) => request<StructureLine[]>(`/api/employees/${employeeId}/salary-structure`),
  runPayroll: (body: { period_month: number; period_year: number }) =>
    request<{ id: number; payslip_count: number }>('/api/payroll/run', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  payrollRuns: () => request<PayrollRun[]>('/api/payroll/runs'),
  payslips: () => request<Payslip[]>('/api/payslips'),
  payslipDeductions: (id: number) => request<DeductionLine[]>(`/api/payslips/${id}/deductions`),
  payslipComponents: (id: number) => request<ComponentLine[]>(`/api/payslips/${id}/components`),
  dashboard: () => request<DashboardData>('/api/dashboard'),
  salaryComponents: () => request<SalaryComponent[]>('/api/salary-components'),
  createSalaryComponent: (body: { name: string; kind: 'earning' | 'deduction' }) =>
    request<{ id: number }>('/api/salary-components', { method: 'POST', body: JSON.stringify(body) }),
  deleteSalaryComponent: (id: number) => request<void>(`/api/salary-components/${id}`, { method: 'DELETE' }),
  setEmployeeSalaryComponent: (body: { employee_id: number; component_id: number; amount: number }) =>
    request<void>('/api/employee-salary-components', { method: 'POST', body: JSON.stringify(body) }),
}
