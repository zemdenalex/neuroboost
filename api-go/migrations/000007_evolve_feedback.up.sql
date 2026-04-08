-- Evolve feedback table into product backlog

-- Widen type column and expand allowed values
ALTER TABLE feedback DROP CONSTRAINT IF EXISTS feedback_type_check;
ALTER TABLE feedback ALTER COLUMN type TYPE VARCHAR(30);
ALTER TABLE feedback ADD CONSTRAINT feedback_type_check
  CHECK (type IN ('bug', 'feature', 'design', 'improvement', 'idea', 'other'));

-- Expand status values (add triaged, approved, denied)
ALTER TABLE feedback DROP CONSTRAINT IF EXISTS feedback_status_check;
ALTER TABLE feedback ADD CONSTRAINT feedback_status_check
  CHECK (status IN ('open', 'triaged', 'approved', 'denied', 'in_progress', 'resolved', 'closed'));

-- New columns for backlog management
ALTER TABLE feedback ADD COLUMN IF NOT EXISTS admin_notes TEXT DEFAULT '';
ALTER TABLE feedback ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';
ALTER TABLE feedback ADD COLUMN IF NOT EXISTS source VARCHAR(20) DEFAULT 'user';

-- Indexes for new columns
CREATE INDEX IF NOT EXISTS idx_feedback_source ON feedback(source);
CREATE INDEX IF NOT EXISTS idx_feedback_tags ON feedback USING GIN(tags);
