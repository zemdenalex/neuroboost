-- 000002_add_auth_fields.down.sql
-- Removes authentication fields from user table
-- WARNING: This will delete auth data!

DROP INDEX IF EXISTS idx_user_verification_token;
DROP INDEX IF EXISTS idx_user_password_reset_token;

ALTER TABLE "user" DROP COLUMN IF EXISTS password_hash;
ALTER TABLE "user" DROP COLUMN IF EXISTS tg_username;
ALTER TABLE "user" DROP COLUMN IF EXISTS tg_first_name;
ALTER TABLE "user" DROP COLUMN IF EXISTS tg_last_name;
ALTER TABLE "user" DROP COLUMN IF EXISTS tg_photo_url;
ALTER TABLE "user" DROP COLUMN IF EXISTS tg_auth_date;
ALTER TABLE "user" DROP COLUMN IF EXISTS email_verified_at;
ALTER TABLE "user" DROP COLUMN IF EXISTS verification_token;
ALTER TABLE "user" DROP COLUMN IF EXISTS verification_token_expires_at;
ALTER TABLE "user" DROP COLUMN IF EXISTS password_reset_token;
ALTER TABLE "user" DROP COLUMN IF EXISTS password_reset_expires_at;
ALTER TABLE "user" DROP COLUMN IF EXISTS updated_at;
ALTER TABLE "user" DROP COLUMN IF EXISTS display_name;
ALTER TABLE "user" DROP COLUMN IF EXISTS timezone;
ALTER TABLE "user" DROP COLUMN IF EXISTS locale;
ALTER TABLE "user" DROP COLUMN IF EXISTS last_login_at;
