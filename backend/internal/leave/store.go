package leave

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeaveType struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	AnnualBalanceDays float64 `json:"annual_balance_days"`
}

type Request struct {
	ID          int64      `json:"id"`
	EmployeeID  int64      `json:"employee_id"`
	LeaveTypeID int64      `json:"leave_type_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     time.Time  `json:"end_date"`
	Days        float64    `json:"days"`
	Status      string     `json:"status"`
	IsLOP       bool       `json:"is_lop"`
	LOPDays     float64    `json:"lop_days"`
	DecidedBy   *int64     `json:"decided_by"`
	DecidedAt   *time.Time `json:"decided_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CalendarEntry is one approved leave span shown on the team calendar: who, what type,
// and when, for anyone the caller is allowed to see per the hierarchy.
type CalendarEntry struct {
	EmployeeID    int64  `json:"employee_id"`
	EmployeeName  string `json:"employee_name"`
	LeaveTypeName string `json:"leave_type_name"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	Days          float64 `json:"days"`
}

type Balance struct {
	LeaveTypeID       int64   `json:"leave_type_id"`
	LeaveTypeName     string  `json:"leave_type_name"`
	AnnualBalanceDays float64 `json:"annual_balance_days"`
	UsedDays          float64 `json:"used_days"`
	RemainingDays     float64 `json:"remaining_days"`
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateType(ctx context.Context, name string, annualDays float64) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO leave_types (name, annual_balance_days) VALUES ($1,$2) RETURNING id`,
		name, annualDays).Scan(&id)
	return id, err
}

func (s *Store) ListTypes(ctx context.Context) ([]LeaveType, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, annual_balance_days FROM leave_types ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LeaveType{}
	for rows.Next() {
		var t LeaveType
		if err := rows.Scan(&t.ID, &t.Name, &t.AnnualBalanceDays); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

const requestColumns = `id, employee_id, leave_type_id, start_date, end_date, days, status, is_lop, lop_days, decided_by, decided_at, created_at`

func scanRequest(row pgx.Row) (*Request, error) {
	var r Request
	err := row.Scan(&r.ID, &r.EmployeeID, &r.LeaveTypeID, &r.StartDate, &r.EndDate, &r.Days,
		&r.Status, &r.IsLOP, &r.LOPDays, &r.DecidedBy, &r.DecidedAt, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func scanRequests(rows pgx.Rows) ([]Request, error) {
	defer rows.Close()
	out := []Request{}
	for rows.Next() {
		var r Request
		if err := rows.Scan(&r.ID, &r.EmployeeID, &r.LeaveTypeID, &r.StartDate, &r.EndDate, &r.Days,
			&r.Status, &r.IsLOP, &r.LOPDays, &r.DecidedBy, &r.DecidedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) ByID(ctx context.Context, id int64) (*Request, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+requestColumns+` FROM leave_requests WHERE id=$1`, id)
	return scanRequest(row)
}

// Create inserts a request. days is end_date minus start_date inclusive of both ends.
func (s *Store) Create(ctx context.Context, employeeID, leaveTypeID int64, start, end time.Time) (int64, float64, error) {
	days := end.Sub(start).Hours()/24 + 1
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO leave_requests (employee_id, leave_type_id, start_date, end_date, days)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`, employeeID, leaveTypeID, start, end, days).Scan(&id)
	return id, days, err
}

// ListForCaller returns exactly the requests the caller is allowed to see: HR Admin sees
// all, a Manager sees their own requests plus their direct reports', an Employee sees
// only their own.
func (s *Store) ListForCaller(ctx context.Context, callerID int64, role string) ([]Request, error) {
	switch role {
	case "hr_admin", "viewer":
		rows, err := s.pool.Query(ctx, `SELECT `+requestColumns+` FROM leave_requests ORDER BY created_at DESC`)
		if err != nil {
			return nil, err
		}
		return scanRequests(rows)
	case "manager":
		rows, err := s.pool.Query(ctx, `SELECT `+requestColumns+` FROM leave_requests
			WHERE employee_id = $1 OR employee_id IN (SELECT id FROM employees WHERE manager_id = $1)
			ORDER BY created_at DESC`, callerID)
		if err != nil {
			return nil, err
		}
		return scanRequests(rows)
	default:
		rows, err := s.pool.Query(ctx, `SELECT `+requestColumns+` FROM leave_requests WHERE employee_id=$1 ORDER BY created_at DESC`, callerID)
		if err != nil {
			return nil, err
		}
		return scanRequests(rows)
	}
}

// Decide approves or rejects a request. On approval, the request's days are drawn first
// from its own leave type, then from whichever other types still have balance left,
// pooling every type into one combined cap for this employee's year; days beyond that
// pool become Loss of Pay (lop_days). Rejecting never touches any balance.
func (s *Store) Decide(ctx context.Context, requestID, deciderID int64, approve bool) (lopDays float64, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var employeeID, leaveTypeID int64
	var days float64
	var startDate time.Time
	err = tx.QueryRow(ctx, `SELECT employee_id, leave_type_id, days, start_date FROM leave_requests WHERE id=$1`, requestID).
		Scan(&employeeID, &leaveTypeID, &days, &startDate)
	if err != nil {
		return 0, err
	}

	status := "rejected"
	if approve {
		status = "approved"
		rows, err := tx.Query(ctx, `
			SELECT lt.id, lt.annual_balance_days - COALESCE((SELECT SUM(a.days) FROM leave_request_allocations a
				JOIN leave_requests lr ON lr.id = a.leave_request_id
				WHERE a.leave_type_id = lt.id AND lr.employee_id = $1 AND extract(year from lr.start_date) = $2), 0)
			FROM leave_types lt
			ORDER BY (lt.id = $3) DESC, lt.id`, employeeID, startDate.Year(), leaveTypeID)
		if err != nil {
			return 0, err
		}
		type remaining struct {
			typeID int64
			left   float64
		}
		var pool []remaining
		for rows.Next() {
			var rm remaining
			if err := rows.Scan(&rm.typeID, &rm.left); err != nil {
				rows.Close()
				return 0, err
			}
			pool = append(pool, rm)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return 0, err
		}

		uncovered := days
		for _, rm := range pool {
			if uncovered <= 0 || rm.left <= 0 {
				continue
			}
			take := rm.left
			if take > uncovered {
				take = uncovered
			}
			if _, err := tx.Exec(ctx, `INSERT INTO leave_request_allocations (leave_request_id, leave_type_id, days) VALUES ($1,$2,$3)`,
				requestID, rm.typeID, take); err != nil {
				return 0, err
			}
			uncovered -= take
		}
		lopDays = uncovered
	}

	isLOP := lopDays > 0
	if _, err := tx.Exec(ctx, `UPDATE leave_requests SET status=$1, is_lop=$2, lop_days=$3, decided_by=$4, decided_at=now() WHERE id=$5`,
		status, isLOP, lopDays, deciderID, requestID); err != nil {
		return 0, err
	}

	return lopDays, tx.Commit(ctx)
}

// CountUnconsumedLOP is a company-wide count of approved LOP flags not yet billed by
// any payroll run, used only to summarize on a dashboard.
func (s *Store) CountUnconsumedLOP(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM leave_requests WHERE is_lop=true AND consumed_by_payroll_run_id IS NULL`).Scan(&n)
	return n, err
}

func (s *Store) Balances(ctx context.Context, employeeID int64, year int) ([]Balance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT lt.id, lt.name, lt.annual_balance_days,
			COALESCE((SELECT SUM(a.days) FROM leave_request_allocations a
				JOIN leave_requests lr ON lr.id = a.leave_request_id
				WHERE a.leave_type_id=lt.id AND lr.employee_id=$1
				AND extract(year from lr.start_date)=$2), 0) AS used
		FROM leave_types lt ORDER BY lt.name`, employeeID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Balance{}
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.LeaveTypeID, &b.LeaveTypeName, &b.AnnualBalanceDays, &b.UsedDays); err != nil {
			return nil, err
		}
		b.RemainingDays = b.AnnualBalanceDays - b.UsedDays
		out = append(out, b)
	}
	return out, rows.Err()
}

// CalendarForEmployees returns every approved leave span, past or future, for the given
// employees, so the caller (anyone who can already see these people per the hierarchy)
// can tell who is out and when without opening each person's own leave history.
func (s *Store) CalendarForEmployees(ctx context.Context, employeeIDs []int64) ([]CalendarEntry, error) {
	if len(employeeIDs) == 0 {
		return []CalendarEntry{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT lr.employee_id, e.full_name, lt.name, lr.start_date, lr.end_date, lr.days
		FROM leave_requests lr
		JOIN employees e ON e.id = lr.employee_id
		JOIN leave_types lt ON lt.id = lr.leave_type_id
		WHERE lr.status = 'approved' AND lr.employee_id = ANY($1)
		ORDER BY lr.start_date`, employeeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CalendarEntry{}
	for rows.Next() {
		var c CalendarEntry
		var start, end time.Time
		if err := rows.Scan(&c.EmployeeID, &c.EmployeeName, &c.LeaveTypeName, &start, &end, &c.Days); err != nil {
			return nil, err
		}
		c.StartDate = start.Format("2006-01-02")
		c.EndDate = end.Format("2006-01-02")
		out = append(out, c)
	}
	return out, rows.Err()
}
