-- Bounded retry for failed reminder deliveries.
--
-- Until now a FAILED row was terminal, and because idx_reminder_dedupe stops
-- the scan re-creating an identical row, one failure meant that reminder was
-- lost for the whole local day. That is how the empty-digest bug (fixed in
-- 52683de) went unnoticed: the morning digest failed, the FAILED row blocked
-- any retry, and nothing was left to notice except a status nobody read.
--
-- attempts makes retry bounded. Without it, requeueing FAILED would retry
-- forever — a user who blocked the bot would generate one failed send a minute
-- until somebody looked.
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0;

-- Rows that already failed under the old behaviour get one more chance rather
-- than staying dead: they failed once, and attempts defaults to 0 for them.
CREATE INDEX IF NOT EXISTS idx_reminder_retry
  ON reminder (status, sent_at)
  WHERE status = 'FAILED';
