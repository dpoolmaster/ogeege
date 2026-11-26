CREATE TABLE auth.users (
    id                uuid PRIMARY KEY,
    email             text NOT NULL,
    password_hash     text NOT NULL,
    role              text NOT NULL,
    status            text NOT NULL,
    email_verified_at timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_role_check CHECK (role IN ('student', 'parent', 'methodist', 'admin')),
    CONSTRAINT users_status_check CHECK (status IN ('unverified', 'active', 'blocked', 'anonymized'))
);

CREATE UNIQUE INDEX users_email_unique ON auth.users (lower(email));

CREATE TABLE auth.sessions (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    token_hash   bytea NOT NULL,
    user_agent   text,
    ip           inet,
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_idx ON auth.sessions (user_id);
CREATE UNIQUE INDEX sessions_token_idx ON auth.sessions (token_hash);
CREATE INDEX sessions_expires_idx ON auth.sessions (expires_at);

CREATE TABLE auth.email_verifications (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX email_verifications_token_unique ON auth.email_verifications (token_hash);
CREATE INDEX email_verifications_user_idx ON auth.email_verifications (user_id);

CREATE TABLE auth.password_resets (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX password_resets_token_unique ON auth.password_resets (token_hash);
CREATE INDEX password_resets_user_idx ON auth.password_resets (user_id);

CREATE TABLE auth.login_attempts (
    id         uuid PRIMARY KEY,
    email      text NOT NULL,
    ip         inet,
    success    boolean NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX login_attempts_email_ip_idx ON auth.login_attempts (lower(email), ip, created_at);
