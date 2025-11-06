CREATE TABLE IF NOT EXISTS need (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  title text NOT NULL,
  due_date date,
  satisfied boolean DEFAULT false,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS need_user_date_idx ON need(user_id, due_date);
CREATE INDEX IF NOT EXISTS need_user_satisfied_idx ON need(user_id, satisfied);
