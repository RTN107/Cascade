-- Loss of Pay is assessed against the employee's combined remaining balance across every
-- leave type, not just the type a request happens to be tagged with, so a request can be
-- partly covered by leave and partly Loss of Pay.
ALTER TABLE leave_requests ADD COLUMN lop_days NUMERIC NOT NULL DEFAULT 0;

-- Records which leave type balances were actually drawn down to cover an approved
-- request's covered days, since one request can draw from more than one type once its own
-- type's balance runs out.
CREATE TABLE leave_request_allocations (
    id               BIGSERIAL PRIMARY KEY,
    leave_request_id BIGINT NOT NULL REFERENCES leave_requests(id),
    leave_type_id    BIGINT NOT NULL REFERENCES leave_types(id),
    days             NUMERIC NOT NULL
);

CREATE INDEX leave_request_allocations_request_idx ON leave_request_allocations(leave_request_id);
CREATE INDEX leave_request_allocations_type_idx ON leave_request_allocations(leave_type_id);
