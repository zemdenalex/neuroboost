DROP INDEX IF EXISTS idx_reminder_one_per_merge_request;
ALTER TABLE reminder DROP COLUMN IF EXISTS merge_request_id;
