CREATE TABLE IF NOT EXISTS opportunity (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  title text NOT NULL,
  description text,
  status text NOT NULL DEFAULT 'NEW',
  tags text[] DEFAULT '{}',
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS opportunity_user_status_idx ON opportunity(user_id, status);
CREATE INDEX IF NOT EXISTS opportunity_created_idx ON opportunity(created_at desc);
