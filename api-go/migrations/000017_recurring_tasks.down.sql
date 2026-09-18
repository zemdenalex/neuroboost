-- Откат 000017.
--
-- ⚠ Таблица сносится вместе с данными: состояния дней восстановить неоткуда,
-- их нет больше нигде. Откатывать только на dev.

DROP INDEX IF EXISTS idx_task_recurring;
DROP INDEX IF EXISTS idx_task_occurrence_task;
DROP TABLE IF EXISTS task_occurrence;

ALTER TABLE task     DROP COLUMN IF EXISTS event_id;
ALTER TABLE reminder DROP COLUMN IF EXISTS nag_count;
ALTER TABLE event    DROP COLUMN IF EXISTS nag_minutes;
ALTER TABLE task     DROP COLUMN IF EXISTS nag_minutes;
ALTER TABLE task     DROP COLUMN IF EXISTS repeat_anchor;
ALTER TABLE task     DROP COLUMN IF EXISTS rrule;
