-- The system role (employee, manager, hr_admin) only controls permissions. Title is the
-- person's actual job designation (Engineering Manager, Software Engineer, and so on),
-- shown in the org tree so it reads like a real company, not a permission level.
ALTER TABLE employees ADD COLUMN title TEXT;
