-- «Remind me regularly until the deadline» (Лешик, 07.11): one rule per task
-- or event. Either {"every_minutes": N} or {"every_days": K, "times": ["12:00", …]}.
-- The scanner expands it only inside its own window; NULL = no rule.
ALTER TABLE task  ADD COLUMN IF NOT EXISTS remind_rule JSONB;
ALTER TABLE event ADD COLUMN IF NOT EXISTS remind_rule JSONB;
