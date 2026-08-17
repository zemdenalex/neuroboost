-- Actual focused minutes logged against a task (e.g. by the Pomodoro timer).
-- Complements estimated_minutes for estimate-vs-reality comparison.
ALTER TABLE task ADD COLUMN IF NOT EXISTS actual_minutes INTEGER NOT NULL DEFAULT 0;
