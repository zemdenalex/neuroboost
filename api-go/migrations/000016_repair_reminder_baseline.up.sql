-- Repair for a divergence older than the migration system.
--
-- 🔴 What happened on 2026-08-18, in production, during the v0.4.10 release:
-- migration 000010 failed on `column "minutes_before" does not exist`,
-- schema_migrations went to `dirty = 10`, golang-migrate then refused to run at
-- all, and the API crash-looped. Production was down until the shape was
-- repaired by hand.
--
-- The cause is in 000001_baseline. It declares:
--
--     CREATE TABLE IF NOT EXISTS reminder ( ... minutes_before INTEGER, ... )
--
-- and production ALREADY HAD a table called `reminder`, created before the
-- migration system existed, with a `method` column and without
-- minutes_before / channel / status / message / sent_at. IF NOT EXISTS found a
-- table, created nothing, and the migration recorded success. The baseline was
-- green and the schema was wrong, and nothing anywhere could tell the
-- difference — for five migrations, until one of them indexed a column the
-- baseline had promised and never delivered.
--
-- ⚠ This migration cannot rescue the chain. Anything replaying 000001..000015
-- against the legacy shape still dies at 000010, which is BEFORE this file. It
-- exists so that a database repaired by hand, or one restored from a pre-2026
-- dump and force-marked past 10, converges on the shape the Go code expects
-- instead of carrying the divergence forward silently.
--
-- The repair SQL for the crash itself — the thing to run when a restored dump
-- puts you back at dirty=10 — is exactly the five statements below, followed by
-- forcing the version to 9.
--
-- The table is a delivery journal that no code wrote to before P2, so on every
-- database this has ever run against it is empty and these are free.

ALTER TABLE reminder ADD COLUMN IF NOT EXISTS minutes_before INTEGER;
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS channel TEXT DEFAULT 'TELEGRAM';
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS status  TEXT DEFAULT 'PENDING';
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS message TEXT;
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS sent_at TIMESTAMPTZ;
