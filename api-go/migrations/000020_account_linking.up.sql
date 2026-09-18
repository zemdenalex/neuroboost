-- Account linking, v0.4.11.5.
--
-- Two one-shot secrets and one merge request.
--
-- 🔴 Plain CREATE TABLE, never IF NOT EXISTS. Gotcha 18: the baseline used
-- IF NOT EXISTS, silently accepted a foreign table of the same name on
-- production, recorded success, and five migrations later took the API down in
-- a crash-loop. A migration that adopts whatever it finds is not idempotent,
-- it is blind.

-- A login link (kind = 'LOGIN') or a six-digit linking code (kind = 'LINK').
--
-- Only the hash is stored. The secret exists in exactly one place — the message
-- the person is looking at — which is the point of gotcha 14: redaction at the
-- log protects the reader, not the value. A value that was never written down
-- cannot leak from a log at all.
CREATE TABLE auth_link_token (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL CHECK (kind IN ('LOGIN', 'LINK')),
    token_hash  TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    attempts    INT NOT NULL DEFAULT 0,
    redeemed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Redeeming looks a token up by hash and kind, never by user.
CREATE INDEX idx_auth_link_token_lookup ON auth_link_token (kind, token_hash);

-- 🔴 One live token per person per kind. Without this, pressing «Войти на сайт»
-- five times leaves five working links, and revoking the one you can see does
-- nothing. Partial, so spent and expired rows accumulate harmlessly.
CREATE UNIQUE INDEX idx_auth_link_token_one_live
    ON auth_link_token (user_id, kind)
    WHERE redeemed_at IS NULL;

-- A merge waiting for the Telegram side to confirm.
--
-- 🔴 This row is the ONLY way a merge can happen. There is deliberately no
-- endpoint that takes two user ids and merges them: the code proves control of
-- the Telegram account, the site session proves control of the email, and this
-- row is the record that both were present at once. Structural, not a policy
-- somebody has to remember.
CREATE TABLE account_merge_request (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- 🔴 SET NULL, not CASCADE, and therefore nullable. A merge ENDS by deleting
    -- the absorbed account; with CASCADE the record of the merge would be
    -- deleted by the merge itself, and the one row that says what happened to
    -- somebody's data would disappear at exactly the moment it starts to matter.
    site_user_id UUID REFERENCES "user"(id) ON DELETE SET NULL,
    tg_user_id   UUID REFERENCES "user"(id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending', 'done', 'rejected', 'expired')),
    -- Which account survives, and what happens to two personal calendars.
    -- Both are answered in the bot (Denis, 17.09: «about the calendar ask too»),
    -- so both are NULL until he answers.
    keep_user_id     UUID REFERENCES "user"(id) ON DELETE SET NULL,
    calendar_choice  TEXT CHECK (calendar_choice IN ('merge', 'keep_both')),
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at  TIMESTAMPTZ,
    CHECK (site_user_id <> tg_user_id)
);

-- At most one open request per side: two half-finished merges of the same
-- account is a state nobody designed for.
CREATE UNIQUE INDEX idx_merge_request_one_open_site
    ON account_merge_request (site_user_id) WHERE status = 'pending';
CREATE UNIQUE INDEX idx_merge_request_one_open_tg
    ON account_merge_request (tg_user_id) WHERE status = 'pending';
