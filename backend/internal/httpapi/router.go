package httpapi

import (
	"net/http"

	"cascade/internal/audit"
	"cascade/internal/employees"
	"cascade/internal/leave"
	"cascade/internal/payroll"
)

type Server struct {
	employees *employees.Store
	audit     *audit.Store
	leave     *leave.Store
	payroll   *payroll.Store
	jwtSecret string
	webOrigin string
}

func NewRouter(empStore *employees.Store, auditStore *audit.Store, leaveStore *leave.Store, payrollStore *payroll.Store, jwtSecret, webOrigin string) http.Handler {
	s := &Server{employees: empStore, audit: auditStore, leave: leaveStore, payroll: payrollStore, jwtSecret: jwtSecret, webOrigin: webOrigin}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/setup/needed", s.handleSetupNeeded)
	mux.HandleFunc("POST /api/setup", s.handleSetup)
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.HandleFunc("POST /api/logout", s.handleLogout)
	mux.HandleFunc("GET /api/me", s.requireAuth(s.handleMe))
	mux.HandleFunc("GET /api/employees", s.requireAuth(s.handleListEmployees))
	mux.HandleFunc("GET /api/departments", s.requireAuth(s.handleListDepartments))
	mux.HandleFunc("POST /api/departments", s.requireRole(employees.RoleHRAdmin, s.handleCreateDepartment))
	mux.HandleFunc("POST /api/employees", s.requireRole(employees.RoleHRAdmin, s.handleCreateEmployee))
	mux.HandleFunc("GET /api/leave-types", s.requireAuth(s.handleListLeaveTypes))
	mux.HandleFunc("POST /api/leave-types", s.requireRole(employees.RoleHRAdmin, s.handleCreateLeaveType))
	mux.HandleFunc("POST /api/leave-requests", s.requireAuth(s.handleCreateLeaveRequest))
	mux.HandleFunc("GET /api/leave-requests", s.requireAuth(s.handleListLeaveRequests))
	mux.HandleFunc("POST /api/leave-requests/{id}/decide", s.requireAuth(s.handleDecideLeaveRequest))
	mux.HandleFunc("GET /api/leave-balances", s.requireAuth(s.handleLeaveBalances))
	mux.HandleFunc("GET /api/leave-calendar", s.requireAuth(s.handleLeaveCalendar))
	mux.HandleFunc("GET /api/employees/{id}/salary-structure", s.requireAuth(s.handleEmployeeStructure))
	mux.HandleFunc("POST /api/payroll/run", s.requireRole(employees.RoleHRAdmin, s.handleRunPayroll))
	mux.HandleFunc("GET /api/payroll/runs", s.requireRole(employees.RoleHRAdmin, s.handleListPayrollRuns))
	mux.HandleFunc("GET /api/payslips", s.requireAuth(s.handleListPayslips))
	mux.HandleFunc("GET /api/payslips/{id}/deductions", s.requireAuth(s.handlePayslipDeductions))
	mux.HandleFunc("GET /api/payslips/{id}/components", s.requireAuth(s.handlePayslipComponents))
	mux.HandleFunc("POST /api/salary-components", s.requireRole(employees.RoleHRAdmin, s.handleCreateSalaryComponent))
	mux.HandleFunc("GET /api/salary-components", s.requireRole(employees.RoleHRAdmin, s.handleListSalaryComponents))
	mux.HandleFunc("DELETE /api/salary-components/{id}", s.requireRole(employees.RoleHRAdmin, s.handleDeleteSalaryComponent))
	mux.HandleFunc("POST /api/employee-salary-components", s.requireRole(employees.RoleHRAdmin, s.handleSetEmployeeSalaryComponent))
	mux.HandleFunc("GET /api/dashboard", s.requireAuth(s.handleDashboard))

	return s.withCORS(mux)
}
