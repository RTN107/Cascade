CREATE TABLE salary_structures (
    employee_id BIGINT PRIMARY KEY REFERENCES employees(id),
    base        NUMERIC NOT NULL,
    allowances  NUMERIC NOT NULL DEFAULT 0
);

CREATE TABLE payroll_runs (
    id           BIGSERIAL PRIMARY KEY,
    period_month INTEGER NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    period_year  INTEGER NOT NULL,
    run_by       BIGINT NOT NULL REFERENCES employees(id),
    run_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (period_month, period_year)
);

CREATE TABLE payslips (
    id             BIGSERIAL PRIMARY KEY,
    payroll_run_id BIGINT NOT NULL REFERENCES payroll_runs(id),
    employee_id    BIGINT NOT NULL REFERENCES employees(id),
    base           NUMERIC NOT NULL,
    allowances     NUMERIC NOT NULL,
    lop_deduction  NUMERIC NOT NULL,
    net_pay        NUMERIC NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payslip_deduction_lines (
    id               BIGSERIAL PRIMARY KEY,
    payslip_id       BIGINT NOT NULL REFERENCES payslips(id),
    leave_request_id BIGINT REFERENCES leave_requests(id),
    amount           NUMERIC NOT NULL,
    description      TEXT NOT NULL
);

-- tracks which LOP flags have already been billed, so a later run never double-deducts
-- the same leave request
ALTER TABLE leave_requests ADD COLUMN consumed_by_payroll_run_id BIGINT REFERENCES payroll_runs(id);

CREATE INDEX payslips_employee_idx ON payslips(employee_id);
CREATE INDEX payslip_deduction_lines_payslip_idx ON payslip_deduction_lines(payslip_id);
