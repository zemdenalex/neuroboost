-- An occurrence can be excepted once per SERIES, not once per person.
--
-- 🔴 The write and the read disagreed. event_exception carried
-- UNIQUE (user_id, event_id, occurrence) from the baseline, so the upsert in
-- detachOccurrence conflicted per user; but fetchExceptions reads exceptions
-- by calendar and deliberately ignores user_id, because an exception is shared
-- state of the series — one member moving next Tuesday moves it for everyone
-- who can see the calendar.
--
-- With two writing members on one calendar (which TransferOwnership creates
-- today: it demotes the previous owner to editor and leaves the new one beside
-- them) both could detach the same occurrence. The second insert did not
-- conflict, so the calendar ended up with TWO replacement events while the
-- original occurrence was hidden once. Nothing resolves that on its own.
--
-- Fixing it in the writer alone would leave the rule living in whichever code
-- path remembers it. The database is where an invariant belongs.

-- Collapse any duplicates that already exist, keeping the most recent
-- exception for each (event_id, occurrence). "Most recent" rather than "first"
-- because the later edit is the one the members last saw take effect.
--
-- The replacement events belonging to the losing rows are deleted too: they
-- are exactly the extra copies this migration exists to remove, and leaving
-- them would keep the visible symptom while claiming the fix.
WITH ranked AS (
    SELECT ctid,
           replacement_event_id,
           ROW_NUMBER() OVER (
               PARTITION BY event_id, occurrence
               ORDER BY created_at DESC, ctid DESC
           ) AS rn
    FROM event_exception
),
losers AS (
    SELECT ctid, replacement_event_id FROM ranked WHERE rn > 1
),
deleted_exceptions AS (
    DELETE FROM event_exception
    WHERE ctid IN (SELECT ctid FROM losers)
)
DELETE FROM event
WHERE id IN (SELECT replacement_event_id FROM losers WHERE replacement_event_id IS NOT NULL);

ALTER TABLE event_exception DROP CONSTRAINT IF EXISTS event_exception_user_id_event_id_occurrence_key;

-- Named explicitly so the ON CONFLICT target in detachOccurrence and this
-- constraint cannot drift apart silently.
ALTER TABLE event_exception
    ADD CONSTRAINT event_exception_series_occurrence_key
    UNIQUE (event_id, occurrence);
