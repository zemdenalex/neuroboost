CREATE TABLE IF NOT EXISTS event_exception (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  event_id uuid NOT NULL REFERENCES event(id) ON DELETE CASCADE,
  occurrence timestamptz NOT NULL,
  skipped boolean DEFAULT true,
  replacement_event_id uuid NULL REFERENCES event(id) ON DELETE SET NULL,
  created_at timestamptz DEFAULT now(),
  UNIQUE (user_id, event_id, occurrence)
);
CREATE INDEX IF NOT EXISTS event_exception_event_idx ON event_exception(user_id, event_id);
