-- users & auth
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE TABLE IF NOT EXISTS "user" (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tg_id text UNIQUE,
  email text,
  created_at timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS user_tg_id_idx ON "user"(tg_id);
