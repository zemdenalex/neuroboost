CREATE TABLE IF NOT EXISTS reflection (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  event_id uuid NOT NULL REFERENCES event(id) ON DELETE CASCADE,
  energy integer CHECK (energy BETWEEN 1 AND 10),
  mood integer CHECK (mood BETWEEN 1 AND 10),
  focus integer CHECK (focus BETWEEN 1 AND 10),
  notes text,
  created_at timestamptz DEFAULT now(),
  UNIQUE(user_id, event_id)
);
CREATE INDEX IF NOT EXISTS reflection_user_idx ON reflection(user_id, created_at desc);
CREATE INDEX IF NOT EXISTS reflection_event_idx ON reflection(event_id);
