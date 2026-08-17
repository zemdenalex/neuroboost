DROP INDEX IF EXISTS idx_reminder_retry;
ALTER TABLE reminder DROP COLUMN IF EXISTS attempts;
