-- Revert to nullable with DEFAULT TRUE (baseline shape)
ALTER TABLE reflection ALTER COLUMN was_on_time DROP NOT NULL;
ALTER TABLE reflection ALTER COLUMN was_completed DROP NOT NULL;
ALTER TABLE reflection ALTER COLUMN was_on_time SET DEFAULT TRUE;
ALTER TABLE reflection ALTER COLUMN was_completed SET DEFAULT TRUE;
