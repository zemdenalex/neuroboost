-- api-go/migrations/000012_calendars.up.sql
--
-- Calendar as an access container. Before this migration, user_id was
-- responsible for two things at once: who owns the row and who can see it.
-- Here they split apart: access is granted by calendar membership, while
-- user_id remains authorship.

CREATE TABLE IF NOT EXISTS calendar (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    color      TEXT,
    kind       TEXT NOT NULL CHECK (kind IN ('personal','shared')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS calendar_member (
    calendar_id UUID NOT NULL REFERENCES calendar(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES "user"(id)   ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('owner','editor','viewer')),
    -- Reserved for busy/free mode. Always 'full' in slice 1.
    visibility  TEXT NOT NULL DEFAULT 'full'   CHECK (visibility IN ('full','busy')),
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('invited','active')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (calendar_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_calendar_member_user ON calendar_member (user_id, status);

ALTER TABLE event ADD COLUMN IF NOT EXISTS calendar_id UUID REFERENCES calendar(id);
ALTER TABLE task  ADD COLUMN IF NOT EXISTS calendar_id UUID REFERENCES calendar(id);

-- One personal calendar per person is an invariant, not a convention.
-- Without it, the schema would allow "two personal calendars for one
-- user", and any code looking up the personal calendar would have to
-- guess which one is the real one.
CREATE UNIQUE INDEX IF NOT EXISTS idx_calendar_one_personal_per_owner
    ON calendar (owner_id)
    WHERE kind = 'personal';

-- Give every existing user their personal calendar.
INSERT INTO calendar (owner_id, name, kind)
SELECT u.id, 'Мой календарь', 'personal'
FROM "user" u
WHERE NOT EXISTS (
    SELECT 1 FROM calendar c WHERE c.owner_id = u.id AND c.kind = 'personal'
);

INSERT INTO calendar_member (calendar_id, user_id, role, status)
SELECT c.id, c.owner_id, 'owner', 'active'
FROM calendar c
WHERE c.kind = 'personal'
ON CONFLICT DO NOTHING;

-- Everything existing moves into the author's personal calendar.
UPDATE event e
SET calendar_id = c.id
FROM calendar c
WHERE c.owner_id = e.user_id AND c.kind = 'personal' AND e.calendar_id IS NULL;

UPDATE task t
SET calendar_id = c.id
FROM calendar c
WHERE c.owner_id = t.user_id AND c.kind = 'personal' AND t.calendar_id IS NULL;

ALTER TABLE event ALTER COLUMN calendar_id SET NOT NULL;
ALTER TABLE task  ALTER COLUMN calendar_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_event_calendar_time ON event (calendar_id, starts_at);
CREATE INDEX IF NOT EXISTS idx_task_calendar       ON task  (calendar_id);
