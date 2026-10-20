package employees

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleHRAdmin  Role = "hr_admin"
	// RoleViewer sees exactly what HR Admin sees; every write is blocked at the
	// middleware regardless of which handler it would otherwise reach.
	RoleViewer Role = "viewer"
)

type Employee struct {
	ID           int64
	Email        string
	PasswordHash string
	FullName     string
	Title        *string
	Role         Role
	DepartmentID *int64
	ManagerID    *int64
	Active       bool
}

type Department struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Count excludes the read-only viewer login: it is not a real employee, so it must
// never show up in a headcount HR Admin reads off the dashboard.
func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM employees WHERE role != 'viewer'`).Scan(&n)
	return n, err
}

func (s *Store) CountDepartments(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM departments`).Scan(&n)
	return n, err
}

const employeeColumns = `id, email, password_hash, full_name, title, role, department_id, manager_id, active`

func scanEmployee(row pgx.Row) (*Employee, error) {
	var e Employee
	err := row.Scan(&e.ID, &e.Email, &e.PasswordHash, &e.FullName, &e.Title, &e.Role, &e.DepartmentID, &e.ManagerID, &e.Active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (s *Store) ByEmail(ctx context.Context, email string) (*Employee, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+employeeColumns+` FROM employees WHERE email = $1`, email)
	return scanEmployee(row)
}

func (s *Store) ByID(ctx context.Context, id int64) (*Employee, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+employeeColumns+` FROM employees WHERE id = $1`, id)
	return scanEmployee(row)
}

func (s *Store) Create(ctx context.Context, e Employee) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO employees (email, password_hash, full_name, title, role, department_id, manager_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		e.Email, e.PasswordHash, e.FullName, e.Title, e.Role, e.DepartmentID, e.ManagerID).Scan(&id)
	return id, err
}

func (s *Store) CreateDepartment(ctx context.Context, name string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO departments (name) VALUES ($1) RETURNING id`, name).Scan(&id)
	return id, err
}

func (s *Store) ListDepartments(ctx context.Context) ([]Department, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name FROM departments ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Department{}
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func scanEmployees(rows pgx.Rows) ([]Employee, error) {
	defer rows.Close()
	out := []Employee{}
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.Email, &e.PasswordHash, &e.FullName, &e.Title, &e.Role, &e.DepartmentID, &e.ManagerID, &e.Active); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Visible returns exactly the employees the caller is allowed to see: HR Admin sees
// everyone, a Manager sees their own department, an Employee sees their own chain of
// managers upward and their own reports downward, computed live from manager_id.
func (s *Store) Visible(ctx context.Context, caller Employee) ([]Employee, error) {
	switch caller.Role {
	case RoleHRAdmin, RoleViewer:
		// the viewer login itself is not a real employee, so it never appears in a
		// list meant to show the org as it actually is
		rows, err := s.pool.Query(ctx, `SELECT `+employeeColumns+` FROM employees WHERE role != 'viewer' ORDER BY full_name`)
		if err != nil {
			return nil, err
		}
		return scanEmployees(rows)
	case RoleManager:
		rows, err := s.pool.Query(ctx, `SELECT `+employeeColumns+` FROM employees WHERE department_id = $1 ORDER BY full_name`, caller.DepartmentID)
		if err != nil {
			return nil, err
		}
		return scanEmployees(rows)
	default:
		rows, err := s.pool.Query(ctx, `
			WITH RECURSIVE up AS (
				SELECT id, manager_id FROM employees WHERE id = $1
				UNION ALL
				SELECT e.id, e.manager_id FROM employees e JOIN up ON e.id = up.manager_id
			),
			down AS (
				SELECT id FROM employees WHERE id = $1
				UNION ALL
				SELECT e.id FROM employees e JOIN down ON e.manager_id = down.id
			)
			SELECT `+employeeColumns+` FROM employees
			WHERE id IN (SELECT id FROM up UNION SELECT id FROM down)
			ORDER BY full_name`, caller.ID)
		if err != nil {
			return nil, err
		}
		return scanEmployees(rows)
	}
}
