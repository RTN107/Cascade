package httpapi

import (
	"net/http"
	"time"

	"cascade/internal/employees"
)

// handleDashboard is a pure read over data every other feature already owns: it stores
// nothing of its own, just different slices of the same tables depending on who's asking.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	caller := callerFrom(r)
	ctx := r.Context()

	switch caller.Role {
	case employees.RoleHRAdmin, employees.RoleViewer:
		totalEmployees, err := s.employees.Count(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		totalDepartments, err := s.employees.CountDepartments(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		allRequests, err := s.leave.ListForCaller(ctx, caller.ID, string(caller.Role))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pending := 0
		statusCounts := map[string]int{"pending": 0, "approved": 0, "rejected": 0}
		for _, req := range allRequests {
			statusCounts[req.Status]++
			if req.Status == "pending" {
				pending++
			}
		}
		outstandingLOP, err := s.leave.CountUnconsumedLOP(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		allEmployees, err := s.employees.Visible(ctx, caller)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		departments, err := s.employees.ListDepartments(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		deptCounts := make(map[int64]int)
		for _, e := range allEmployees {
			if e.DepartmentID != nil {
				deptCounts[*e.DepartmentID]++
			}
		}
		departmentBreakdown := make([]map[string]any, 0, len(departments))
		for _, d := range departments {
			departmentBreakdown = append(departmentBreakdown, map[string]any{"name": d.Name, "count": deptCounts[d.ID]})
		}
		runs, err := s.payroll.ListRuns(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var lastRun map[string]any
		if len(runs) > 0 {
			total, err := s.payroll.TotalNetPay(ctx, runs[0].ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			lastRun = map[string]any{
				"period_month":  runs[0].PeriodMonth,
				"period_year":   runs[0].PeriodYear,
				"total_net_pay": total,
			}
		}
		writeJSON(w, map[string]any{
			"role":                   caller.Role,
			"total_employees":        totalEmployees,
			"total_departments":      totalDepartments,
			"pending_leave_requests": pending,
			"outstanding_lop_flags":  outstandingLOP,
			"last_run":               lastRun,
			"leave_status_breakdown": statusCounts,
			"department_breakdown":   departmentBreakdown,
		})

	case employees.RoleManager:
		visible, err := s.employees.Visible(ctx, caller)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		requests, err := s.leave.ListForCaller(ctx, caller.ID, string(caller.Role))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pendingApprovals := 0
		for _, req := range requests {
			if req.Status == "pending" && req.EmployeeID != caller.ID {
				pendingApprovals++
			}
		}
		writeJSON(w, map[string]any{
			"role":              caller.Role,
			"team_size":         len(visible),
			"pending_approvals": pendingApprovals,
		})

	default:
		requests, err := s.leave.ListForCaller(ctx, caller.ID, string(caller.Role))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pending := 0
		for _, req := range requests {
			if req.Status == "pending" {
				pending++
			}
		}
		balances, err := s.leave.Balances(ctx, caller.ID, time.Now().Year())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var totalRemaining float64
		for _, b := range balances {
			totalRemaining += b.RemainingDays
		}
		payslips, err := s.payroll.PayslipsForCaller(ctx, caller.ID, false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var latestNetPay *float64
		var latestNetPayMonth, latestNetPayYear *int
		if len(payslips) > 0 {
			latestNetPay = &payslips[0].NetPay
			latestNetPayMonth = &payslips[0].PeriodMonth
			latestNetPayYear = &payslips[0].PeriodYear
		}
		writeJSON(w, map[string]any{
			"role":                   caller.Role,
			"pending_leave_requests": pending,
			"total_remaining_days":   totalRemaining,
			"latest_net_pay":         latestNetPay,
			"latest_net_pay_month":   latestNetPayMonth,
			"latest_net_pay_year":    latestNetPayYear,
		})
	}
}
