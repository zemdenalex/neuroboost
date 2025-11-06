CREATE TABLE IF NOT EXISTS alert_status (
  user_id uuid PRIMARY KEY REFERENCES "user"(id) ON DELETE CASCADE,
  level text NOT NULL DEFAULT 'GREEN',
  days_until_risk integer DEFAULT 999,
  last_checked_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);
