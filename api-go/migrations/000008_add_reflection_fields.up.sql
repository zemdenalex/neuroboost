-- Add columns if missing (for environments predating baseline addition)
ALTER TABLE reflection ADD COLUMN IF NOT EXISTS was_completed BOOLEAN;
ALTER TABLE reflection ADD COLUMN IF NOT EXISTS was_on_time BOOLEAN;

-- Backfill any NULLs (from old nullable defaults)
UPDATE reflection SET was_completed = COALESCE(was_completed, false);
UPDATE reflection SET was_on_time = COALESCE(was_on_time, false);

-- Normalize to NOT NULL with DEFAULT false
ALTER TABLE reflection ALTER COLUMN was_completed SET NOT NULL;
ALTER TABLE reflection ALTER COLUMN was_on_time SET NOT NULL;
ALTER TABLE reflection ALTER COLUMN was_completed SET DEFAULT false;
ALTER TABLE reflection ALTER COLUMN was_on_time SET DEFAULT false;
