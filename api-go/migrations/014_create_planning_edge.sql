CREATE TABLE IF NOT EXISTS planning_edge (
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  src uuid NOT NULL REFERENCES planning_node(id) ON DELETE CASCADE,
  dst uuid NOT NULL REFERENCES planning_node(id) ON DELETE CASCADE,
  kind edge_kind NOT NULL,
  weight integer DEFAULT 1,
  created_at timestamptz DEFAULT now(),
  PRIMARY KEY (user_id, src, dst, kind),
  CHECK (src <> dst)
);
CREATE INDEX IF NOT EXISTS planning_edge_src_idx ON planning_edge(user_id, src);
CREATE INDEX IF NOT EXISTS planning_edge_dst_idx ON planning_edge(user_id, dst);
