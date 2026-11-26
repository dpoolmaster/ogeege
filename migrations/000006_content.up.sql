CREATE TABLE content.exams (
    id         uuid PRIMARY KEY,
    code       text NOT NULL,
    title      text NOT NULL,
    subject    text NOT NULL,
    status     text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT exams_status_check CHECK (status IN ('draft', 'published', 'archived'))
);

CREATE UNIQUE INDEX exams_code_unique ON content.exams (code);

CREATE TABLE content.exam_years (
    id               uuid PRIMARY KEY,
    exam_id          uuid NOT NULL REFERENCES content.exams (id) ON DELETE CASCADE,
    year             integer NOT NULL,
    duration_minutes integer NOT NULL,
    status           text NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT exam_years_year_check CHECK (year >= 2000 AND year <= 2100),
    CONSTRAINT exam_years_duration_check CHECK (duration_minutes > 0),
    CONSTRAINT exam_years_status_check CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT exam_years_exam_year_unique UNIQUE (exam_id, year)
);

CREATE INDEX exam_years_exam_idx ON content.exam_years (exam_id, status);

CREATE TABLE content.kim_numbers (
    id                uuid PRIMARY KEY,
    exam_year_id      uuid NOT NULL REFERENCES content.exam_years (id) ON DELETE CASCADE,
    number            integer NOT NULL,
    part              integer NOT NULL,
    answer_type       text NOT NULL,
    max_primary_score numeric(6, 2) NOT NULL,
    title             text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT kim_numbers_number_check CHECK (number > 0),
    CONSTRAINT kim_numbers_part_check CHECK (part IN (1, 2)),
    CONSTRAINT kim_numbers_answer_type_check CHECK (answer_type IN ('short', 'extended')),
    CONSTRAINT kim_numbers_score_check CHECK (max_primary_score >= 0),
    CONSTRAINT kim_numbers_year_number_unique UNIQUE (exam_year_id, number)
);

CREATE INDEX kim_numbers_year_idx ON content.kim_numbers (exam_year_id, number);

CREATE TABLE content.topics (
    id          uuid PRIMARY KEY,
    exam_id     uuid NOT NULL REFERENCES content.exams (id) ON DELETE CASCADE,
    parent_id   uuid REFERENCES content.topics (id) ON DELETE RESTRICT,
    code        text NOT NULL,
    title       text NOT NULL,
    order_index integer NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT topics_order_index_check CHECK (order_index >= 0),
    CONSTRAINT topics_exam_code_unique UNIQUE (exam_id, code)
);

CREATE INDEX topics_parent_idx ON content.topics (parent_id);
CREATE INDEX topics_exam_order_idx ON content.topics (exam_id, order_index);

CREATE TABLE content.kim_topics (
    kim_number_id uuid NOT NULL REFERENCES content.kim_numbers (id) ON DELETE CASCADE,
    topic_id      uuid NOT NULL REFERENCES content.topics (id) ON DELETE CASCADE,
    weight        numeric(6, 3) NOT NULL,
    CONSTRAINT kim_topics_weight_check CHECK (weight >= 0),
    PRIMARY KEY (kim_number_id, topic_id)
);

CREATE INDEX kim_topics_topic_idx ON content.kim_topics (topic_id);

CREATE TABLE content.theory (
    id         uuid PRIMARY KEY,
    topic_id   uuid NOT NULL REFERENCES content.topics (id) ON DELETE CASCADE,
    content_md text NOT NULL,
    version    integer NOT NULL,
    status     text NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT theory_version_check CHECK (version > 0),
    CONSTRAINT theory_status_check CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT theory_topic_version_unique UNIQUE (topic_id, version)
);

CREATE INDEX theory_topic_status_idx ON content.theory (topic_id, status, version DESC);

CREATE TABLE content.tasks (
    id             uuid PRIMARY KEY,
    exam_year_id   uuid NOT NULL REFERENCES content.exam_years (id) ON DELETE RESTRICT,
    kim_number_id  uuid NOT NULL REFERENCES content.kim_numbers (id) ON DELETE RESTRICT,
    topic_id       uuid REFERENCES content.topics (id) ON DELETE SET NULL,
    difficulty     text NOT NULL,
    statement_md   text NOT NULL,
    answer_spec    jsonb NOT NULL,
    answer_display text,
    source         text NOT NULL,
    source_ref     text,
    status         text NOT NULL,
    is_public      boolean NOT NULL DEFAULT false,
    content_hash   bytea NOT NULL,
    version        integer NOT NULL DEFAULT 1,
    created_by     uuid,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tasks_difficulty_check CHECK (difficulty IN ('easy', 'medium', 'hard')),
    CONSTRAINT tasks_status_check CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT tasks_version_check CHECK (version > 0)
);

CREATE INDEX tasks_kim_idx ON content.tasks (kim_number_id, status);
CREATE INDEX tasks_topic_idx ON content.tasks (topic_id, status);
CREATE INDEX tasks_public_idx ON content.tasks (is_public, status);
CREATE UNIQUE INDEX tasks_hash_unique ON content.tasks (content_hash);

CREATE TABLE content.task_versions (
    id             uuid PRIMARY KEY,
    task_id        uuid NOT NULL REFERENCES content.tasks (id) ON DELETE CASCADE,
    version        integer NOT NULL,
    statement_md   text NOT NULL,
    answer_spec    jsonb NOT NULL,
    answer_display text,
    difficulty     text NOT NULL,
    kim_number_id  uuid NOT NULL,
    topic_id       uuid,
    source         text NOT NULL,
    source_ref     text,
    created_by     uuid,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT task_versions_version_check CHECK (version > 0),
    CONSTRAINT task_versions_task_version_unique UNIQUE (task_id, version)
);

CREATE INDEX task_versions_task_idx ON content.task_versions (task_id, version DESC);

CREATE TABLE content.task_options (
    id         uuid PRIMARY KEY,
    task_id    uuid NOT NULL REFERENCES content.tasks (id) ON DELETE CASCADE,
    position   integer NOT NULL,
    content_md text NOT NULL,
    is_correct boolean NOT NULL DEFAULT false,
    CONSTRAINT task_options_position_check CHECK (position > 0),
    CONSTRAINT task_options_task_position_unique UNIQUE (task_id, position)
);

CREATE TABLE content.task_attachments (
    id          uuid PRIMARY KEY,
    task_id     uuid NOT NULL REFERENCES content.tasks (id) ON DELETE CASCADE,
    kind        text NOT NULL,
    storage_key text NOT NULL,
    url         text,
    alt         text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT task_attachments_kind_check CHECK (kind IN ('image')),
    CONSTRAINT task_attachments_key_unique UNIQUE (storage_key)
);

CREATE INDEX task_attachments_task_idx ON content.task_attachments (task_id);

CREATE TABLE content.variants (
    id               uuid PRIMARY KEY,
    exam_year_id     uuid NOT NULL REFERENCES content.exam_years (id) ON DELETE CASCADE,
    title            text NOT NULL,
    kind             text NOT NULL,
    duration_minutes integer NOT NULL,
    status           text NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT variants_kind_check CHECK (kind IN ('diagnostic', 'practice', 'mock')),
    CONSTRAINT variants_duration_check CHECK (duration_minutes > 0),
    CONSTRAINT variants_status_check CHECK (status IN ('draft', 'published', 'archived'))
);

CREATE INDEX variants_year_status_idx ON content.variants (exam_year_id, status);

CREATE TABLE content.variant_tasks (
    id         uuid PRIMARY KEY,
    variant_id uuid NOT NULL REFERENCES content.variants (id) ON DELETE CASCADE,
    position   integer NOT NULL,
    task_id    uuid NOT NULL REFERENCES content.tasks (id) ON DELETE RESTRICT,
    points     numeric(6, 2) NOT NULL,
    CONSTRAINT variant_tasks_position_check CHECK (position > 0),
    CONSTRAINT variant_tasks_points_check CHECK (points >= 0),
    CONSTRAINT variant_tasks_variant_position_unique UNIQUE (variant_id, position),
    CONSTRAINT variant_tasks_variant_task_unique UNIQUE (variant_id, task_id)
);

CREATE INDEX variant_tasks_task_idx ON content.variant_tasks (task_id);

CREATE TABLE content.rubrics (
    id            uuid PRIMARY KEY,
    kim_number_id uuid NOT NULL REFERENCES content.kim_numbers (id) ON DELETE CASCADE,
    exam_year_id  uuid NOT NULL REFERENCES content.exam_years (id) ON DELETE CASCADE,
    version       integer NOT NULL,
    criteria_json jsonb NOT NULL,
    status        text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT rubrics_version_check CHECK (version > 0),
    CONSTRAINT rubrics_status_check CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT rubrics_number_year_version_unique UNIQUE (kim_number_id, exam_year_id, version)
);

CREATE INDEX rubrics_lookup_idx ON content.rubrics (kim_number_id, exam_year_id, status);

CREATE TABLE content.score_scales (
    id            uuid PRIMARY KEY,
    exam_year_id  uuid NOT NULL REFERENCES content.exam_years (id) ON DELETE CASCADE,
    primary_score integer NOT NULL,
    test_score    integer NOT NULL,
    grade         integer,
    min_threshold integer,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT score_scales_primary_check CHECK (primary_score >= 0),
    CONSTRAINT score_scales_test_check CHECK (test_score >= 0),
    CONSTRAINT score_scales_grade_check CHECK (grade IS NULL OR (grade >= 1 AND grade <= 5)),
    CONSTRAINT score_scales_year_primary_unique UNIQUE (exam_year_id, primary_score)
);

CREATE INDEX score_scales_year_idx ON content.score_scales (exam_year_id, primary_score);
