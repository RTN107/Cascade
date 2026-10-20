package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"cascade/internal/employees"
)

type createLeaveTypeRequest struct {
	Name              string  `json:"name"`
	AnnualBalanceDays float64 `json:"annual_balance_days"`
}

func (s *Server) handleCreateLeaveType(w http.ResponseWriter, r *http.Request) {
	var req createLeaveTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	id, err := s.leave.CreateType(r.Context(), req.Name, req.AnnualBalanceDays)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := callerFrom(r).ID
	if err := s.audit.Record(r.Context(), &actor, "create", "leave_type", id, req); err != nil {
		log.Println("audit write failed:", err)
	}
	writeJSON(w, map[string]int64{"id": id})
}

func (s *Server) handleListLeaveTypes(w http.ResponseWriter, r *http.Request) {
	list, err := s.leave.ListTypes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

type createLeaveRequestRequest struct {
	LeaveTypeID int64  `json:"leave_type_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

func (s *Server) handleCreateLeaveRequest(w http.ResponseWriter, r *http.Request) {
	var req createLeaveRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		http.Error(w, "invalid start_date", http.StatusBadRequest)
		return
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		http.Error(w, "invalid end_date", http.StatusBadRequest)
		return
	}
	if end.Before(start) {
		http.Error(w, "end_date must not be before start_date", http.StatusBadRequest)
		return
	}
	caller := callerFrom(r)
	id, days, err := s.leave.Create(r.Context(), caller.ID, req.LeaveTypeID, start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.audit.Record(r.Context(), &caller.ID, "create", "leave_request", id, map[string]any{"days": days}); err != nil {
		log.Println("audit write failed:", err)
	}
	writeJSON(w, map[string]any{"id": id, "days": days})
}

func (s *Server) handleListLeaveRequests(w http.ResponseWriter, r *http.Request) {
	caller := callerFrom(r)
	list, err := s.leave.ListForCaller(r.Context(), caller.ID, string(caller.Role))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

type decideLeaveRequestBody struct {
	Approve bool `json:"approve"`
}

func (s *Server) handleDecideLeaveRequest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req decideLeaveRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	lr, err := s.leave.ByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if lr == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	requester, err := s.employees.ByID(r.Context(), lr.EmployeeID)
	if err != nil || requester == nil {
		http.Error(w, "requester not found", http.StatusInternalServerError)
		return
	}
	caller := callerFrom(r)
	// An approver is HR Admin, or the requester's current manager, resolved live, the
	// same rule as hierarchy visibility: whoever manages this person right now decides.
	isApprover := caller.Role == employees.RoleHRAdmin ||
		(caller.Role == employees.RoleManager && requester.ManagerID != nil && *requester.ManagerID == caller.ID)
	if !isApprover {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	lopDays, err := s.leave.Decide(r.Context(), id, caller.ID, req.Approve)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	action := "reject"
	if req.Approve {
		action = "approve"
	}
	isLOP := lopDays > 0
	if err := s.audit.Record(r.Context(), &caller.ID, action, "leave_request", id, map[string]any{"is_lop": isLOP, "lop_days": lopDays}); err != nil {
		log.Println("audit write failed:", err)
	}
	writeJSON(w, map[string]any{"status": action, "is_lop": isLOP, "lop_days": lopDays})
}

func (s *Server) handleLeaveBalances(w http.ResponseWriter, r *http.Request) {
	caller := callerFrom(r)
	balances, err := s.leave.Balances(r.Context(), caller.ID, time.Now().Year())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, balances)
}

func (s *Server) handleLeaveCalendar(w http.ResponseWriter, r *http.Request) {
	caller := callerFrom(r)
	visible, err := s.employees.Visible(r.Context(), caller)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ids := make([]int64, len(visible))
	for i, e := range visible {
		ids[i] = e.ID
	}
	entries, err := s.leave.CalendarForEmployees(r.Context(), ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, entries)
}
