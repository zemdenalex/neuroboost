-- Reverses 000015.
--
-- 🔴 Order matters: the invite rows have to go BEFORE the index is narrowed
-- back, or two invitations to different calendars for one person would collide
-- and the CREATE UNIQUE INDEX would fail halfway, leaving schema_migrations
-- dirty. Dropping the column would take them anyway; doing it explicitly means
-- the reason is written down rather than relied upon.
DELETE FROM reminder WHERE source_kind = 'INVITE';

DROP INDEX IF EXISTS idx_reminder_dedupe;

CREATE UNIQUE INDEX IF NOT EXISTS idx_reminder_dedupe
  ON reminder (user_id, source_kind, COALESCE(event_id, task_id), occurrence_start, minutes_before)
  NULLS NOT DISTINCT;

ALTER TABLE reminder DROP COLUMN IF EXISTS calendar_id;
