-- Remove settings column
DROP INDEX IF EXISTS idx_user_settings;
ALTER TABLE "user" DROP COLUMN IF EXISTS settings;
