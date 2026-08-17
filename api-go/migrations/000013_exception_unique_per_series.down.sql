-- Restores the per-user uniqueness. Note this cannot restore the duplicate
-- rows the up-migration collapsed: they were the defect, not data.
ALTER TABLE event_exception DROP CONSTRAINT IF EXISTS event_exception_series_occurrence_key;

ALTER TABLE event_exception
    ADD CONSTRAINT event_exception_user_id_event_id_occurrence_key
    UNIQUE (user_id, event_id, occurrence);
