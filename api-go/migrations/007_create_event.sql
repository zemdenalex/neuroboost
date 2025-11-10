CREATE TABLE IF NOT EXISTS event (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  title text NOT NULL,
  starts_at timestamptz NOT NULL,
  ends_at timestamptz NOT NULL,
  all_day boolean DEFAULT false,
  tz text,
  rrule text,
  color text,
  tags text[] DEFAULT '{}',
  task_id uuid REFERENCES task(id) ON DELETE SET NULL,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS event_user_time_idx ON event(user_id, starts_at, ends_at);
CREATE INDEX IF NOT EXISTS event_user_allday_idx ON event(user_id, all_day);
CREATE INDEX IF NOT EXISTS event_task_idx ON event(task_id);
