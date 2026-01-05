-- 000006_add_missing_columns.up.sql
-- Fixes schema mismatch between Go handlers and database
-- Adds missing columns to event and task tables

-- ============================================
-- EVENT TABLE FIXES
-- ============================================

-- Add description column
ALTER TABLE event ADD COLUMN IF NOT EXISTS description TEXT;

-- Rename tz to timezone (Go code expects 'timezone')
ALTER TABLE event RENAME COLUMN tz TO timezone;

-- Add location column
ALTER TABLE event ADD COLUMN IF NOT EXISTS location TEXT;

-- Add is_work_event column
ALTER TABLE event ADD COLUMN IF NOT EXISTS is_work_event BOOLEAN DEFAULT TRUE;

-- ============================================
-- TASK TABLE FIXES
-- ============================================

-- Add category column (EMERGENCY, ASAP, MUST_TODAY, DEADLINE_SOON, IF_POSSIBLE, BUFFER)
ALTER TABLE task ADD COLUMN IF NOT EXISTS category TEXT;

-- Add due_date column
ALTER TABLE task ADD COLUMN IF NOT EXISTS due_date TIMESTAMPTZ;

-- Add tags array column
ALTER TABLE task ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';

-- Add contexts array column (@home, @work, @computer, etc.)
ALTER TABLE task ADD COLUMN IF NOT EXISTS contexts TEXT[] DEFAULT '{}';

-- Add energy column (1-5 scale)
ALTER TABLE task ADD COLUMN IF NOT EXISTS energy INTEGER CHECK (energy IS NULL OR energy BETWEEN 1 AND 5);

-- Add parent_id for subtasks
ALTER TABLE task ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES task(id) ON DELETE SET NULL;

-- Add completed_at timestamp
ALTER TABLE task ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

-- Change status default from 'OPEN' to 'TODO'
ALTER TABLE task ALTER COLUMN status SET DEFAULT 'TODO';

-- Update existing 'OPEN' statuses to 'TODO'
UPDATE task SET status = 'TODO' WHERE status = 'OPEN';

-- ============================================
-- INDEXES
-- ============================================

-- Index for task due_date queries
CREATE INDEX IF NOT EXISTS idx_task_due_date ON task(due_date) WHERE due_date IS NOT NULL;

-- Index for parent_id (subtasks)
CREATE INDEX IF NOT EXISTS idx_task_parent ON task(parent_id) WHERE parent_id IS NOT NULL;

-- Index for category filtering
CREATE INDEX IF NOT EXISTS idx_task_category ON task(user_id, category) WHERE category IS NOT NULL;

-- Index for contexts array (GIN for array contains queries)
CREATE INDEX IF NOT EXISTS idx_task_contexts ON task USING gin(contexts);