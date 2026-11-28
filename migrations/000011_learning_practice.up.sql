ALTER TABLE learning.sessions DROP CONSTRAINT sessions_kind_check;
ALTER TABLE learning.sessions
    ADD CONSTRAINT sessions_kind_check
    CHECK (kind IN ('diagnostic', 'practice', 'review', 'mock'));

ALTER TABLE learning.sessions
    ADD COLUMN plan_item_id uuid REFERENCES learning.plan_items (id) ON DELETE SET NULL;

CREATE INDEX sessions_plan_item_idx ON learning.sessions (plan_item_id);
