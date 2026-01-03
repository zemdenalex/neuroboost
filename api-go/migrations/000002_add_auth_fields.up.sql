-- 000002_add_auth_fields.up.sql
-- Adds authentication fields to user table

-- Add password hash for email/password auth
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS password_hash TEXT;

-- Add Telegram profile fields
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS tg_username TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS tg_first_name TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS tg_last_name TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS tg_photo_url TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS tg_auth_date TIMESTAMPTZ;

-- Add email verification fields
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS verification_token TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS verification_token_expires_at TIMESTAMPTZ;

-- Add password reset fields
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS password_reset_token TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS password_reset_expires_at TIMESTAMPTZ;

-- Add updated_at timestamp
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Add display name (for UI)
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS display_name TEXT;

-- Add timezone preference
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS timezone TEXT DEFAULT 'Europe/Moscow';

-- Add locale preference
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS locale TEXT DEFAULT 'ru';

-- Add last login tracking
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- Change tg_id from TEXT to BIGINT if it's still TEXT
-- This is safe because we check the column type first
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'user' 
        AND column_name = 'tg_id' 
        AND data_type = 'text'
    ) THEN
        -- Drop the unique constraint first if exists
        ALTER TABLE "user" DROP CONSTRAINT IF EXISTS user_tg_id_key;
        -- Convert column type
        ALTER TABLE "user" ALTER COLUMN tg_id TYPE BIGINT USING tg_id::BIGINT;
        -- Re-add unique constraint
        ALTER TABLE "user" ADD CONSTRAINT user_tg_id_key UNIQUE (tg_id);
    END IF;
END $$;

-- Create indexes for new columns
CREATE INDEX IF NOT EXISTS idx_user_verification_token ON "user"(verification_token) WHERE verification_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_password_reset_token ON "user"(password_reset_token) WHERE password_reset_token IS NOT NULL;
