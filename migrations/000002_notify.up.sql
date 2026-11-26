CREATE TABLE notify.templates (
    id         uuid PRIMARY KEY,
    code       text NOT NULL,
    subject    text NOT NULL,
    body_md    text NOT NULL,
    channel    text NOT NULL,
    version    integer NOT NULL,
    status     text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT templates_channel_check CHECK (channel IN ('email')),
    CONSTRAINT templates_status_check CHECK (status IN ('active', 'inactive')),
    CONSTRAINT templates_version_check CHECK (version > 0)
);

CREATE UNIQUE INDEX templates_code_version_unique ON notify.templates (code, version);
CREATE UNIQUE INDEX templates_active_code_unique ON notify.templates (code) WHERE status = 'active';

CREATE TABLE notify.outbox (
    id           uuid PRIMARY KEY,
    user_id      uuid,
    channel      text NOT NULL,
    recipient    text NOT NULL,
    template     text NOT NULL,
    payload_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    status       text NOT NULL,
    attempts     integer NOT NULL DEFAULT 0,
    scheduled_at timestamptz NOT NULL DEFAULT now(),
    sent_at      timestamptz,
    last_error   text,
    dedup_key    text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT outbox_channel_check CHECK (channel IN ('email')),
    CONSTRAINT outbox_status_check CHECK (status IN ('pending', 'sent', 'failed')),
    CONSTRAINT outbox_attempts_check CHECK (attempts >= 0)
);

CREATE INDEX outbox_due_idx ON notify.outbox (scheduled_at) WHERE status = 'pending';
CREATE INDEX outbox_user_idx ON notify.outbox (user_id);
CREATE UNIQUE INDEX outbox_dedup_unique ON notify.outbox (dedup_key) WHERE dedup_key IS NOT NULL;
