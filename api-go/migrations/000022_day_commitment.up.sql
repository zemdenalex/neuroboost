-- «Задачи дня» (spec 2026-09-22 §7). A day is a DATE in the user's zone, never
-- an instant: a task has no clock (learning-a-day-is-a-date-not-an-instant).
CREATE TABLE IF NOT EXISTS day_commitment (
    user_id    UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    day        DATE NOT NULL,
    task_id    UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Soft delete: «removed before noon» stays visible to anyone asking why a
    -- day is the colour it is.
    removed_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, day, task_id)
);
CREATE INDEX IF NOT EXISTS idx_day_commitment_task ON day_commitment(task_id);

-- «The day is taken» — the proposal was confirmed. A day without a row here
-- is ⬛ however much was done (Denis, 22.09).
CREATE TABLE IF NOT EXISTS day_commitment_day (
    user_id      UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    day          DATE NOT NULL,
    confirmed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, day)
);
