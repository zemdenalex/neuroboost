CREATE TABLE IF NOT EXISTS task_requirement (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  task_id uuid NOT NULL REFERENCES task(id) ON DELETE CASCADE,
  kind requirement_kind NOT NULL,
  value text NOT NULL,
  is_optional boolean DEFAULT false,
  created_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS task_requirement_task_idx ON task_requirement(user_id, task_id);
