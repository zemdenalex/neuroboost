DO $$ BEGIN
  CREATE TYPE node_kind AS ENUM ('DREAM','GOAL','PROJECT','PHASE','TASK','EVENT','OPPORTUNITY','NEED');
  CREATE TYPE edge_kind AS ENUM ('DERIVES','BLOCKED_BY','CONTAINS','SCHEDULES','RELATES_TO');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS planning_node (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  kind node_kind NOT NULL,
  ref_id uuid,
  title text NOT NULL,
  description text,
  metadata jsonb,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS planning_node_user_kind_idx ON planning_node(user_id, kind);
CREATE INDEX IF NOT EXISTS planning_node_ref_idx ON planning_node(ref_id);
