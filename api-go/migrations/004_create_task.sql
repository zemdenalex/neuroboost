CREATE TABLE IF NOT EXISTS task (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  title text NOT NULL,
  description text,
  status text NOT NULL DEFAULT 'OPEN',
  priority integer DEFAULT 0,
  estimated_minutes integer,
  opportunity_id uuid REFERENCES opportunity(id) ON DELETE SET NULL,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS task_user_status_idx ON task(user_id, status);
CREATE INDEX IF NOT EXISTS task_user_priority_idx ON task(user_id, priority);
CREATE INDEX IF NOT EXISTS task_opportunity_idx ON task(opportunity_id);
