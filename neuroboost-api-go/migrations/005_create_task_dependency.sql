DO $$ BEGIN
  CREATE TYPE requirement_kind AS ENUM ('DEVICE','LOCATION','SOFTWARE','CONDITION');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
CREATE TABLE IF NOT EXISTS task_dependency (
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  task_id uuid NOT NULL REFERENCES task(id) ON DELETE CASCADE,
  depends_on_task_id uuid NOT NULL REFERENCES task(id) ON DELETE CASCADE,
  created_at timestamptz DEFAULT now(),
  PRIMARY KEY (user_id, task_id, depends_on_task_id),
  CHECK (task_id <> depends_on_task_id)
);
CREATE INDEX IF NOT EXISTS task_dependency_task_idx ON task_dependency(user_id, task_id);
CREATE INDEX IF NOT EXISTS task_dependency_depends_idx ON task_dependency(user_id, depends_on_task_id);
