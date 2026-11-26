CREATE TABLE profile.profiles (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL UNIQUE,
    first_name text,
    last_name  text,
    birth_date date,
    grade      integer,
    timezone   text,
    contacts   jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT profiles_grade_check CHECK (grade IS NULL OR (grade >= 1 AND grade <= 11))
);

CREATE TABLE profile.consents (
    id               uuid PRIMARY KEY,
    user_id          uuid NOT NULL,
    kind             text NOT NULL,
    granted          boolean NOT NULL,
    document_version text NOT NULL,
    granted_at       timestamptz NOT NULL DEFAULT now(),
    ip               inet,
    CONSTRAINT consents_kind_check CHECK (kind IN ('personal_data', 'legal_representative'))
);

CREATE INDEX consents_user_kind_idx ON profile.consents (user_id, kind);

CREATE TABLE profile.study_goals (
    id             uuid PRIMARY KEY,
    user_id        uuid NOT NULL,
    exam_id        uuid NOT NULL,
    exam_date      date NOT NULL,
    target_score   integer NOT NULL,
    weekly_minutes integer NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT study_goals_target_score_check CHECK (target_score >= 0 AND target_score <= 100),
    CONSTRAINT study_goals_weekly_minutes_check CHECK (weekly_minutes > 0 AND weekly_minutes <= 10080)
);

CREATE INDEX study_goals_user_created_idx ON profile.study_goals (user_id, created_at DESC);

CREATE TABLE profile.settings (
    id                  uuid PRIMARY KEY,
    user_id             uuid NOT NULL UNIQUE,
    email_notifications boolean NOT NULL DEFAULT true,
    weekly_report       boolean NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
