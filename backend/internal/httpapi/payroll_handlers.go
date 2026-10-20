package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"cascade/internal/employees"
)

// handleEmployeeStructure is the one place a pay structure is read: HR Admin can look up
// anyone, everyone else can only look up their own.
func (s *Server) handleEmployeeStructure(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	caller := callerFrom(r)
	if caller.Role != employees.RoleHRAdmin && caller.Role != employees.RoleViewer && id != caller.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	lines, err := s.payroll.EmployeeStructure(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, lines)
}

type runPayrollRequest struct {
	PeriodMonth int `json:"period_month"`
	PeriodYear  int `json:"period_year"`
}

func (s *Server) handleRunPayroll(w http.ResponseWriter, r *http.Request) {
	var req runPayrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	caller := callerFrom(r)
	runID, count, err := s.payroll.Run(r.Context(), req.PeriodMonth, req.PeriodYear, caller.ID)
	if err != nil {
		// all-or-nothing: nothing committed, surface exactly why so HR can fix it and retry
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err := s.audit.Record(r.Context(), &caller.ID, "run", "payroll_run", runID,
		map[string]any{"period_month": req.PeriodMonth, "period_year": req.PeriodYear, "payslip_count": count}); err != nil {
		log.Println("audit write failed:", err)
	}
	writeJSON(w, map[string]any{"id": runID, "payslip_count": count})
}

func (s *Server) handleListPayrollRuns(w http.ResponseWriter, r *http.Request) {
	list, err := s.payroll.ListRuns(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (s *Server) handleListPayslips(w http.ResponseWriter, r *http.Request) {
	caller := callerFrom(r)
	seesAll := caller.Role == employees.RoleHRAdmin || caller.Role == employees.RoleViewer
	list, err := s.payroll.PayslipsForCaller(r.Context(), caller.ID, seesAll)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (s *Server) handlePayslipDeductions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	payslip, err := s.payroll.PayslipByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if payslip == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	caller := callerFrom(r)
	if caller.Role != employees.RoleHRAdmin && caller.Role != employees.RoleViewer && payslip.EmployeeID != caller.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	lines, err := s.payroll.DeductionLines(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, lines)
}

func (s *Server) handlePayslipComponents(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	payslip, err := s.payroll.PayslipByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if payslip == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	caller := callerFrom(r)
	if caller.Role != employees.RoleHRAdmin && caller.Role != employees.RoleViewer && payslip.EmployeeID != caller.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	lines, err := s.payroll.ComponentLines(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, lines)
}

type createSalaryComponentRequest struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func (s *Server) handleCreateSalaryComponent(w http.ResponseWriter, r *http.Request) {
	var req createSalaryComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Kind != "earning" && req.Kind != "deduction" {
		http.Error(w, "kind must be earning or deduction", http.StatusBadRequest)
		return
	}
	id, err := s.payroll.CreateComponent(r.Context(), req.Name, req.Kind)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := callerFrom(r).ID
	if err := s.audit.Record(r.Context(), &actor, "create", "salary_component", id, req); err != nil {
		log.Println("audit write failed:", err)
	}
	writeJSON(w, map[string]int64{"id": id})
}

func (s *Server) handleListSalaryComponents(w http.ResponseWriter, r *http.Request) {
	list, err := s.payroll.ListComponents(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (s *Server) handleDeleteSalaryComponent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.payroll.DeleteComponent(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	actor := callerFrom(r).ID
	if err := s.audit.Record(r.Context(), &actor, "delete", "salary_component", id, nil); err != nil {
		log.Println("audit write failed:", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

type setEmployeeComponentRequest struct {
	EmployeeID  int64   `json:"employee_id"`
	ComponentID int64   `json:"component_id"`
	Amount      float64 `json:"amount"`
}

func (s *Server) handleSetEmployeeSalaryComponent(w http.ResponseWriter, r *http.Request) {
	var req setEmployeeComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := s.payroll.SetEmployeeComponent(r.Context(), req.EmployeeID, req.ComponentID, req.Amount); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := callerFrom(r).ID
	if err := s.audit.Record(r.Context(), &actor, "set", "employee_salary_component", req.EmployeeID, req); err != nil {
		log.Println("audit write failed:", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

