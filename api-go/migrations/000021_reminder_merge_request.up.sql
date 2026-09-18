-- The merge request needs to reach the bot, and the bot already has exactly one
-- channel for "here is something to answer": the reminder row with a
-- source_kind, which is how calendar invitations arrived in 000015.
--
-- A second channel would mean a second poller, a second delivery record, a
-- second place for «оно не пришло» to hide. So LINK joins EVENT, TASK, DIGEST
-- and INVITE rather than getting its own pipe.
ALTER TABLE reminder ADD COLUMN merge_request_id UUID
    REFERENCES account_merge_request(id) ON DELETE CASCADE;

-- ⚠ One notification per request. Without this a retried enqueue asks the same
-- person the same question twice, and the second answer acts on a merge that
-- already happened.
CREATE UNIQUE INDEX idx_reminder_one_per_merge_request
    ON reminder (merge_request_id)
    WHERE merge_request_id IS NOT NULL;
