CREATE TABLE learning.study_plans (
    id                 uuid PRIMARY KEY,
    user_id            uuid NOT NULL,
    exam_id            uuid NOT NULL,
    exam_date          date NOT NULL,
    target_score       integer NOT NULL,
    weekly_minutes     integer NOT NULL,
    status             text NOT NULL,
    version            integer NOT NULL,
    last_replanned_at  timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT study_plans_status_check CHECK (status IN ('active', 'superseded')),
    CONSTRAINT study_plans_target_check CHECK (target_score >= 0 AND target_score <= 100),
    CONSTRAINT study_plans_weekly_check CHECK (weekly_minutes >= 1 AND weekly_minutes <= 10080),
    CONSTRAINT study_plans_version_check CHECK (version > 0)
);

CREATE INDEX study_plans_user_status_idx ON learning.study_plans (user_id, status);
CREATE UNIQUE INDEX study_plans_active_unique ON learning.study_plans (user_id) WHERE status = 'active';

CREATE TABLE learning.plan_weeks (
    id         uuid PRIMARY KEY,
    plan_id    uuid NOT NULL REFERENCES learning.study_plans (id) ON DELETE CASCADE,
    week_start date NOT NULL,
    phase      text NOT NULL,
    goals_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT plan_weeks_phase_check CHECK (phase IN ('diagnostics', 'base', 'practice', 'review')),
    CONSTRAINT plan_weeks_plan_week_unique UNIQUE (plan_id, week_start)
);

CREATE INDEX plan_weeks_plan_idx ON learning.plan_weeks (plan_id, week_start);

CREATE TABLE learning.plan_items (
    id            uuid PRIMARY KEY,
    week_id       uuid NOT NULL REFERENCES learning.plan_weeks (id) ON DELETE CASCADE,
    day_of_week   integer NOT NULL,
    kind          text NOT NULL,
    topic_id      uuid,
    kim_number_id uuid,
    task_count    integer NOT NULL,
    status        text NOT NULL DEFAULT 'pending',
    CONSTRAINT plan_items_day_check CHECK (day_of_week >= 1 AND day_of_week <= 7),
    CONSTRAINT plan_items_kind_check CHECK (kind IN ('diagnostic', 'theory', 'practice', 'review')),
    CONSTRAINT plan_items_task_count_check CHECK (task_count > 0),
    CONSTRAINT plan_items_status_check CHECK (status IN ('pending', 'done'))
);

CREATE INDEX plan_items_week_idx ON learning.plan_items (week_id, day_of_week);

CREATE TABLE learning.sessions (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL,
    exam_id       uuid NOT NULL,
    kind          text NOT NULL,
    variant_id    uuid,
    status        text NOT NULL,
    started_at    timestamptz NOT NULL,
    deadline_at   timestamptz,
    finished_at   timestamptz,
    score_primary numeric(8, 2),
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sessions_kind_check CHECK (kind IN ('diagnostic', 'practice', 'mock')),
    CONSTRAINT sessions_status_check CHECK (status IN ('in_progress', 'completed'))
);

CREATE INDEX sessions_user_created_idx ON learning.sessions (user_id, created_at DESC);
CREATE INDEX sessions_in_progress_idx ON learning.sessions (status) WHERE status = 'in_progress';

CREATE TABLE learning.session_items (
    id            uuid PRIMARY KEY,
    session_id    uuid NOT NULL REFERENCES learning.sessions (id) ON DELETE CASCADE,
    position      integer NOT NULL,
    task_id       uuid NOT NULL,
    task_snapshot jsonb NOT NULL,
    max_points    numeric(6, 2) NOT NULL,
    status        text NOT NULL DEFAULT 'pending',
    CONSTRAINT session_items_position_check CHECK (position > 0),
    CONSTRAINT session_items_max_points_check CHECK (max_points >= 0),
    CONSTRAINT session_items_status_check CHECK (status IN ('pending', 'answered')),
    CONSTRAINT session_items_session_position_unique UNIQUE (session_id, position)
);

CREATE TABLE learning.answers (
    id                uuid PRIMARY KEY,
    session_item_id   uuid NOT NULL UNIQUE REFERENCES learning.session_items (id) ON DELETE CASCADE,
    raw_answer        text NOT NULL,
    normalized_answer text NOT NULL,
    is_correct        boolean NOT NULL,
    awarded_points    numeric(6, 2) NOT NULL,
    check_kind        text NOT NULL,
    checked_at        timestamptz NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT answers_awarded_points_check CHECK (awarded_points >= 0)
);

CREATE TABLE learning.topic_mastery (
    user_id           uuid NOT NULL,
    topic_id          uuid NOT NULL,
    mastery           numeric(4, 3) NOT NULL,
    attempts          integer NOT NULL,
    correct           integer NOT NULL,
    interval_days     integer NOT NULL DEFAULT 1,
    ease              numeric(3, 2) NOT NULL DEFAULT 2.50,
    last_practiced_at timestamptz,
    next_review_at    timestamptz,
    PRIMARY KEY (user_id, topic_id),
    CONSTRAINT topic_mastery_value_check CHECK (mastery >= 0 AND mastery <= 1),
    CONSTRAINT topic_mastery_attempts_check CHECK (attempts >= 0),
    CONSTRAINT topic_mastery_correct_check CHECK (correct >= 0 AND correct <= attempts),
    CONSTRAINT topic_mastery_interval_check CHECK (interval_days >= 0)
);

CREATE INDEX topic_mastery_user_idx ON learning.topic_mastery (user_id);

CREATE TABLE learning.kim_mastery (
    user_id           uuid NOT NULL,
    kim_number_id     uuid NOT NULL,
    mastery           numeric(4, 3) NOT NULL,
    attempts          integer NOT NULL,
    correct           integer NOT NULL,
    last_practiced_at timestamptz,
    PRIMARY KEY (user_id, kim_number_id),
    CONSTRAINT kim_mastery_value_check CHECK (mastery >= 0 AND mastery <= 1),
    CONSTRAINT kim_mastery_attempts_check CHECK (attempts >= 0),
    CONSTRAINT kim_mastery_correct_check CHECK (correct >= 0 AND correct <= attempts)
);

CREATE INDEX kim_mastery_user_idx ON learning.kim_mastery (user_id);

CREATE TABLE learning.session_results (
    session_id     uuid PRIMARY KEY REFERENCES learning.sessions (id) ON DELETE CASCADE,
    primary_score  numeric(8, 2) NOT NULL,
    max_score      numeric(8, 2) NOT NULL,
    test_score     integer,
    grade          integer,
    breakdown_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT session_results_primary_check CHECK (primary_score >= 0),
    CONSTRAINT session_results_max_check CHECK (max_score >= 0),
    CONSTRAINT session_results_grade_check CHECK (grade IS NULL OR (grade >= 1 AND grade <= 5))
);
