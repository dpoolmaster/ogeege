-- Откат ограничения завершится ошибкой, если в базе уже есть сессии kind = 'review'.
-- Перед откатом удалите или преобразуйте такие сессии.
DROP INDEX IF EXISTS learning.sessions_plan_item_idx;
ALTER TABLE learning.sessions DROP COLUMN IF EXISTS plan_item_id;
ALTER TABLE learning.sessions DROP CONSTRAINT sessions_kind_check;
ALTER TABLE learning.sessions
    ADD CONSTRAINT sessions_kind_check
    CHECK (kind IN ('diagnostic', 'practice', 'mock'));
