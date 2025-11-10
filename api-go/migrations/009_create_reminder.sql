CREATE TABLE IF NOT EXISTS reminder (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  event_id uuid REFERENCES event(id) ON DELETE CASCADE,
  remind_at timestamptz NOT NULL,
  method text DEFAULT 'PUSH',
  created_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS reminder_user_time_idx ON reminder(user_id, remind_at);
CREATE INDEX IF NOT EXISTS reminder_event_idx ON reminder(event_id);
