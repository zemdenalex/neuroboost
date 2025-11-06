CREATE TABLE IF NOT EXISTS pattern_metrics (
  user_id uuid NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  week_start date NOT NULL,
  completion_rate numeric,
  mood_impact numeric,
  energy_impact numeric,
  time_accuracy numeric,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now(),
  PRIMARY KEY (user_id, week_start)
);
CREATE INDEX IF NOT EXISTS pattern_metrics_user_week_idx ON pattern_metrics(user_id, week_start desc);
