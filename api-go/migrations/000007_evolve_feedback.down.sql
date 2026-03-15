-- Revert feedback table to original schema

-- Drop new indexes
DROP INDEX IF EXISTS idx_feedback_tags;
DROP INDEX IF EXISTS idx_feedback_source;

-- Drop new columns
ALTER TABLE feedback DROP COLUMN IF EXISTS source;
ALTER TABLE feedback DROP COLUMN IF EXISTS tags;
ALTER TABLE feedback DROP COLUMN IF EXISTS admin_notes;

-- Restore original status constraint
ALTER TABLE feedback DROP CONSTRAINT IF EXISTS feedback_status_check;
ALTER TABLE feedback ADD CONSTRAINT feedback_status_check
  CHECK (status IN ('open', 'in_progress', 'resolved', 'closed'));

-- Restore original type constraint and column width
ALTER TABLE feedback DROP CONSTRAINT IF EXISTS feedback_type_check;
ALTER TABLE feedback ALTER COLUMN type TYPE VARCHAR(20);
ALTER TABLE feedback ADD CONSTRAINT feedback_type_check
  CHECK (type IN ('bug', 'feature', 'other'));
