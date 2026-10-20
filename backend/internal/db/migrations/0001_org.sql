-- Departments and the reporting hierarchy are the root everything else reads from.
CREATE TYPE user_role AS ENUM ('employee', 'manager', 'hr_admin');

CREATE TABLE departments (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employees (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    role          user_role NOT NULL DEFAULT 'employee',
    department_id BIGINT REFERENCES departments(id),
    manager_id    BIGINT REFERENCES employees(id),
    active        BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (manager_id IS NULL OR manager_id <> id)
);

CREATE INDEX employees_manager_idx ON employees(manager_id);
CREATE INDEX employees_department_idx ON employees(department_id);
