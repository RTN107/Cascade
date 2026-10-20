-- Every state-changing action writes here, linked to the record it affected.
CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    actor_id    BIGINT REFERENCES employees(id),
    action      TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   BIGINT NOT NULL,
    details     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_entity_idx ON audit_log(entity_type, entity_id);
