package payroll

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Run struct {
	ID          int64     `json:"id"`
	PeriodMonth int       `json:"period_month"`
	PeriodYear  int       `json:"period_year"`
	RunBy       int64     `json:"run_by"`
	RunAt       time.Time `json:"run_at"`
}

type Payslip struct {
	ID              int64     `json:"id"`
	PayrollRunID    int64     `json:"payroll_run_id"`
	PeriodMonth     int       `json:"period_month"`
	PeriodYear      int       `json:"period_year"`
	EmployeeID      int64     `json:"employee_id"`
	GrossEarnings   float64   `json:"gross_earnings"`
	TotalDeductions float64   `json:"total_deductions"`
	LOPDeduction    float64   `json:"lop_deduction"`
	NetPay          float64   `json:"net_pay"`
	CreatedAt       time.Time `json:"created_at"`
}

type DeductionLine struct {
	ID             int64   `json:"id"`
	PayslipID      int64   `json:"payslip_id"`
	LeaveRequestID *int64  `json:"leave_request_id"`
	Amount         float64 `json:"amount"`
	Description    string  `json:"description"`
	LeaveStartDate *string `json:"leave_start_date"`
	LeaveEndDate   *string `json:"leave_end_date"`
}

type lopItem struct {
	ID            int64
	LOPDays       float64
	RequestedDays float64
	StartDate     time.Time
	EndDate       time.Time
}

// Component is a named pay item HR Admin defines once, Basic Pay, HRA, Provident Fund,
// whatever the organization needs. Kind groups it under Earnings or Deductions. Exactly
// one component is ever marked IsBase, the anchor the Loss of Pay daily rate is computed
// from, seeded once by migration and never created through the API.
type Component struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	IsBase bool   `json:"is_base"`
}

// StructureLine is one row of an employee's full pay structure: every component that
// exists, paired with this employee's amount for it, or null if it has not been assigned
// yet. This is the one place that both shows and edits what an employee is paid.
type StructureLine struct {
	ComponentID int64    `json:"component_id"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	IsBase      bool     `json:"is_base"`
	Amount      *float64 `json:"amount"`
}

type ComponentLine struct {
	ID          int64   `json:"id"`
	PayslipID   int64   `json:"payslip_id"`
	ComponentID *int64  `json:"component_id"`
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	Amount      float64 `json:"amount"`
}

type componentAmount struct {
	ID     int64
	Name   string
	Kind   string
	IsBase bool
	Amount float64
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateComponent(ctx context.Context, name, kind string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO salary_components (name, kind) VALUES ($1,$2) RETURNING id`, name, kind).Scan(&id)
	return id, err
}

// DeleteComponent removes a pay component HR Admin no longer wants to offer. The base
// component can never be removed, payroll depends on every employee having one. Existing
// payslip lines keep their own copy of the name and amount, so past payslips read the same
// after removal; only the live, go-forward assignment disappears.
func (s *Store) DeleteComponent(ctx context.Context, id int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var isBase bool
	err = tx.QueryRow(ctx, `SELECT is_base FROM salary_components WHERE id=$1`, id).Scan(&isBase)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if isBase {
		return errors.New("the base component cannot be removed")
	}

	if _, err := tx.Exec(ctx, `DELETE FROM employee_salary_components WHERE component_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE payslip_component_lines SET component_id=NULL WHERE component_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM salary_components WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListComponents(ctx context.Context) ([]Component, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, kind, is_base FROM salary_components ORDER BY is_base DESC, kind, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Component{}
	for rows.Next() {
		var c Component
		if err := rows.Scan(&c.ID, &c.Name, &c.Kind, &c.IsBase); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) SetEmployeeComponent(ctx context.Context, employeeID, componentID int64, amount float64) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO employee_salary_components (employee_id, component_id, amount)
		VALUES ($1,$2,$3)
		ON CONFLICT (employee_id, component_id) DO UPDATE SET amount = $3`, employeeID, componentID, amount)
	return err
}

// EmployeeStructure returns every component in the system paired with this employee's
// amount, or a null amount if it has not been assigned yet, so the caller can tell assign
// from update without a second lookup.
func (s *Store) EmployeeStructure(ctx context.Context, employeeID int64) ([]StructureLine, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sc.id, sc.name, sc.kind, sc.is_base, esc.amount
		FROM salary_components sc
		LEFT JOIN employee_salary_components esc ON esc.component_id = sc.id AND esc.employee_id = $1
		ORDER BY sc.is_base DESC, sc.kind, sc.name`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StructureLine{}
	for rows.Next() {
		var line StructureLine
		if err := rows.Scan(&line.ComponentID, &line.Name, &line.Kind, &line.IsBase, &line.Amount); err != nil {
			return nil, err
		}
		out = append(out, line)
	}
	return out, rows.Err()
}

// employeeComponents returns this employee's configured pay components, inside the run's
// transaction, so the run sees a consistent snapshot of what's configured right now.
func employeeComponents(ctx context.Context, tx pgx.Tx, employeeID int64) ([]componentAmount, error) {
	rows, err := tx.Query(ctx, `
		SELECT sc.id, sc.name, sc.kind, sc.is_base, esc.amount
		FROM employee_salary_components esc
		JOIN salary_components sc ON sc.id = esc.component_id
		WHERE esc.employee_id = $1`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []componentAmount{}
	for rows.Next() {
		var c componentAmount
		if err := rows.Scan(&c.ID, &c.Name, &c.Kind, &c.IsBase, &c.Amount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// unconsumedLOP returns this employee's approved LOP leave requests that have not yet
// been billed in any payroll run, so a later run never double-deducts the same leave.
func unconsumedLOP(ctx context.Context, tx pgx.Tx, employeeID int64) ([]lopItem, error) {
	rows, err := tx.Query(ctx, `SELECT id, lop_days, days, start_date, end_date FROM leave_requests
		WHERE employee_id=$1 AND is_lop=true AND status='approved' AND consumed_by_payroll_run_id IS NULL`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []lopItem{}
	for rows.Next() {
		var item lopItem
		if err := rows.Scan(&item.ID, &item.LOPDays, &item.RequestedDays, &item.StartDate, &item.EndDate); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// Run processes every active employee in one transaction. If any employee has no amount
// set for Basic Pay, nothing is committed at all, that is the all-or-nothing guarantee.
func (s *Store) Run(ctx context.Context, periodMonth, periodYear int, runBy int64) (int64, int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx)

	var runID int64
	err = tx.QueryRow(ctx, `INSERT INTO payroll_runs (period_month, period_year, run_by) VALUES ($1,$2,$3) RETURNING id`,
		periodMonth, periodYear, runBy).Scan(&runID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, 0, fmt.Errorf("payroll has already been run for %d/%d", periodMonth, periodYear)
		}
		return 0, 0, err
	}

	rows, err := tx.Query(ctx, `SELECT id FROM employees WHERE active = true ORDER BY id`)
	if err != nil {
		return 0, 0, err
	}
	var employeeIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, 0, err
		}
		employeeIDs = append(employeeIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	for _, empID := range employeeIDs {
		components, err := employeeComponents(ctx, tx, empID)
		if err != nil {
			return 0, 0, err
		}

		var basicPay, grossEarnings, totalDeductions float64
		var hasBasicPay bool
		for _, c := range components {
			if c.IsBase {
				basicPay = c.Amount
				hasBasicPay = true
			}
			if c.Kind == "earning" {
				grossEarnings += c.Amount
			} else {
				totalDeductions += c.Amount
			}
		}
		if !hasBasicPay {
			return 0, 0, fmt.Errorf("employee %d has no Basic Pay set, run aborted", empID)
		}

		lopRequests, err := unconsumedLOP(ctx, tx, empID)
		if err != nil {
			return 0, 0, err
		}
		var lopDays float64
		for _, lr := range lopRequests {
			lopDays += lr.LOPDays
		}

		lopDeduction, netPay := CalculateNetPay(basicPay, grossEarnings, totalDeductions, lopDays)

		var payslipID int64
		err = tx.QueryRow(ctx, `INSERT INTO payslips (payroll_run_id, employee_id, gross_earnings, total_deductions, lop_deduction, net_pay)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			runID, empID, grossEarnings, totalDeductions, lopDeduction, netPay).Scan(&payslipID)
		if err != nil {
			return 0, 0, err
		}

		for _, c := range components {
			if _, err := tx.Exec(ctx, `INSERT INTO payslip_component_lines (payslip_id, component_id, name, kind, amount)
				VALUES ($1,$2,$3,$4,$5)`, payslipID, c.ID, c.Name, c.Kind, c.Amount); err != nil {
				return 0, 0, err
			}
		}

		for _, lr := range lopRequests {
			lineAmount, _ := CalculateNetPay(basicPay, 0, 0, lr.LOPDays)
			desc := fmt.Sprintf("Loss of pay: %.0f of %.0f day(s) requested, %s to %s", lr.LOPDays, lr.RequestedDays,
				lr.StartDate.Format("2006-01-02"), lr.EndDate.Format("2006-01-02"))
			if _, err := tx.Exec(ctx, `INSERT INTO payslip_deduction_lines (payslip_id, leave_request_id, amount, description)
				VALUES ($1,$2,$3,$4)`, payslipID, lr.ID, lineAmount, desc); err != nil {
				return 0, 0, err
			}
			if _, err := tx.Exec(ctx, `UPDATE leave_requests SET consumed_by_payroll_run_id=$1 WHERE id=$2`, runID, lr.ID); err != nil {
				return 0, 0, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return runID, len(employeeIDs), nil
}

func (s *Store) TotalNetPay(ctx context.Context, runID int64) (float64, error) {
	var total float64
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(SUM(net_pay),0) FROM payslips WHERE payroll_run_id=$1`, runID).Scan(&total)
	return total, err
}

func (s *Store) ListRuns(ctx context.Context) ([]Run, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, period_month, period_year, run_by, run_at FROM payroll_runs
		ORDER BY period_year DESC, period_month DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.PeriodMonth, &r.PeriodYear, &r.RunBy, &r.RunAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) PayslipsForCaller(ctx context.Context, callerID int64, isHRAdmin bool) ([]Payslip, error) {
	query := `SELECT p.id, p.payroll_run_id, r.period_month, r.period_year, p.employee_id, p.gross_earnings, p.total_deductions, p.lop_deduction, p.net_pay, p.created_at
		FROM payslips p JOIN payroll_runs r ON r.id = p.payroll_run_id`
	args := []any{}
	if !isHRAdmin {
		query += ` WHERE p.employee_id = $1`
		args = append(args, callerID)
	}
	query += ` ORDER BY r.period_year DESC, r.period_month DESC, p.employee_id`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Payslip{}
	for rows.Next() {
		var p Payslip
		if err := rows.Scan(&p.ID, &p.PayrollRunID, &p.PeriodMonth, &p.PeriodYear, &p.EmployeeID, &p.GrossEarnings, &p.TotalDeductions, &p.LOPDeduction, &p.NetPay, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) PayslipByID(ctx context.Context, id int64) (*Payslip, error) {
	var p Payslip
	err := s.pool.QueryRow(ctx, `SELECT id, payroll_run_id, employee_id, gross_earnings, total_deductions, lop_deduction, net_pay, created_at
		FROM payslips WHERE id=$1`, id).
		Scan(&p.ID, &p.PayrollRunID, &p.EmployeeID, &p.GrossEarnings, &p.TotalDeductions, &p.LOPDeduction, &p.NetPay, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *Store) ComponentLines(ctx context.Context, payslipID int64) ([]ComponentLine, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, payslip_id, component_id, name, kind, amount
		FROM payslip_component_lines WHERE payslip_id = $1 ORDER BY kind, name`, payslipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ComponentLine{}
	for rows.Next() {
		var c ComponentLine
		if err := rows.Scan(&c.ID, &c.PayslipID, &c.ComponentID, &c.Name, &c.Kind, &c.Amount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) DeductionLines(ctx context.Context, payslipID int64) ([]DeductionLine, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT dl.id, dl.payslip_id, dl.leave_request_id, dl.amount, dl.description, lr.start_date, lr.end_date
		FROM payslip_deduction_lines dl
		LEFT JOIN leave_requests lr ON lr.id = dl.leave_request_id
		WHERE dl.payslip_id = $1`, payslipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeductionLine{}
	for rows.Next() {
		var d DeductionLine
		var start, end *time.Time
		if err := rows.Scan(&d.ID, &d.PayslipID, &d.LeaveRequestID, &d.Amount, &d.Description, &start, &end); err != nil {
			return nil, err
		}
		if start != nil {
			formatted := start.Format("2006-01-02")
			d.LeaveStartDate = &formatted
		}
		if end != nil {
			formatted := end.Format("2006-01-02")
			d.LeaveEndDate = &formatted
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
