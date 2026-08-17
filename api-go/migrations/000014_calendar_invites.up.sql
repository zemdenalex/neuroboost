-- api-go/migrations/000014_calendar_invites.up.sql
--
-- Sharing a calendar with someone. Two channels, because they answer two
-- different questions and Denis asked for both (17.08.2026):
--
--   1. The person already has an account → invite them by email. That needs no
--      table at all: calendar_member has carried status='invited' since 000012,
--      and CalendarIDsFor has always filtered on 'active', so an invited member
--      sees nothing until they accept. That half was built and tested in slice 1.
--
--   2. The person may not have an account yet, or you would rather just send a
--      link → this table. A token, a short life, and one use.
--
-- 🔴 Two hours, his number. A link that grants write access to a calendar is a
-- bearer credential: whoever holds it is the invitee. Short life is the whole
-- defence, since we cannot know who opened it.

CREATE TABLE IF NOT EXISTS calendar_invite (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id UUID NOT NULL REFERENCES calendar(id) ON DELETE CASCADE,

    -- The secret. Stored as-is rather than hashed: it lives two hours, it is
    -- readable by anyone with database access anyway (as is every event in
    -- here), and hashing would buy nothing while making "show me my active
    -- links" impossible. If invites ever become long-lived, revisit this.
    token       TEXT NOT NULL UNIQUE,

    -- What the link grants. Fixed when the link is made, not when it is used:
    -- otherwise the grant would depend on when someone clicked.
    role        TEXT NOT NULL CHECK (role IN ('editor','viewer')),

    created_by  UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,

    -- Single use. NULL means unused; set on acceptance, together with who used
    -- it, so a link that turns up in the wrong hands leaves a trace.
    used_at     TIMESTAMPTZ,
    used_by     UUID REFERENCES "user"(id) ON DELETE SET NULL
);

-- The lookup every acceptance makes.
CREATE INDEX IF NOT EXISTS idx_calendar_invite_token ON calendar_invite (token);

-- Listing a calendar's live links, and sweeping dead ones.
CREATE INDEX IF NOT EXISTS idx_calendar_invite_calendar ON calendar_invite (calendar_id, expires_at);

-- 🔴 owner is not invitable. The role CHECK above already refuses it, and this
-- comment says why rather than leaving it looking like an omission: a calendar
-- has exactly one owner, it is the creator, and transferring ownership is a
-- different operation with different consequences (the owner can delete).
