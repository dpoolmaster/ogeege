CREATE TABLE admin.audit_log (
    id          uuid PRIMARY KEY,
    actor_id    uuid NOT NULL,
    actor_role  text NOT NULL,
    action      text NOT NULL,
    entity      text NOT NULL,
    entity_id   uuid,
    before_json jsonb,
    after_json  jsonb,
    ip          inet,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_log_actor_role_check CHECK (actor_role IN ('methodist', 'admin'))
);

CREATE INDEX audit_log_actor_idx ON admin.audit_log (actor_id, created_at DESC);
CREATE INDEX audit_log_entity_idx ON admin.audit_log (entity, entity_id, created_at DESC);
CREATE INDEX audit_log_created_idx ON admin.audit_log (created_at DESC);
