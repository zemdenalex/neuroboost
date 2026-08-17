-- api-go/migrations/000015_reminder_calendar.up.sql
--
-- A notification that is about a CALENDAR rather than an event or a task.
--
-- Denis, 17.08: an invitation should reach the person in Telegram, not only
-- wait in the app for them to notice. The delivery pipeline (P2) already does
-- everything needed — claim, retry, ack, buttons — and it addresses rows in
-- `reminder`. What it could not carry was "which calendar", because a reminder
-- pointed at an event or a task and nothing else.

ALTER TABLE reminder ADD COLUMN IF NOT EXISTS calendar_id UUID REFERENCES calendar(id) ON DELETE CASCADE;

-- 🔴 The dedupe index has to be rebuilt, and this is the risky part of the
-- migration, so read before changing it.
--
-- The old key is (user_id, source_kind, COALESCE(event_id, task_id),
-- occurrence_start, minutes_before) with NULLS NOT DISTINCT — that last clause
-- is what stops the morning digest going out three times, because a digest row
-- has NULL in three of those columns and by default Postgres treats two NULLs
-- as different values.
--
-- An INVITE row has NULL for event_id, task_id, occurrence_start AND
-- minutes_before. Under the old key, two invitations to two different calendars
-- for the same person collide, and the second one silently never arrives.
--
-- Adding calendar_id to the key fixes that and changes nothing for existing
-- rows: every row already in the table has calendar_id NULL, so under
-- NULLS NOT DISTINCT they compare exactly as they did before. The index becomes
-- more permissive only for rows that carry a calendar — which is precisely the
-- new kind.
DROP INDEX IF EXISTS idx_reminder_dedupe;

CREATE UNIQUE INDEX IF NOT EXISTS idx_reminder_dedupe
  ON reminder (user_id, source_kind, COALESCE(event_id, task_id), calendar_id, occurrence_start, minutes_before)
  NULLS NOT DISTINCT;

-- The notifier claims by (status, remind_at); nothing about that changes.
