-- Reverses 000014. Drops the link table only.
--
-- ⚠ Memberships created BY those links are deliberately left alone: they are
-- calendar_member rows like any other by then, indistinguishable from ones made
-- by an in-app invite, and revoking access is a decision, not a rollback.
DROP TABLE IF EXISTS calendar_invite;
