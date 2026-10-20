package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cascade/internal/auth"
	"cascade/internal/employees"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	if err := s.employees.Ping(r.Context()); err != nil {
		status = "database unreachable"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	writeJSON(w, map[string]string{"status": status})
}

func (s *Server) handleSetupNeeded(w http.ResponseWriter, r *http.Request) {
	n, err := s.employees.Count(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"needed": n == 0})
}

type setupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	n, err := s.employees.Count(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Error(w, "setup already completed", http.StatusConflict)
		return
	}
	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, err := s.employees.Create(r.Context(), employees.Employee{
		Email: req.Email, PasswordHash: hash, FullName: req.FullName, Role: employees.RoleHRAdmin, Active: true,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.audit.Record(r.Context(), nil, "bootstrap", "employee", id, map[string]string{"email": req.Email}); err != nil {
		log.Println("audit write failed:", err)
	}
	s.setSession(w, id, string(employees.RoleHRAdmin))
	writeJSON(w, map[string]any{"id": id})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	emp, err := s.employees.ByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if emp == nil || !auth.CheckPassword(emp.PasswordHash, req.Password) {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}
	s.setSession(w, emp.ID, string(emp.Role))
	writeJSON(w, publicEmployee(*emp))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1, SameSite: http.SameSiteNoneMode, Secure: true})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, publicEmployee(callerFrom(r)))
}

func (s *Server) handleListEmployees(w http.ResponseWriter, r *http.Request) {
	list, err := s.employees.Visible(r.Context(), callerFrom(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, len(list))
	for i, e := range list {
		out[i] = publicEmployee(e)
	}
	writeJSON(w, out)
}

func (s *Server) handleListDepartments(w http.ResponseWriter, r *http.Request) {
	list, err := s.employees.ListDepartments(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

type createDepartmentRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req createDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	id, err := s.employees.CreateDepartment(r.Context(), req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := callerFrom(r).ID
	if err := s.audit.Record(r.Context(), &actor, "create", "department", id, req); err != nil {
		log.Println("audit write failed:", err)
	}
	writeJSON(w, map[string]int64{"id": id})
}

type createEmployeeRequest struct {
	Email        string  `json:"email"`
	Password     string  `json:"password"`
	FullName     string  `json:"full_name"`
	Title        *string `json:"title"`
	Role         string  `json:"role"`
	DepartmentID *int64  `json:"department_id"`
	ManagerID    *int64  `json:"manager_id"`
}

func (s *Server) handleCreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req createEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, err := s.employees.Create(r.Context(), employees.Employee{
		Email: req.Email, PasswordHash: hash, FullName: req.FullName, Title: req.Title,
		Role: employees.Role(req.Role), DepartmentID: req.DepartmentID, ManagerID: req.ManagerID, Active: true,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := callerFrom(r).ID
	if err := s.audit.Record(r.Context(), &actor, "create", "employee", id, map[string]string{"email": req.Email}); err != nil {
		log.Println("audit write failed:", err)
	}
	writeJSON(w, map[string]int64{"id": id})
}

func (s *Server) setSession(w http.ResponseWriter, employeeID int64, role string) {
	token, err := auth.IssueToken(s.jwtSecret, employeeID, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		// a deployed frontend and API live on two different sites, so the browser
		// treats every request between them as cross-site; only SameSite=None,
		// which requires Secure, is ever sent back in that case
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		Expires:  time.Now().Add(24 * time.Hour),
	})
}

func publicEmployee(e employees.Employee) map[string]any {
	return map[string]any{
		"id": e.ID, "email": e.Email, "full_name": e.FullName, "title": e.Title, "role": e.Role,
		"department_id": e.DepartmentID, "manager_id": e.ManagerID,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
