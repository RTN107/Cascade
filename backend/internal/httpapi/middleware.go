package httpapi

import (
	"context"
	"net/http"

	"cascade/internal/auth"
	"cascade/internal/employees"
)

type contextKey string

const employeeContextKey contextKey = "employee"

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		claims, err := auth.ParseToken(s.jwtSecret, cookie.Value)
		if err != nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		emp, err := s.employees.ByID(r.Context(), claims.EmployeeID)
		if err != nil || emp == nil {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		// the viewer login is read only everywhere, so no handler ever has to
		// remember to check for it itself
		if emp.Role == employees.RoleViewer && r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "this login is read only", http.StatusForbidden)
			return
		}
		ctx := context.WithValue(r.Context(), employeeContextKey, *emp)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) requireRole(role employees.Role, next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		caller := callerFrom(r)
		// a GET the viewer reaches here on is always one of HR Admin's own read
		// screens; any write would already have been stopped above
		sameAccessAsAdmin := role == employees.RoleHRAdmin && caller.Role == employees.RoleViewer
		if caller.Role != role && !sameAccessAsAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

func callerFrom(r *http.Request) employees.Employee {
	return r.Context().Value(employeeContextKey).(employees.Employee)
}

// withCORS allows the frontend, on its own port, to call this API with cookies attached.
// The origin must be named exactly; credentials cannot be sent to a wildcard origin.
func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.webOrigin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
