-- 000006_add_missing_columns.down.sql
-- Rollback: removes columns added in up migration
-- WARNING: This will delete data in these columns!

-- ============================================
-- REMOVE INDEXES FIRST
-- ============================================

DROP INDEX IF EXISTS idx_task_contexts;
DROP INDEX IF EXISTS idx_task_category;
DROP INDEX IF EXISTS idx_task_parent;
DROP INDEX IF EXISTS idx_task_due_date;

-- ============================================
-- TASK TABLE ROLLBACK
-- ============================================

-- Revert status default and update values back to OPEN
UPDATE task SET status = 'OPEN' WHERE status = 'TODO';
ALTER TABLE task ALTER COLUMN status SET DEFAULT 'OPEN';

-- Remove added columns
ALTER TABLE task DROP COLUMN IF EXISTS completed_at;
ALTER TABLE task DROP COLUMN IF EXISTS parent_id;
ALTER TABLE task DROP COLUMN IF EXISTS energy;
ALTER TABLE task DROP COLUMN IF EXISTS contexts;
ALTER TABLE task DROP COLUMN IF EXISTS tags;
ALTER TABLE task DROP COLUMN IF EXISTS due_date;
ALTER TABLE task DROP COLUMN IF EXISTS category;

-- ============================================
-- EVENT TABLE ROLLBACK
-- ============================================

ALTER TABLE event DROP COLUMN IF EXISTS is_work_event;
ALTER TABLE event DROP COLUMN IF EXISTS location;

-- Rename timezone back to tz
ALTER TABLE event RENAME COLUMN timezone TO tz;

ALTER TABLE event DROP COLUMN IF EXISTS description;