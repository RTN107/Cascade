-- Lets HR Admin define extra named pay items (Provident Fund, a bonus, insurance, etc.)
-- beyond base and allowances, without touching code.
CREATE TYPE salary_component_kind AS ENUM ('earning', 'deduction');

CREATE TABLE salary_components (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    kind salary_component_kind NOT NULL
);

CREATE TABLE employee_salary_components (
    employee_id  BIGINT NOT NULL REFERENCES employees(id),
    component_id BIGINT NOT NULL REFERENCES salary_components(id),
    amount       NUMERIC NOT NULL,
    PRIMARY KEY (employee_id, component_id)
);

-- name and kind are copied here at run time, so a payslip still shows what a component
-- was called even if it's later renamed or removed.
CREATE TABLE payslip_component_lines (
    id           BIGSERIAL PRIMARY KEY,
    payslip_id   BIGINT NOT NULL REFERENCES payslips(id),
    component_id BIGINT REFERENCES salary_components(id),
    name         TEXT NOT NULL,
    kind         salary_component_kind NOT NULL,
    amount       NUMERIC NOT NULL
);
