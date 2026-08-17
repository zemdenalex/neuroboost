-- P2 notifications. Reminder offsets are a property of the entity, not
-- pre-generated rows: a daily recurring event with five offsets would
-- otherwise be 1825 rows a year. Values are minutes before event start /
-- task due_date.
ALTER TABLE event ADD COLUMN IF NOT EXISTS reminder_offsets INTEGER[] NOT NULL DEFAULT '{}';
ALTER TABLE task  ADD COLUMN IF NOT EXISTS reminder_offsets INTEGER[] NOT NULL DEFAULT '{}';

-- The dormant `reminder` table (baseline 000001, never written to by any Go
-- code) becomes the delivery journal. Without it a worker restart would
-- re-send everything.
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS source_kind TEXT;
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS task_id UUID REFERENCES task(id) ON DELETE CASCADE;
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS occurrence_start TIMESTAMPTZ;

-- Backfill before NOT NULL, so a non-empty table survives this migration.
UPDATE reminder SET source_kind = 'TASK'  WHERE source_kind IS NULL AND task_id  IS NOT NULL;
-- Everything still unclassified becomes 'EVENT'. Deliberately an UPDATE and
-- not a DELETE: the table is believed empty in production (no Go code has
-- ever written to it) but that was established by reading code, not by
-- querying prod -- and this migration auto-runs on API start. An UPDATE
-- satisfies SET NOT NULL just as well and cannot destroy a row.
UPDATE reminder SET source_kind = 'EVENT' WHERE source_kind IS NULL;

ALTER TABLE reminder ALTER COLUMN source_kind SET NOT NULL;

-- Idempotency: one occurrence x one offset = one send, forever.
-- NULLS NOT DISTINCT is load-bearing: by default Postgres treats two NULLs in
-- a unique index as different values, so a digest row (no occurrence) or a
-- snooze row (no offset) would pass the index any number of times. With a
-- 3-minute scan window ticking every minute, the digest would go out three
-- times every morning.
CREATE UNIQUE INDEX IF NOT EXISTS idx_reminder_dedupe
  ON reminder (user_id, source_kind, COALESCE(event_id, task_id), occurrence_start, minutes_before)
  NULLS NOT DISTINCT;
