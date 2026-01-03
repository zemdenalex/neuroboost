-- Add is_admin column to user table
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT FALSE;

-- Create index for admin queries
CREATE INDEX IF NOT EXISTS idx_user_admin ON "user"(is_admin) WHERE is_admin = TRUE;
