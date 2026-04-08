-- Add settings JSONB column for user preferences
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}';

-- Add comment explaining the structure
COMMENT ON COLUMN "user".settings IS 'User preferences: {header_variant, work_days, work_start, work_end, features, ui_scale}';

-- Create index for common queries on settings
CREATE INDEX IF NOT EXISTS idx_user_settings ON "user" USING gin (settings);
