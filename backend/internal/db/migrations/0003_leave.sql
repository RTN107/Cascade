-- Balances and approver are never stored, both are computed from this table plus the
-- hierarchy in the employees table.
CREATE TYPE leave_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE leave_types (
    id                  BIGSERIAL PRIMARY KEY,
    name                TEXT NOT NULL UNIQUE,
    annual_balance_days NUMERIC NOT NULL
);

CREATE TABLE leave_requests (
    id            BIGSERIAL PRIMARY KEY,
    employee_id   BIGINT NOT NULL REFERENCES employees(id),
    leave_type_id BIGINT NOT NULL REFERENCES leave_types(id),
    start_date    DATE NOT NULL,
    end_date      DATE NOT NULL,
    days          NUMERIC NOT NULL,
    status        leave_status NOT NULL DEFAULT 'pending',
    is_lop        BOOLEAN NOT NULL DEFAULT false,
    decided_by    BIGINT REFERENCES employees(id),
    decided_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

CREATE INDEX leave_requests_employee_idx ON leave_requests(employee_id);
CREATE INDEX leave_requests_status_idx ON leave_requests(status);
