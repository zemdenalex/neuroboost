DROP INDEX IF EXISTS idx_reminder_dedupe;
ALTER TABLE reminder DROP COLUMN IF EXISTS occurrence_start;
ALTER TABLE reminder DROP COLUMN IF EXISTS task_id;
ALTER TABLE reminder DROP COLUMN IF EXISTS source_kind;
ALTER TABLE task  DROP COLUMN IF EXISTS reminder_offsets;
ALTER TABLE event DROP COLUMN IF EXISTS reminder_offsets;
