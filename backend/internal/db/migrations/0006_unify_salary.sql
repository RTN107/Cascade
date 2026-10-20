-- Base pay and allowances were special hardcoded fields living apart from the
-- generic pay-component system (Provident Fund etc). That split was the whole bug:
-- two disconnected places configured the same employee's pay. Folding base pay in as
-- the one mandatory earning component makes there one place, employee_salary_components,
-- that holds everything a payslip is built from.
ALTER TABLE salary_components ADD COLUMN is_base BOOLEAN NOT NULL DEFAULT false;
INSERT INTO salary_components (name, kind, is_base) VALUES ('Basic Pay', 'earning', true);

DROP TABLE salary_structures;

ALTER TABLE payslips DROP COLUMN base;
ALTER TABLE payslips DROP COLUMN allowances;
ALTER TABLE payslips ADD COLUMN gross_earnings NUMERIC NOT NULL DEFAULT 0;
ALTER TABLE payslips ADD COLUMN total_deductions NUMERIC NOT NULL DEFAULT 0;
