-- Inbox (spec docs/team/architecture/V003-20261010-arc-inbox-bot.md): notes,
-- ideas, links and feedback Denis sends a separate Telegram bot, kept to be
-- gone through later instead of acted on at once. One owner, no user FK: the
-- bot is «just for me», and the owner is a Telegram id, not an account.
CREATE TABLE IF NOT EXISTS inbox_note (
    -- BIGSERIAL, not UUID: the bot answers «✅ #142» and the PC script takes
    -- `set 142 done` — the id is typed by a person.
    id            BIGSERIAL PRIMARY KEY,
    owner_tg_id   BIGINT NOT NULL,
    type          TEXT NOT NULL DEFAULT 'inbox'
                  CHECK (type IN ('inbox','idea','link','feedback','reflection','note')),
    projects      TEXT[] NOT NULL DEFAULT '{}',
    text          TEXT NOT NULL DEFAULT '',
    -- Where a forward came from: {kind, name, username, url, date}. NULL for
    -- Denis's own messages.
    source        JSONB,
    -- chat | inline
    via           TEXT NOT NULL DEFAULT 'chat',
    status        TEXT NOT NULL DEFAULT 'new'
                  CHECK (status IN ('new','discussed','done','dropped')),
    -- Telegram redelivers an update the bot did not confirm, so the same
    -- message can be posted twice; the bot names it («chat:<id>:<msg>»,
    -- «inline:<result>», «album:<group>») and the second POST finds the first.
    client_key    TEXT NOT NULL UNIQUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_inbox_note_status ON inbox_note(status, id);

CREATE TABLE IF NOT EXISTS inbox_media (
    id              BIGSERIAL PRIMARY KEY,
    note_id         BIGINT NOT NULL REFERENCES inbox_note(id) ON DELETE CASCADE,
    kind            TEXT NOT NULL,          -- photo | voice | audio | video | document | video_note
    mime            TEXT NOT NULL DEFAULT '',
    file_name       TEXT NOT NULL DEFAULT '',
    size            BIGINT NOT NULL DEFAULT 0,
    tg_file_unique_id TEXT NOT NULL,
    -- NULL when the file was too large to carry: the note still says a file
    -- was there, instead of pretending it never was.
    data            BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (note_id, tg_file_unique_id)
);
