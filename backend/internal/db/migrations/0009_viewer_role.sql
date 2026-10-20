-- A read-only login for public demos: it can see exactly what HR Admin sees but every
-- write is blocked at the middleware before it reaches a handler.
ALTER TYPE user_role ADD VALUE 'viewer';
