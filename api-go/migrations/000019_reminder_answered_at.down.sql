DROP INDEX IF EXISTS idx_reminder_unanswered;
ALTER TABLE reminder DROP COLUMN IF EXISTS answered_at;
