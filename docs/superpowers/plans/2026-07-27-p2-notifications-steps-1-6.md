# P2 Notifications (steps 1–6) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reminders actually arrive in Telegram — one API-side worker computes what is due, a notifier goroutine inside the existing bot pulls, sends and acks.

**Architecture:** Reminder offsets live as an `INTEGER[]` column on `event` and `task`; nothing is materialised ahead of time. A once-a-minute ticker inside `api-go` expands recurrences over a short scan window, computes due (occurrence × offset) pairs, and inserts `PENDING` rows into the repurposed `reminder` table with `ON CONFLICT DO NOTHING`. The bot pulls those rows over a service-token-protected `/api/svc/` endpoint, sends them, and acks. All time math is pure functions over `[]time.Time`, testable without a database.

**Tech Stack:** Go 1.22, chi v5, pgx/v5, PostgreSQL 16, golang-migrate, `go-telegram-bot-api/v5` (bot module is separate: `github.com/zemdenalex/neuroboost-bot`).

**Spec:** `docs/superpowers/specs/2026-07-27-p2-notifications-design.md`. This plan covers spec §10 steps 1–6 only. Steps 7–10 (buttons, `<ReminderOffsets/>` UI, foreign-host deploy) are a separate plan.

## Global Constraints

- **Never edit `.env`.** Local values are passed inline to `docker compose`. Credentials never land in files (project rule: Tracker App only).
- **Never modify an existing migration.** `000010` is the next number; a mistake costs a `000011`.
- **Parameterised pgx queries only** — never string-concatenated SQL.
- **No `any` in TypeScript** (no frontend work in this plan, but the rule stands).
- **Backend done means:** `cd api-go && go build ./... && go test ./...` both pass.
- **Bot module is separate:** `cd bot && go build ./...` — it has its own `go.mod` and is not covered by `api-go`'s build.
- **Postgres is `16-alpine`** — `NULLS NOT DISTINCT` (PG15+) is available.
- Commit after each task. Never add a Claude/Anthropic co-author trailer.

---

## File Structure

| File | Responsibility |
|---|---|
| `api-go/migrations/000010_add_reminders.up.sql` / `.down.sql` | Offsets columns + `reminder` journal columns + dedupe index |
| `api-go/internal/events/occurrences.go` | **New.** Exported wrapper over the unexported `expandRecurrence` so other packages can list occurrences |
| `api-go/internal/reminders/settings.go` | Parse the `reminders` blob out of `"user".settings` JSONB — per-field fallback, never errors |
| `api-go/internal/reminders/due.go` | **Pure.** `(occurrences × offsets) → what is due in this window` |
| `api-go/internal/reminders/quiet.go` | **Pure.** Quiet-hours shift/drop, digest local-midnight timing |
| `api-go/internal/reminders/scan.go` | DB: read users/events/tasks, insert `PENDING` rows |
| `api-go/internal/reminders/worker.go` | The ticker; owns the scan window |
| `api-go/internal/reminders/service.go` | `/api/svc/` handlers + constant-time token middleware + rate limit |
| `api-go/internal/config/config.go` | Adds `ServiceToken` |
| `api-go/cmd/api/main.go` | Wires the routes and starts the worker |
| `bot/internal/notifier/notifier.go` | **New.** Pull → send → ack goroutine |

Each `reminders` file is deliberately small: the two pure files carry all the logic worth testing, and the two DB files carry no logic worth testing. That split is what makes the feature verifiable without a database.

---

### Task 1: Migration 000010

**Files:**
- Create: `api-go/migrations/000010_add_reminders.up.sql`
- Create: `api-go/migrations/000010_add_reminders.down.sql`

**Interfaces:**
- Consumes: nothing.
- Produces: columns `event.reminder_offsets INTEGER[]`, `task.reminder_offsets INTEGER[]`, `reminder.source_kind TEXT NOT NULL`, `reminder.task_id UUID`, `reminder.occurrence_start TIMESTAMPTZ`, and unique index `idx_reminder_dedupe`.

**Context the implementer needs:** the `reminder` table already exists in `000001_baseline.up.sql:151-165` and has **never been written to by any Go code**. It already carries `user_id`, `event_id`, `remind_at`, `minutes_before`, `channel`, `status`, `message`, `sent_at`. We are repurposing it as a delivery journal, not creating a new table.

- [ ] **Step 1: Write the up migration**

```sql
-- P2 notifications. Reminder offsets are a property of the entity, not
-- pre-generated rows: a daily recurring event with five offsets would
-- otherwise be 1825 rows a year. Values are minutes before event start /
-- task due_date.
ALTER TABLE event ADD COLUMN IF NOT EXISTS reminder_offsets INTEGER[] NOT NULL DEFAULT '{}';
ALTER TABLE task  ADD COLUMN IF NOT EXISTS reminder_offsets INTEGER[] NOT NULL DEFAULT '{}';

-- The dormant `reminder` table becomes the delivery journal. Without it a
-- worker restart would re-send everything.
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS source_kind TEXT;
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS task_id UUID REFERENCES task(id) ON DELETE CASCADE;
ALTER TABLE reminder ADD COLUMN IF NOT EXISTS occurrence_start TIMESTAMPTZ;

-- Backfill before NOT NULL, so a non-empty table survives this migration.
UPDATE reminder SET source_kind = 'TASK'  WHERE source_kind IS NULL AND task_id  IS NOT NULL;
-- Everything still unclassified becomes 'EVENT'. Deliberately an UPDATE and
-- not a DELETE: the table is believed empty in production (no Go code has
-- ever written to it) but that was established by reading code, not by
-- querying prod — and this migration auto-runs on API start. An UPDATE
-- satisfies SET NOT NULL just as well and cannot destroy a row.
UPDATE reminder SET source_kind = 'EVENT' WHERE source_kind IS NULL;

ALTER TABLE reminder ALTER COLUMN source_kind SET NOT NULL;

-- Idempotency: one occurrence x one offset = one send, forever.
-- NULLS NOT DISTINCT is load-bearing: by default Postgres treats two NULLs in
-- a unique index as different values, so a digest row (no occurrence) or a
-- snooze row (no offset) would pass the index any number of times. With a
-- 3-minute scan window ticking every minute, the digest would go out three
-- times every morning.
CREATE UNIQUE INDEX IF NOT EXISTS idx_reminder_dedupe
  ON reminder (user_id, source_kind, COALESCE(event_id, task_id), occurrence_start, minutes_before)
  NULLS NOT DISTINCT;
```

Note the `DELETE` on step 3 of the backfill: a pre-existing row with neither `event_id` nor `task_id` cannot be classified and cannot be delivered. The table is empty in production, so this deletes nothing; it exists so `SET NOT NULL` cannot fail.

- [ ] **Step 2: Write the down migration**

```sql
DROP INDEX IF EXISTS idx_reminder_dedupe;
ALTER TABLE reminder DROP COLUMN IF EXISTS occurrence_start;
ALTER TABLE reminder DROP COLUMN IF EXISTS task_id;
ALTER TABLE reminder DROP COLUMN IF EXISTS source_kind;
ALTER TABLE task  DROP COLUMN IF EXISTS reminder_offsets;
ALTER TABLE event DROP COLUMN IF EXISTS reminder_offsets;
```

- [ ] **Step 3: Run it against a real local database, up and down**

Migrations auto-run when the API container starts, so a broken `000010` means the API will not boot on staging. Test it now — and test `down` now, because after it reaches staging you cannot.

```bash
cd "E:/Projects/007 - Ventures/V003 - NeuroBoost"
POSTGRES_PASSWORD=localdev JWT_SECRET=localdevsecretlocaldevsecret32 \
  docker compose -f docker-compose.dev.yml up -d db api
docker compose -f docker-compose.dev.yml logs api | tail -30
```

Expected: log line showing migration `10` applied, API listening, no error.

- [ ] **Step 4: Verify the dedupe index actually dedupes NULLs**

This is the one assertion that cannot be made by reading the SQL. Run it against the live dev DB:

```bash
docker compose -f docker-compose.dev.yml exec -T db psql -U neuroboost -d neuroboost -c "
  INSERT INTO reminder (user_id, source_kind, remind_at, occurrence_start, minutes_before)
  SELECT id, 'DIGEST', NOW(), NULL, NULL FROM \"user\" LIMIT 1;
  INSERT INTO reminder (user_id, source_kind, remind_at, occurrence_start, minutes_before)
  SELECT id, 'DIGEST', NOW(), NULL, NULL FROM \"user\" LIMIT 1;"
```

Expected: the **second** INSERT fails with `duplicate key value violates unique constraint "idx_reminder_dedupe"`. If it succeeds, `NULLS NOT DISTINCT` did not take effect — stop and fix before continuing. Clean up: `DELETE FROM reminder;`

- [ ] **Step 5: Verify down works, then re-apply up**

```bash
docker compose -f docker-compose.dev.yml exec -T db psql -U neuroboost -d neuroboost \
  -c "\d reminder"
```
Expected after re-running up: `source_kind` present and `not null`.

- [ ] **Step 6: Commit**

```bash
git add api-go/migrations/000010_add_reminders.up.sql api-go/migrations/000010_add_reminders.down.sql
git commit -m "feat(reminders): migration 000010 — offsets columns and delivery journal"
```

---

### Task 2: Reminder offsets flow through the events and tasks API

**Files:**
- Modify: `api-go/internal/events/types.go` (add field to `Event`, `CreateEventRequest`, `UpdateEventRequest`)
- Modify: `api-go/internal/events/handlers.go` (SELECT/INSERT/UPDATE the column)
- Modify: `api-go/internal/tasks/types.go` (add field to `Task`, `CreateTaskRequest`, `UpdateTaskRequest`)
- Modify: `api-go/internal/tasks/handlers.go` (SELECT/INSERT/UPDATE the column, including the batch insert path)

**Interfaces:**
- Consumes: the columns from Task 1.
- Produces: `events.Event.ReminderOffsets []int` (JSON `reminder_offsets`), `tasks.Task.ReminderOffsets []int` (JSON `reminder_offsets`). Task 5's scan reads these columns directly from SQL, not through these structs, but the API must round-trip them or nobody can ever set an offset.

**Why this task exists at all:** without it the worker scans a table where `reminder_offsets` is always `'{}'` and nothing is ever due. The UI that sets offsets is step 8 (a later plan); this task is what makes the column reachable via the existing REST API, which is enough to test end-to-end with `curl`.

- [ ] **Step 1: Add the field to the event structs**

In `api-go/internal/events/types.go`, add to `Event` (after `Tags`):

```go
	ReminderOffsets []int `json:"reminder_offsets"`
```

Add to `CreateEventRequest`:

```go
	// Pointer, not a bare slice: the spec distinguishes "field absent" (apply
	// the user's default preset) from "explicitly empty" (deliberately no
	// reminders). A bare []int cannot tell those apart.
	ReminderOffsets *[]int `json:"reminder_offsets,omitempty"`
```

Add to `UpdateEventRequest`:

```go
	ReminderOffsets *[]int `json:"reminder_offsets,omitempty"`
```

- [ ] **Step 2: Add the same three fields to the task structs**

In `api-go/internal/tasks/types.go`, add `ReminderOffsets []int` with tag `json:"reminder_offsets"` to `Task`, and `ReminderOffsets *[]int` with tag `json:"reminder_offsets,omitempty"` to `CreateTaskRequest` and `UpdateTaskRequest`.

- [ ] **Step 3: Read the column in every event SELECT**

Find every `SELECT` in `api-go/internal/events/handlers.go` that scans into an `Event` (list, get, create-returning, update-returning). Add `COALESCE(reminder_offsets, '{}')` to the column list and `&event.ReminderOffsets` to the matching `Scan` call, in the same position.

`COALESCE` is belt-and-braces — the column is `NOT NULL DEFAULT '{}'` — but the scan target is a slice and a surprise NULL would be a runtime error rather than an empty list.

- [ ] **Step 4: Write the column on event create and update**

In the create INSERT, add `reminder_offsets` to the column list and bind:

```go
	offsets := []int{}
	if req.ReminderOffsets != nil {
		offsets = *req.ReminderOffsets
	}
```

In the dynamic update builder (the same pattern the file already uses for other optional fields), add:

```go
	if req.ReminderOffsets != nil {
		updates = append(updates, fmt.Sprintf("reminder_offsets = $%d", argNum))
		args = append(args, *req.ReminderOffsets)
		argNum++
	}
```

- [ ] **Step 5: Do the same for tasks, including the batch path**

Repeat steps 3–4 in `api-go/internal/tasks/handlers.go`. The batch endpoint from P1 shares `insertTask` — adding the column there covers both `CreateHandler` and `BatchCreateHandler` in one edit. Confirm that is still true before editing; if `insertTask` no longer exists, edit both call sites.

- [ ] **Step 6: Build and run the full backend test suite**

```bash
cd api-go && go build ./... && go test ./...
```
Expected: build clean, all existing tests pass. No new tests in this task — it is plumbing with no branching logic beyond the nil check, and the nil-vs-empty distinction is exercised by Task 4's preset tests.

- [ ] **Step 7: Verify the round-trip against the running dev API**

```bash
# Register a throwaway user and keep the token in a shell variable only.
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"p2@local.test","password":"localdevpassword","display_name":"P2"}' \
  | python -c "import sys,json;print(json.load(sys.stdin)['data']['token'])")

curl -s -X POST http://localhost:8081/api/events \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"title":"offsets round-trip","starts_at":"2026-08-01T09:00:00Z","ends_at":"2026-08-01T10:00:00Z","reminder_offsets":[1440,60]}'
```
Expected: response JSON contains `"reminder_offsets":[1440,60]`.

- [ ] **Step 8: Commit**

```bash
git add api-go/internal/events api-go/internal/tasks
git commit -m "feat(reminders): reminder_offsets round-trips through events and tasks API"
```

---

### Task 3: Occurrences wrapper + `DueReminders` pure function

**Files:**
- Create: `api-go/internal/events/occurrences.go`
- Create: `api-go/internal/reminders/due.go`
- Test: `api-go/internal/reminders/due_test.go`

**Interfaces:**
- Consumes: `events.Event` (already exists), the unexported `expandRecurrence(event Event, rangeStart, rangeEnd time.Time, exceptions []time.Time) []Event` at `events/recurrence.go:81`.
- Produces:
  - `events.OccurrencesInRange(ev Event, from, to time.Time, exceptions []time.Time) []time.Time`
  - `reminders.Due{OccurrenceStart time.Time; MinutesBefore int; RemindAt time.Time}`
  - `reminders.DueReminders(occurrences []time.Time, offsets []int, windowStart, windowEnd time.Time) []Due`

**Design note for the implementer:** `DueReminders` deliberately takes `[]time.Time`, not an `Event`. That keeps the whole reminder calculation independent of the events package, the database and RRULE parsing — which is why it can be tested exhaustively in milliseconds. The worker is responsible for producing the occurrence list.

- [ ] **Step 1: Write the exported occurrences wrapper**

`api-go/internal/events/occurrences.go`:

```go
package events

import "time"

// OccurrencesInRange returns the start times of every occurrence of ev that
// falls inside [from, to). For a non-recurring event that is either its own
// start or nothing.
//
// This exists because expandRecurrence is unexported and returns []Event with
// synthetic instance IDs; callers outside this package (the reminder worker)
// want start times, not events.
func OccurrencesInRange(ev Event, from, to time.Time, exceptions []time.Time) []time.Time {
	instances := expandRecurrence(ev, from, to, exceptions)
	starts := make([]time.Time, 0, len(instances))
	for _, inst := range instances {
		starts = append(starts, inst.StartsAt)
	}
	return starts
}
```

- [ ] **Step 2: Write the failing test for `DueReminders`**

`api-go/internal/reminders/due_test.go`:

```go
package reminders

import (
	"testing"
	"time"
)

func at(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.UTC)
}

func TestDueRemindersMatchesOffsetInsideWindow(t *testing.T) {
	occ := at(2026, 8, 1, 10, 0)
	// Window is the minute in which the 60-minute reminder comes due.
	got := DueReminders([]time.Time{occ}, []int{1440, 60}, at(2026, 8, 1, 8, 59), at(2026, 8, 1, 9, 1))
	if len(got) != 1 {
		t.Fatalf("want 1 due reminder, got %d: %+v", len(got), got)
	}
	if got[0].MinutesBefore != 60 {
		t.Errorf("MinutesBefore = %d, want 60", got[0].MinutesBefore)
	}
	if !got[0].RemindAt.Equal(at(2026, 8, 1, 9, 0)) {
		t.Errorf("RemindAt = %v, want 09:00", got[0].RemindAt)
	}
	if !got[0].OccurrenceStart.Equal(occ) {
		t.Errorf("OccurrenceStart = %v, want %v", got[0].OccurrenceStart, occ)
	}
}

func TestDueRemindersIsHalfOpen(t *testing.T) {
	occ := at(2026, 8, 1, 10, 0)
	// remind_at is exactly 09:00. A window starting at 09:00 includes it;
	// a window ending at 09:00 does not. Without this, consecutive scan
	// windows would both claim the same reminder — harmless thanks to the
	// unique index, but it would hide real bugs behind ON CONFLICT.
	if n := len(DueReminders([]time.Time{occ}, []int{60}, at(2026, 8, 1, 9, 0), at(2026, 8, 1, 9, 1))); n != 1 {
		t.Errorf("window starting exactly at remind_at: got %d, want 1", n)
	}
	if n := len(DueReminders([]time.Time{occ}, []int{60}, at(2026, 8, 1, 8, 59), at(2026, 8, 1, 9, 0))); n != 0 {
		t.Errorf("window ending exactly at remind_at: got %d, want 0", n)
	}
}

func TestDueRemindersEveryOccurrenceOfARecurringSeries(t *testing.T) {
	// Three daily occurrences; only the one whose reminder lands in the
	// window is due. This is the property that makes recurring events work:
	// the result carries which occurrence it belongs to.
	occs := []time.Time{
		at(2026, 8, 1, 10, 0),
		at(2026, 8, 2, 10, 0),
		at(2026, 8, 3, 10, 0),
	}
	got := DueReminders(occs, []int{60}, at(2026, 8, 2, 8, 59), at(2026, 8, 2, 9, 1))
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if !got[0].OccurrenceStart.Equal(at(2026, 8, 2, 10, 0)) {
		t.Errorf("wrong occurrence: %v", got[0].OccurrenceStart)
	}
}

func TestDueRemindersMultipleOffsetsCanFireTogether(t *testing.T) {
	// Two events an hour apart in the same window, or one event whose 60- and
	// 1440-minute reminders coincide — both must survive as separate rows.
	occs := []time.Time{at(2026, 8, 1, 10, 0), at(2026, 8, 2, 10, 0)}
	got := DueReminders(occs, []int{60, 1440}, at(2026, 8, 1, 8, 59), at(2026, 8, 1, 9, 1))
	if len(got) != 2 {
		t.Fatalf("want 2 (60min before Aug1, 1440min before Aug2), got %d: %+v", len(got), got)
	}
}

func TestDueRemindersEmptyInputs(t *testing.T) {
	if n := len(DueReminders(nil, []int{60}, at(2026, 8, 1, 0, 0), at(2026, 8, 1, 1, 0))); n != 0 {
		t.Errorf("no occurrences should yield nothing, got %d", n)
	}
	if n := len(DueReminders([]time.Time{at(2026, 8, 1, 10, 0)}, nil, at(2026, 8, 1, 0, 0), at(2026, 8, 2, 0, 0))); n != 0 {
		t.Errorf("no offsets should yield nothing, got %d", n)
	}
}

func TestDueRemindersIgnoresNegativeOffsets(t *testing.T) {
	// -1 is the snooze sentinel in the reminder table and must never be
	// interpretable as "one minute after the event".
	occ := at(2026, 8, 1, 10, 0)
	if n := len(DueReminders([]time.Time{occ}, []int{-1}, at(2026, 8, 1, 9, 59), at(2026, 8, 1, 10, 2))); n != 0 {
		t.Errorf("negative offset must be ignored, got %d", n)
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

```bash
cd api-go && go test ./internal/reminders/ -v
```
Expected: FAIL — the package does not exist yet (`no Go files in .../internal/reminders`).

- [ ] **Step 4: Write the minimal implementation**

`api-go/internal/reminders/due.go`:

```go
// Package reminders computes which notifications are due and hands them to
// the notifier. The time math lives in pure functions (due.go, quiet.go) so
// it can be tested without a database; scan.go and worker.go hold the I/O.
package reminders

import "time"

// Due is one reminder that ought to exist: a single occurrence paired with a
// single offset. It maps 1:1 onto a row in the reminder journal.
type Due struct {
	OccurrenceStart time.Time
	MinutesBefore   int
	RemindAt        time.Time
}

// DueReminders returns the (occurrence, offset) pairs whose remind-at moment
// falls inside the half-open window [windowStart, windowEnd).
//
// Half-open matters: the worker scans overlapping windows so a skipped tick
// (deploy, restart) cannot lose a reminder. The unique index makes the
// overlap safe, but a closed window would make every boundary reminder
// collide on purpose, which would mask genuine duplicate bugs behind
// ON CONFLICT DO NOTHING.
func DueReminders(occurrences []time.Time, offsets []int, windowStart, windowEnd time.Time) []Due {
	due := []Due{}
	for _, occ := range occurrences {
		for _, off := range offsets {
			// Negative offsets are not "after the event": -1 is the snooze
			// sentinel stored in reminder.minutes_before and must never be
			// produced by a scan.
			if off < 0 {
				continue
			}
			remindAt := occ.Add(-time.Duration(off) * time.Minute)
			if remindAt.Before(windowStart) || !remindAt.Before(windowEnd) {
				continue
			}
			due = append(due, Due{
				OccurrenceStart: occ,
				MinutesBefore:   off,
				RemindAt:        remindAt,
			})
		}
	}
	return due
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd api-go && go test ./internal/reminders/ -v && go build ./...
```
Expected: all six tests PASS, build clean.

- [ ] **Step 6: Commit**

```bash
git add api-go/internal/events/occurrences.go api-go/internal/reminders/
git commit -m "feat(reminders): DueReminders pure function and occurrences wrapper"
```

---

### Task 4: Quiet hours, digest timing, and settings parsing

**Files:**
- Create: `api-go/internal/reminders/quiet.go`
- Create: `api-go/internal/reminders/settings.go`
- Test: `api-go/internal/reminders/quiet_test.go`
- Test: `api-go/internal/reminders/settings_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks (both files are pure).
- Produces:
  - `reminders.ShiftForQuietHours(remindAt time.Time, minutesBefore int, quietStart, quietEnd string, loc *time.Location) (time.Time, bool)`
  - `reminders.DigestDue(windowStart, windowEnd time.Time, digestAt string, loc *time.Location) (time.Time, bool)`
  - `reminders.Settings` struct and `reminders.ParseSettings(raw []byte) Settings`
  - `reminders.OffsetsForPreset(s Settings, presetName string) []int`

**Context:** `quiet_hours_start` / `quiet_hours_end` already exist in the frontend `UserSettings` type (`web/src/api/auth.ts:28-29`) and currently affect nothing. They are stored inside the `"user".settings` JSONB blob, which is user-writable — so parsing must never panic or error out on garbage. Per-field fallback, exactly like P1's `web/src/lib/quickTask/settings.ts`.

- [ ] **Step 1: Write the failing quiet-hours tests**

`api-go/internal/reminders/quiet_test.go`:

```go
package reminders

import (
	"testing"
	"time"
)

func msk() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}
	return loc
}

func TestQuietHoursLeavesDaytimeAlone(t *testing.T) {
	// 14:00 Moscow, quiet 22:00–07:00 — nothing to do.
	in := time.Date(2026, 8, 1, 11, 0, 0, 0, time.UTC) // 14:00 MSK
	out, ok := ShiftForQuietHours(in, 60, "22:00", "07:00", msk())
	if !ok || !out.Equal(in) {
		t.Fatalf("daytime reminder changed: out=%v ok=%v", out, ok)
	}
}

func TestQuietHoursShiftsToEndOfWindow(t *testing.T) {
	// 02:00 MSK falls inside 22:00–07:00; a day-ahead reminder is still
	// useful at 07:00, so shift rather than drop.
	in := time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC) // 02:00 MSK Aug 1
	out, ok := ShiftForQuietHours(in, 1440, "22:00", "07:00", msk())
	if !ok {
		t.Fatal("day-ahead reminder was dropped, want shifted")
	}
	want := time.Date(2026, 8, 1, 7, 0, 0, 0, msk())
	if !out.Equal(want) {
		t.Errorf("shifted to %v, want %v", out.In(msk()), want)
	}
}

func TestQuietHoursDropsShortNoticeReminders(t *testing.T) {
	// A "15 minutes before" reminder delivered at 07:00 for a 02:15 event is
	// worse than useless — it is a lie. Drop it instead.
	in := time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC) // 02:00 MSK
	if _, ok := ShiftForQuietHours(in, 15, "22:00", "07:00", msk()); ok {
		t.Error("<=15min reminder in quiet hours should be dropped")
	}
	if _, ok := ShiftForQuietHours(in, 16, "22:00", "07:00", msk()); !ok {
		t.Error("16min reminder should still be shifted, not dropped")
	}
}

func TestQuietHoursNonWrappingWindow(t *testing.T) {
	// Not everyone sleeps across midnight. 01:00–07:00 must work too.
	in := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) // 03:00 MSK
	out, ok := ShiftForQuietHours(in, 1440, "01:00", "07:00", msk())
	if !ok {
		t.Fatal("dropped, want shifted")
	}
	if got := out.In(msk()).Hour(); got != 7 {
		t.Errorf("shifted to hour %d, want 7", got)
	}
}

func TestQuietHoursDisabledByEmptyStrings(t *testing.T) {
	in := time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC)
	out, ok := ShiftForQuietHours(in, 15, "", "", msk())
	if !ok || !out.Equal(in) {
		t.Errorf("empty quiet hours must be a no-op, got out=%v ok=%v", out, ok)
	}
}

func TestDigestFiresOnceInLocalMidnightTerms(t *testing.T) {
	// 08:00 MSK on Aug 1 == 05:00 UTC.
	start := time.Date(2026, 8, 1, 4, 59, 0, 0, time.UTC)
	end := time.Date(2026, 8, 1, 5, 1, 0, 0, time.UTC)
	day, ok := DigestDue(start, end, "08:00", msk())
	if !ok {
		t.Fatal("digest should be due in this window")
	}
	// The returned day is the LOCAL midnight the digest belongs to — that is
	// what goes into occurrence_start, so the unique index dedupes per local
	// day rather than per UTC day.
	wantDay := time.Date(2026, 8, 1, 0, 0, 0, 0, msk())
	if !day.Equal(wantDay) {
		t.Errorf("digest day = %v, want %v", day.In(msk()), wantDay)
	}
}

func TestDigestNotDueOutsideWindow(t *testing.T) {
	start := time.Date(2026, 8, 1, 6, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 1, 6, 1, 0, 0, time.UTC)
	if _, ok := DigestDue(start, end, "08:00", msk()); ok {
		t.Error("digest fired an hour late")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
cd api-go && go test ./internal/reminders/ -run 'Quiet|Digest' -v
```
Expected: FAIL — `undefined: ShiftForQuietHours`, `undefined: DigestDue`.

- [ ] **Step 3: Implement quiet.go**

```go
package reminders

import (
	"strconv"
	"strings"
	"time"
)

// quietGraceMinutes: a reminder with this much notice or less is dropped
// rather than shifted. Moving a "15 minutes before" reminder to the end of
// quiet hours would deliver it after the thing it warns about.
const quietGraceMinutes = 15

// parseHHMM turns "08:00" into minutes since local midnight. ok=false for
// anything unparseable — the settings blob is user-writable.
func parseHHMM(v string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(v), ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, false
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// ShiftForQuietHours reports when a reminder should actually be delivered.
// The second return value is false when the reminder should be dropped.
func ShiftForQuietHours(remindAt time.Time, minutesBefore int, quietStart, quietEnd string, loc *time.Location) (time.Time, bool) {
	startMin, okStart := parseHHMM(quietStart)
	endMin, okEnd := parseHHMM(quietEnd)
	if !okStart || !okEnd || startMin == endMin {
		return remindAt, true // quiet hours not configured
	}

	local := remindAt.In(loc)
	nowMin := local.Hour()*60 + local.Minute()

	// The window may wrap midnight (22:00–07:00) or not (01:00–07:00).
	inQuiet := false
	if startMin < endMin {
		inQuiet = nowMin >= startMin && nowMin < endMin
	} else {
		inQuiet = nowMin >= startMin || nowMin < endMin
	}
	if !inQuiet {
		return remindAt, true
	}
	if minutesBefore <= quietGraceMinutes {
		return time.Time{}, false
	}

	// End of the quiet window: today if we are before it, tomorrow if we are
	// in the pre-midnight half of a wrapping window.
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	end := day.Add(time.Duration(endMin) * time.Minute)
	if !end.After(local) {
		end = end.AddDate(0, 0, 1)
	}
	return end, true
}

// DigestDue reports whether the user's local clock crosses digestAt inside
// the half-open window [windowStart, windowEnd). The returned time is the
// LOCAL midnight of the day the digest belongs to; it goes into
// reminder.occurrence_start so the unique index dedupes per local day.
func DigestDue(windowStart, windowEnd time.Time, digestAt string, loc *time.Location) (time.Time, bool) {
	atMin, ok := parseHHMM(digestAt)
	if !ok {
		return time.Time{}, false
	}
	local := windowStart.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

	// Check today and tomorrow: a window can straddle local midnight.
	for _, d := range []time.Time{day, day.AddDate(0, 0, 1)} {
		fireAt := d.Add(time.Duration(atMin) * time.Minute)
		if !fireAt.Before(windowStart) && fireAt.Before(windowEnd) {
			return d, true
		}
	}
	return time.Time{}, false
}
```

- [ ] **Step 4: Run the quiet-hours tests to verify they pass**

```bash
cd api-go && go test ./internal/reminders/ -run 'Quiet|Digest' -v
```
Expected: all seven PASS.

- [ ] **Step 5: Write the failing settings tests**

`api-go/internal/reminders/settings_test.go`:

```go
package reminders

import "testing"

func TestParseSettingsDefaultsOnEmptyBlob(t *testing.T) {
	s := ParseSettings(nil)
	if s.DigestAt != "08:00" {
		t.Errorf("DigestAt = %q, want 08:00", s.DigestAt)
	}
	if !s.QuietHoursRespected {
		t.Error("QuietHoursRespected should default true")
	}
	if got := OffsetsForPreset(s, s.DefaultEventPreset); len(got) != 2 {
		t.Errorf("default event preset = %v, want the 2-entry обычное preset", got)
	}
}

func TestParseSettingsGarbageFallsBackPerField(t *testing.T) {
	// The settings blob is user-writable. A broken digest_at must not take
	// the presets down with it.
	s := ParseSettings([]byte(`{"reminders":{"digest_at":12345,"presets":{"важное":[60]}}}`))
	if s.DigestAt != "08:00" {
		t.Errorf("bad digest_at should fall back, got %q", s.DigestAt)
	}
	if got := OffsetsForPreset(s, "важное"); len(got) != 1 || got[0] != 60 {
		t.Errorf("preset lost: %v", got)
	}
}

func TestParseSettingsMalformedJSONIsNotFatal(t *testing.T) {
	s := ParseSettings([]byte(`{not json at all`))
	if s.DigestAt != "08:00" {
		t.Errorf("malformed JSON should yield defaults, got %q", s.DigestAt)
	}
}

func TestOffsetsForUnknownPresetIsEmptyNotNil(t *testing.T) {
	// The scan ranges over this; nil would work in Go but an explicit empty
	// slice keeps the "no reminders" case identical to the "без" preset.
	got := OffsetsForPreset(ParseSettings(nil), "does-not-exist")
	if got == nil || len(got) != 0 {
		t.Errorf("unknown preset = %v, want empty non-nil", got)
	}
}
```

- [ ] **Step 6: Run to verify failure**

```bash
cd api-go && go test ./internal/reminders/ -run Settings -v
```
Expected: FAIL — `undefined: ParseSettings`.

- [ ] **Step 7: Implement settings.go**

```go
package reminders

import "encoding/json"

// Settings is the `reminders` section of the "user".settings JSONB blob.
// No migration is needed for any of this — settings has been JSONB since
// migration 000005, the same trick P1 used for quick-task defaults.
type Settings struct {
	Presets             map[string][]int
	DefaultEventPreset  string
	DefaultTaskPreset   string
	DigestAt            string
	DigestEnabled       bool
	QuietHoursRespected bool
	QuietHoursStart     string
	QuietHoursEnd       string
}

// DefaultSettings is what a user who has never opened the settings page gets.
func DefaultSettings() Settings {
	return Settings{
		Presets: map[string][]int{
			"важное":  {43200, 10080, 4320, 1440, 60},
			"обычное": {1440, 60},
			"без":     {},
		},
		DefaultEventPreset:  "обычное",
		DefaultTaskPreset:   "обычное",
		DigestAt:            "08:00",
		DigestEnabled:       true,
		QuietHoursRespected: true,
	}
}

// rawSettings mirrors the JSON shape with pointer fields, so "absent" is
// distinguishable from "present but zero" — false must not be overwritten by
// a default of true.
type rawSettings struct {
	Reminders *struct {
		Presets             map[string][]int `json:"presets"`
		DefaultEventPreset  *string          `json:"default_event_preset"`
		DefaultTaskPreset   *string          `json:"default_task_preset"`
		DigestAt            *string          `json:"digest_at"`
		DigestEnabled       *bool            `json:"digest_enabled"`
		QuietHoursRespected *bool            `json:"quiet_hours_respected"`
	} `json:"reminders"`
	QuietHoursStart *string `json:"quiet_hours_start"`
	QuietHoursEnd   *string `json:"quiet_hours_end"`
}

// ParseSettings never fails. The blob is user-writable and a scan that
// panics on one malformed profile would stop reminders for everybody.
func ParseSettings(raw []byte) Settings {
	s := DefaultSettings()
	if len(raw) == 0 {
		return s
	}
	var parsed rawSettings
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Try field-by-field rather than giving up: a single bad value
		// (digest_at as a number) must not discard the user's presets.
		var loose map[string]json.RawMessage
		if json.Unmarshal(raw, &loose) != nil {
			return s
		}
		if rem, ok := loose["reminders"]; ok {
			applyLoose(&s, rem)
		}
		return s
	}
	if parsed.QuietHoursStart != nil {
		s.QuietHoursStart = *parsed.QuietHoursStart
	}
	if parsed.QuietHoursEnd != nil {
		s.QuietHoursEnd = *parsed.QuietHoursEnd
	}
	r := parsed.Reminders
	if r == nil {
		return s
	}
	if len(r.Presets) > 0 {
		s.Presets = r.Presets
	}
	if r.DefaultEventPreset != nil {
		s.DefaultEventPreset = *r.DefaultEventPreset
	}
	if r.DefaultTaskPreset != nil {
		s.DefaultTaskPreset = *r.DefaultTaskPreset
	}
	if r.DigestAt != nil {
		if _, ok := parseHHMM(*r.DigestAt); ok {
			s.DigestAt = *r.DigestAt
		}
	}
	if r.DigestEnabled != nil {
		s.DigestEnabled = *r.DigestEnabled
	}
	if r.QuietHoursRespected != nil {
		s.QuietHoursRespected = *r.QuietHoursRespected
	}
	return s
}

// applyLoose salvages individually-valid fields from a reminders object whose
// strict unmarshal failed.
func applyLoose(s *Settings, rem json.RawMessage) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(rem, &fields) != nil {
		return
	}
	if v, ok := fields["presets"]; ok {
		var presets map[string][]int
		if json.Unmarshal(v, &presets) == nil && len(presets) > 0 {
			s.Presets = presets
		}
	}
	if v, ok := fields["digest_at"]; ok {
		var at string
		if json.Unmarshal(v, &at) == nil {
			if _, valid := parseHHMM(at); valid {
				s.DigestAt = at
			}
		}
	}
	if v, ok := fields["digest_enabled"]; ok {
		var enabled bool
		if json.Unmarshal(v, &enabled) == nil {
			s.DigestEnabled = enabled
		}
	}
}

// OffsetsForPreset resolves a preset name to its offsets. An unknown name
// yields an empty slice, never nil, so callers can range over it and so
// "unknown preset" behaves exactly like the "без" preset.
func OffsetsForPreset(s Settings, name string) []int {
	if offsets, ok := s.Presets[name]; ok && offsets != nil {
		return offsets
	}
	return []int{}
}
```

- [ ] **Step 8: Run every reminders test and the full suite**

```bash
cd api-go && go test ./internal/reminders/ -v && go test ./... && go build ./...
```
Expected: all reminders tests PASS, whole suite green, build clean.

- [ ] **Step 9: Commit**

```bash
git add api-go/internal/reminders/
git commit -m "feat(reminders): quiet hours, digest timing and settings parsing"
```

---

### Task 5: The scan and the worker ticker

**Files:**
- Create: `api-go/internal/reminders/scan.go`
- Create: `api-go/internal/reminders/worker.go`
- Modify: `api-go/cmd/api/main.go`

**Interfaces:**
- Consumes: `DueReminders`, `ShiftForQuietHours`, `DigestDue`, `ParseSettings`, `OffsetsForPreset` (Tasks 3–4); `events.OccurrencesInRange` (Task 3); the `reminder_offsets` columns (Task 1).
- Produces: `reminders.InitDB(db *pgxpool.Pool)`, `reminders.StartWorker(ctx context.Context, log *slog.Logger)` — the ticker, started from `main`.

**Design constraints from the spec:**
- Scan window is `[now − 2 min, now + 1 min)`. The overlap means a skipped tick (deploy, restart) cannot silently lose a reminder; the unique index makes re-scanning free.
- **Users without `tg_id` are skipped at scan time** — their reminders are never created, rather than accumulating as undeliverable rows.
- Tasks that are already `DONE` or `CANCELLED` produce nothing.
- Insert is `ON CONFLICT DO NOTHING` — a conflict means "already sent", not an error.

- [ ] **Step 1: Write scan.go**

```go
package reminders

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"neuroboost/api-go/internal/events"
)

var db *pgxpool.Pool

// InitDB follows the package-level-pool pattern the events and tasks packages
// already use (see cmd/api/main.go).
func InitDB(pool *pgxpool.Pool) { db = pool }

type scanUser struct {
	id       string
	timezone string
	settings []byte
}

// Scan finds everything due in [from, to) and writes PENDING journal rows.
// It returns how many rows it actually inserted, which is what the worker
// logs — a number that should normally be small and is easy to eyeball.
func Scan(ctx context.Context, from, to time.Time, log *slog.Logger) (int, error) {
	// Users without tg_id are skipped here rather than downstream, so we
	// never accumulate rows nobody can deliver.
	rows, err := db.Query(ctx, `
		SELECT id, COALESCE(timezone, 'Europe/Moscow'), COALESCE(settings, '{}')
		FROM "user"
		WHERE tg_id IS NOT NULL`)
	if err != nil {
		return 0, err
	}
	var users []scanUser
	for rows.Next() {
		var u scanUser
		if err := rows.Scan(&u.id, &u.timezone, &u.settings); err != nil {
			rows.Close()
			return 0, err
		}
		users = append(users, u)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	inserted := 0
	for _, u := range users {
		loc, err := time.LoadLocation(u.timezone)
		if err != nil {
			loc = time.UTC
		}
		st := ParseSettings(u.settings)

		n, err := scanEvents(ctx, u, st, loc, from, to)
		if err != nil {
			// One user's bad data must not stop everyone else's reminders.
			log.Error("reminder scan failed for user", slog.String("user_id", u.id), slog.String("error", err.Error()))
			continue
		}
		inserted += n

		n, err = scanTasks(ctx, u, st, loc, from, to)
		if err != nil {
			log.Error("reminder task scan failed", slog.String("user_id", u.id), slog.String("error", err.Error()))
			continue
		}
		inserted += n

		if st.DigestEnabled {
			if day, ok := DigestDue(from, to, st.DigestAt, loc); ok {
				n, err := insertDigest(ctx, u.id, day)
				if err != nil {
					log.Error("digest insert failed", slog.String("user_id", u.id), slog.String("error", err.Error()))
				} else {
					inserted += n
				}
			}
		}
	}
	return inserted, nil
}

// maxOffsetMinutes bounds how far ahead we must look for occurrences: the
// largest offset any preset can use is a month (43200 minutes).
const maxOffsetMinutes = 43200

func scanEvents(ctx context.Context, u scanUser, st Settings, loc *time.Location, from, to time.Time) (int, error) {
	// Look ahead by the largest offset in use: an occurrence up to a month
	// away can have a reminder due right now.
	horizon := to.Add(maxOffsetMinutes * time.Minute)

	rows, err := db.Query(ctx, `
		SELECT id, user_id, title, starts_at, ends_at, all_day, rrule,
		       COALESCE(timezone, 'Europe/Moscow'), reminder_offsets
		FROM event
		WHERE user_id = $1
		  AND cardinality(reminder_offsets) > 0
		  AND (rrule IS NOT NULL OR starts_at BETWEEN $2 AND $3)`,
		u.id, from, horizon)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type candidate struct {
		ev      events.Event
		offsets []int
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.ev.ID, &c.ev.UserID, &c.ev.Title, &c.ev.StartsAt,
			&c.ev.EndsAt, &c.ev.AllDay, &c.ev.Rrule, &c.ev.Timezone, &c.offsets); err != nil {
			return 0, err
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	inserted := 0
	for _, c := range candidates {
		exceptions := fetchEventExceptions(ctx, u.id, c.ev.ID)
		occurrences := events.OccurrencesInRange(c.ev, from, horizon, exceptions)
		for _, d := range DueReminders(occurrences, c.offsets, from, to) {
			remindAt := d.RemindAt
			if st.QuietHoursRespected {
				shifted, ok := ShiftForQuietHours(remindAt, d.MinutesBefore, st.QuietHoursStart, st.QuietHoursEnd, loc)
				if !ok {
					continue
				}
				remindAt = shifted
			}
			n, err := insertReminder(ctx, u.id, "EVENT", &c.ev.ID, nil, d.OccurrenceStart, d.MinutesBefore, remindAt, c.ev.Title)
			if err != nil {
				return inserted, err
			}
			inserted += n
		}
	}
	return inserted, nil
}

func fetchEventExceptions(ctx context.Context, userID, eventID string) []time.Time {
	rows, err := db.Query(ctx,
		`SELECT occurrence_date FROM event_exception WHERE event_id = $1 AND user_id = $2`,
		eventID, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []time.Time
	for rows.Next() {
		var t time.Time
		if rows.Scan(&t) == nil {
			out = append(out, t)
		}
	}
	return out
}

func scanTasks(ctx context.Context, u scanUser, st Settings, loc *time.Location, from, to time.Time) (int, error) {
	horizon := to.Add(maxOffsetMinutes * time.Minute)
	rows, err := db.Query(ctx, `
		SELECT id, title, due_date, reminder_offsets
		FROM task
		WHERE user_id = $1
		  AND due_date IS NOT NULL
		  AND cardinality(reminder_offsets) > 0
		  AND status NOT IN ('DONE', 'CANCELLED')
		  AND due_date BETWEEN $2 AND $3`,
		u.id, from, horizon)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type candidate struct {
		id      string
		title   string
		due     time.Time
		offsets []int
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.title, &c.due, &c.offsets); err != nil {
			return 0, err
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	inserted := 0
	for _, c := range candidates {
		// A task has exactly one "occurrence": its due date.
		for _, d := range DueReminders([]time.Time{c.due}, c.offsets, from, to) {
			remindAt := d.RemindAt
			if st.QuietHoursRespected {
				shifted, ok := ShiftForQuietHours(remindAt, d.MinutesBefore, st.QuietHoursStart, st.QuietHoursEnd, loc)
				if !ok {
					continue
				}
				remindAt = shifted
			}
			id := c.id
			n, err := insertReminder(ctx, u.id, "TASK", nil, &id, d.OccurrenceStart, d.MinutesBefore, remindAt, c.title)
			if err != nil {
				return inserted, err
			}
			inserted += n
		}
	}
	return inserted, nil
}

// insertReminder writes one journal row. A unique-index conflict means we
// already scheduled this exact (occurrence, offset) pair — not an error.
func insertReminder(ctx context.Context, userID, kind string, eventID, taskID *string,
	occurrence time.Time, minutesBefore int, remindAt time.Time, message string) (int, error) {
	tag, err := db.Exec(ctx, `
		INSERT INTO reminder (user_id, source_kind, event_id, task_id, occurrence_start,
		                      minutes_before, remind_at, status, channel, message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING', 'TELEGRAM', $8)
		ON CONFLICT DO NOTHING`,
		userID, kind, eventID, taskID, occurrence, minutesBefore, remindAt, message)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// insertDigest uses local midnight as occurrence_start and the -2 sentinel as
// minutes_before. Both are explicit values rather than NULL: NULLS NOT
// DISTINCT protects the index, but a real value makes the row readable and
// keeps the dedupe key meaningful without relying on index semantics alone.
func insertDigest(ctx context.Context, userID string, localDay time.Time) (int, error) {
	tag, err := db.Exec(ctx, `
		INSERT INTO reminder (user_id, source_kind, occurrence_start, minutes_before,
		                      remind_at, status, channel, message)
		VALUES ($1, 'DIGEST', $2, -2, NOW(), 'PENDING', 'TELEGRAM', '')
		ON CONFLICT DO NOTHING`,
		userID, localDay)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
```

**Sentinel values, stated once so they cannot drift:** `minutes_before = -1` means a snooze (one-off, no offset — used in step 7, a later plan); `minutes_before = -2` means a daily digest. `DueReminders` refuses negative offsets (Task 3), so neither can ever be produced by a scan.

- [ ] **Step 2: Write worker.go**

```go
package reminders

import (
	"context"
	"log/slog"
	"time"
)

const (
	tickInterval = time.Minute
	// The scan window deliberately overlaps previous ticks: a missed tick
	// (deploy, restart, GC pause) must not lose a reminder. The unique index
	// makes re-scanning the same minute free.
	windowBehind = 2 * time.Minute
	windowAhead  = 1 * time.Minute
)

// StartWorker runs the reminder scan once a minute until ctx is cancelled.
// It lives inside the API process rather than in cron because it needs both
// the pgx pool and events.OccurrencesInRange, and both live here.
func StartWorker(ctx context.Context, log *slog.Logger) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	log.Info("reminder worker started", slog.String("interval", tickInterval.String()))
	for {
		select {
		case <-ctx.Done():
			log.Info("reminder worker stopped")
			return
		case now := <-ticker.C:
			from := now.Add(-windowBehind)
			to := now.Add(windowAhead)
			n, err := Scan(ctx, from, to, log)
			if err != nil {
				log.Error("reminder scan failed", slog.String("error", err.Error()))
				continue
			}
			if n > 0 {
				log.Info("reminders scheduled", slog.Int("count", n))
			}
		}
	}
}
```

- [ ] **Step 3: Wire it into main.go**

In `api-go/cmd/api/main.go`, add `rem "neuroboost/api-go/internal/reminders"` to the import block, add `"context"` if absent, then after the existing `InitDB` calls (`exp.InitDB(db)`):

```go
	rem.InitDB(db)

	// The reminder worker runs for the life of the process.
	workerCtx, stopWorker := context.WithCancel(context.Background())
	defer stopWorker()
	go rem.StartWorker(workerCtx, log)
```

- [ ] **Step 4: Build and run the whole suite**

```bash
cd api-go && go build ./... && go test ./...
```
Expected: build clean, all tests pass. `scan.go` has no unit tests by design — it is I/O with the logic already tested in Tasks 3–4; it is verified empirically in the next step.

- [ ] **Step 5: Verify a reminder row actually appears**

Restart the dev API so the worker runs, create an event two minutes out with a 1-minute offset, and watch the journal fill:

```bash
cd "E:/Projects/007 - Ventures/V003 - NeuroBoost"
POSTGRES_PASSWORD=localdev JWT_SECRET=localdevsecretlocaldevsecret32 \
  docker compose -f docker-compose.dev.yml up -d --build api

# tg_id is normally set by Telegram login; set it by hand so the scan sees the user.
docker compose -f docker-compose.dev.yml exec -T db psql -U neuroboost -d neuroboost \
  -c "UPDATE \"user\" SET tg_id = 1 WHERE email = 'p2@local.test';"

# Create an event starting 2 minutes from now with a 1-minute reminder, then wait.
# (Use the $TOKEN from Task 2 step 7, or register again.)
```

After ~2 minutes:
```bash
docker compose -f docker-compose.dev.yml exec -T db psql -U neuroboost -d neuroboost \
  -c "SELECT source_kind, minutes_before, remind_at, status FROM reminder;"
```
Expected: exactly **one** `EVENT` row with `minutes_before = 1`, status `PENDING`. Wait another two minutes and query again — still exactly one row. That second check is the real test: it proves the overlapping window plus the unique index does not duplicate.

- [ ] **Step 6: Commit**

```bash
git add api-go/internal/reminders/scan.go api-go/internal/reminders/worker.go api-go/cmd/api/main.go
git commit -m "feat(reminders): scan worker writes pending reminders once a minute"
```

---

### Task 6: `/api/svc` endpoints and the service token

**Files:**
- Create: `api-go/internal/reminders/service.go`
- Test: `api-go/internal/reminders/service_test.go`
- Modify: `api-go/internal/config/config.go`
- Modify: `api-go/cmd/api/main.go`
- Modify: `.env.example`

**Interfaces:**
- Consumes: the `reminder` journal rows written by Task 5.
- Produces:
  - `reminders.ServiceTokenMiddleware(token string) func(http.Handler) http.Handler`
  - `reminders.PendingHandler` — `GET /api/svc/notifications/pending`
  - `reminders.AckHandler` — `POST /api/svc/notifications/{id}/ack`
  - `config.Config.ServiceToken` from env `SERVICE_TOKEN`

**🔴 The security constraint that shapes this task:** today the bot reaches the API at `http://api:8080` inside the compose network. Once the bot moves to the foreign host (step 10, later plan), this becomes a **public HTTPS endpoint** returning every user's `tg_id` and message text, with a static token as the only defence. Hence:

- the prefix is `/api/svc/`, **not** `/api/internal/` — nobody should read the name and assume network isolation exists;
- token comparison is `crypto/subtle.ConstantTimeCompare`, never `==`;
- the prefix is rate-limited;
- every rejected request is logged with its IP;
- the token is an env var on both sides and **is never written to a file** (project rule: credentials live in the Tracker App).

- [ ] **Step 1: Add ServiceToken to config**

In `api-go/internal/config/config.go`, add `ServiceToken string` to the struct and `ServiceToken: os.Getenv("SERVICE_TOKEN"),` to `Load()`. Use bare `os.Getenv`, not `getEnv` with a default — a default service token is a backdoor.

Add to `.env.example` (the example file only — never `.env`):

```
# Shared secret between the API and the notifier bot. Generate with
# `openssl rand -hex 32`. The /api/svc endpoints are disabled when unset.
SERVICE_TOKEN=
```

- [ ] **Step 2: Write the failing middleware test**

`api-go/internal/reminders/service_test.go`:

```go
package reminders

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestServiceTokenAcceptsCorrectToken(t *testing.T) {
	h := ServiceTokenMiddleware("s3cr3t")(okHandler())
	req := httptest.NewRequest("GET", "/api/svc/notifications/pending", nil)
	req.Header.Set("X-Service-Token", "s3cr3t")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("correct token rejected: %d", rec.Code)
	}
}

func TestServiceTokenRejectsWrongAndMissing(t *testing.T) {
	h := ServiceTokenMiddleware("s3cr3t")(okHandler())
	for _, tc := range []struct{ name, token string }{
		{"wrong", "nope"},
		{"missing", ""},
		{"prefix of the real token", "s3c"},
		{"real token plus suffix", "s3cr3tX"},
	} {
		req := httptest.NewRequest("GET", "/api/svc/notifications/pending", nil)
		if tc.token != "" {
			req.Header.Set("X-Service-Token", tc.token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: got %d, want 401", tc.name, rec.Code)
		}
	}
}

func TestServiceEndpointsDisabledWhenTokenUnset(t *testing.T) {
	// An empty configured token must not mean "anything matches", and must
	// not mean "empty header matches" either. Unset == closed.
	h := ServiceTokenMiddleware("")(okHandler())
	req := httptest.NewRequest("GET", "/api/svc/notifications/pending", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("unset token should close the endpoint, got %d", rec.Code)
	}
}
```

- [ ] **Step 3: Run to verify failure**

```bash
cd api-go && go test ./internal/reminders/ -run ServiceToken -v
```
Expected: FAIL — `undefined: ServiceTokenMiddleware`.

- [ ] **Step 4: Implement service.go**

```go
package reminders

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/util"
)

var svcLog *slog.Logger

// InitService gives the service endpoints their logger. Rejected requests are
// logged with the caller IP because this prefix is reachable from the public
// internet once the notifier moves to a foreign host.
func InitService(log *slog.Logger) { svcLog = log }

// ServiceTokenMiddleware guards the /api/svc prefix with a shared secret.
//
// This is NOT an internal endpoint, whatever it is called. The bot currently
// reaches the API over the compose network, but after the move abroad this is
// a public HTTPS endpoint handing out every user's tg_id and message text,
// and the token is the only thing in the way.
func ServiceTokenMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Unset means closed, never open. A deployment that forgets
			// SERVICE_TOKEN must fail loudly rather than serve everything.
			if token == "" {
				util.RespondError(w, http.StatusServiceUnavailable, "SERVICE_DISABLED",
					"Service endpoints are not configured")
				return
			}
			presented := r.Header.Get("X-Service-Token")
			// Constant time: a byte-by-byte == leaks the token through
			// response timing to anyone who can call this endpoint.
			if subtle.ConstantTimeCompare([]byte(presented), []byte(token)) != 1 {
				if svcLog != nil {
					svcLog.Warn("service token rejected",
						slog.String("ip", r.RemoteAddr),
						slog.String("path", r.URL.Path))
				}
				util.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid service token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimiter is a fixed-window counter over the whole /api/svc prefix. The
// legitimate caller is one bot polling once a minute, so the limit can be
// tight; anything near it is either a bug or an attack.
type rateLimiter struct {
	mu       sync.Mutex
	count    int
	window   time.Time
	limit    int
	duration time.Duration
}

func newRateLimiter(limit int, d time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, duration: d}
}

func (rl *rateLimiter) allow(now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if now.Sub(rl.window) >= rl.duration {
		rl.window = now
		rl.count = 0
	}
	rl.count++
	return rl.count <= rl.limit
}

// RateLimitMiddleware caps the /api/svc prefix at 60 requests a minute.
func RateLimitMiddleware() func(http.Handler) http.Handler {
	rl := newRateLimiter(60, time.Minute)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow(time.Now()) {
				if svcLog != nil {
					svcLog.Warn("service rate limit hit", slog.String("ip", r.RemoteAddr))
				}
				util.RespondError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PendingNotification is what the notifier receives.
type PendingNotification struct {
	ID         string `json:"id"`
	TgID       int64  `json:"tg_id"`
	Text       string `json:"text"`
	SourceKind string `json:"source_kind"`
	SourceID   string `json:"source_id"`
}

const (
	pendingBatchLimit = 100
	// A row handed to the notifier but never acked goes back to PENDING
	// after this long — the notifier died between claim and ack.
	sendingStaleAfter = 5 * time.Minute
)

// PendingHandler claims up to 100 due notifications and marks them SENDING.
//
// remind_at <= NOW() is essential: a snooze row is created with remind_at in
// the future, and without this filter it would be delivered the moment it
// was created.
func PendingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Reclaim rows a dead notifier left behind before claiming new ones.
	// The interval is built from an integer minute count rather than passing
	// a Go duration string: "5m0s" happens to parse as a Postgres interval,
	// but that is a coincidence of two formats, not a contract.
	if _, err := db.Exec(ctx, `
		UPDATE reminder SET status = 'PENDING'
		WHERE status = 'SENDING' AND sent_at < NOW() - make_interval(mins => $1)`,
		int(sendingStaleAfter.Minutes())); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to reclaim stale reminders")
		return
	}

	// Claim and return in one statement so two notifier instances cannot
	// claim the same row.
	//
	// Two comma-separated FROM items, NOT a JOIN. In an UPDATE ... FROM, the
	// update target `r` is not in scope for a JOIN's ON clause, so
	// `... claimed JOIN "user" u ON u.id = r.user_id` is a runtime planner
	// error that go build cannot see.
	//
	// `u.tg_id IS NOT NULL` is repeated here even though the worker already
	// filters on it: the worker filters at scan time, and a user can clear
	// their tg_id between scan and claim. tg_id is a nullable column and
	// n.TgID is a non-pointer int64, so without this the scan panics.
	rows, err := db.Query(ctx, `
		UPDATE reminder r
		SET status = 'SENDING', sent_at = NOW()
		FROM (
			SELECT id FROM reminder
			WHERE status = 'PENDING' AND remind_at <= NOW()
			ORDER BY remind_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		) claimed,
		"user" u
		WHERE r.id = claimed.id
		  AND u.id = r.user_id
		  AND u.tg_id IS NOT NULL
		RETURNING r.id, u.tg_id, COALESCE(r.message, ''), r.source_kind,
		          COALESCE(r.event_id::text, r.task_id::text, '')`,
		pendingBatchLimit)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to claim reminders")
		return
	}
	defer rows.Close()

	out := []PendingNotification{}
	for rows.Next() {
		var n PendingNotification
		if err := rows.Scan(&n.ID, &n.TgID, &n.Text, &n.SourceKind, &n.SourceID); err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to read reminders")
			return
		}
		out = append(out, n)
	}
	util.RespondJSON(w, http.StatusOK, out)
}

type ackRequest struct {
	Delivered bool   `json:"delivered"`
	Error     string `json:"error,omitempty"`
}

// AckHandler records the outcome of one delivery attempt.
func AckHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req ackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	status := "FAILED"
	if req.Delivered {
		status = "SENT"
	}
	if _, err := db.Exec(r.Context(),
		`UPDATE reminder SET status = $1, sent_at = NOW() WHERE id = $2`,
		status, id); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to record delivery")
		return
	}
	if !req.Delivered && svcLog != nil {
		svcLog.Warn("notification delivery failed",
			slog.String("reminder_id", id), slog.String("error", req.Error))
	}
	util.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
```

The import block above deliberately omits `context`: both handlers take their context from `r.Context()`, so nothing in this file names the package.

- [ ] **Step 5: Run the middleware tests**

```bash
cd api-go && go test ./internal/reminders/ -run ServiceToken -v && go build ./...
```
Expected: all four subtests PASS, build clean.

- [ ] **Step 6: Wire the routes into main.go**

In `api-go/cmd/api/main.go`, after `rem.InitDB(db)` add `rem.InitService(log)`, and register the prefix alongside the other route groups:

```go
	// Service endpoints for the notifier bot. Guarded by a shared secret, NOT
	// by network topology — the notifier runs on a foreign host.
	r.Route("/api/svc", func(sr chi.Router) {
		sr.Use(rem.RateLimitMiddleware())
		sr.Use(rem.ServiceTokenMiddleware(cfg.ServiceToken))
		sr.Get("/notifications/pending", rem.PendingHandler)
		sr.Post("/notifications/{id}/ack", rem.AckHandler)
	})
```

Rate limit **before** the token check, so a flood of bad-token requests is cheap to reject.

- [ ] **Step 7: Verify against the running API**

```bash
cd "E:/Projects/007 - Ventures/V003 - NeuroBoost"
SVC=$(openssl rand -hex 32)
POSTGRES_PASSWORD=localdev JWT_SECRET=localdevsecretlocaldevsecret32 SERVICE_TOKEN=$SVC \
  docker compose -f docker-compose.dev.yml up -d --build api

curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8081/api/svc/notifications/pending
# Expected: 401

curl -s -H "X-Service-Token: $SVC" http://localhost:8081/api/svc/notifications/pending
# Expected: JSON array — the PENDING row from Task 5, now claimed.

docker compose -f docker-compose.dev.yml exec -T db psql -U neuroboost -d neuroboost \
  -c "SELECT status FROM reminder;"
# Expected: SENDING
```

Then ack it, substituting the id from the previous response:
```bash
curl -s -X POST -H "X-Service-Token: $SVC" -H 'Content-Type: application/json' \
  -d '{"delivered":true}' http://localhost:8081/api/svc/notifications/<id>/ack
```
Expected: `{"data":{"ok":true}}`, and the row's status becomes `SENT`.

- [ ] **Step 8: Commit**

```bash
git add api-go/internal/reminders/service.go api-go/internal/reminders/service_test.go \
        api-go/internal/config/config.go api-go/cmd/api/main.go .env.example
git commit -m "feat(reminders): service-token-guarded pending/ack endpoints"
```

---

### Task 7: Notifier goroutine in the bot

**Files:**
- Create: `bot/internal/notifier/notifier.go`
- Modify: `bot/internal/config/config.go`
- Modify: `bot/internal/api/client.go`
- Modify: `bot/cmd/main.go`

**Interfaces:**
- Consumes: `GET /api/svc/notifications/pending` and `POST /api/svc/notifications/{id}/ack` from Task 6.
- Produces: `notifier.Start(ctx context.Context, bot *tgbotapi.BotAPI, client *api.Client, interval time.Duration)`.

**Context:** the bot is a **separate Go module** (`github.com/zemdenalex/neuroboost-bot`, `bot/go.mod`) and is not built by `api-go`'s build. Its existing API client (`bot/internal/api/client.go`) sends a user JWT via `token string` parameters; the service endpoints use a different header, so this needs its own method rather than reusing `get`/`post`.

**One process, not two** (spec §7): the notifier is a goroutine inside the existing bot, not a second container. They share the bot token and the Telegram HTTP client; splitting them would mean two deploys and two secrets for one `for` loop.

- [ ] **Step 1: Add the service token to bot config**

In `bot/internal/config/config.go`, add `ServiceToken string` to the `Config` struct and `ServiceToken: os.Getenv("SERVICE_TOKEN"),` in `Load()`.

- [ ] **Step 2: Add service-token methods to the bot API client**

Append to `bot/internal/api/client.go`:

```go
// PendingNotification mirrors the API's /api/svc payload.
type PendingNotification struct {
	ID         string `json:"id"`
	TgID       int64  `json:"tg_id"`
	Text       string `json:"text"`
	SourceKind string `json:"source_kind"`
	SourceID   string `json:"source_id"`
}

// PendingNotifications claims the due notifications. Unlike every other call
// on this client it authenticates with a service token, not a user JWT — it
// deliberately reads across all users.
func (c *Client) PendingNotifications(serviceToken string) ([]PendingNotification, error) {
	req, err := http.NewRequest("GET", c.base+"/svc/notifications/pending", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Service-Token", serviceToken)
	var result []PendingNotification
	if err := c.do(req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// AckNotification records whether delivery succeeded.
func (c *Client) AckNotification(serviceToken, id string, delivered bool, sendErr string) error {
	body, err := json.Marshal(map[string]any{"delivered": delivered, "error": sendErr})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.base+"/svc/notifications/"+id+"/ack", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Service-Token", serviceToken)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, nil)
}
```

Check the existing imports at the top of `client.go` and add `bytes` / `encoding/json` only if they are not already there. Also confirm `c.base` is the field name used by `NewClient` (`client.go:18`) — if it is named differently, use the actual name.

- [ ] **Step 3: Write the notifier**

`bot/internal/notifier/notifier.go`:

```go
// Package notifier delivers reminders computed by the API. It is a goroutine
// inside the existing bot rather than a second service: both need the same
// bot token and the same Telegram client.
package notifier

import (
	"context"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

// Start polls for due notifications until ctx is cancelled.
func Start(ctx context.Context, bot *tgbotapi.BotAPI, client *api.Client, serviceToken string, interval time.Duration) {
	if serviceToken == "" {
		log.Println("notifier: SERVICE_TOKEN is not set, notifications disabled")
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("notifier: polling every %s", interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deliverBatch(bot, client, serviceToken)
		}
	}
}

func deliverBatch(bot *tgbotapi.BotAPI, client *api.Client, serviceToken string) {
	pending, err := client.PendingNotifications(serviceToken)
	if err != nil {
		// Do not ack anything: rows stay SENDING and the API reclaims them
		// after five minutes.
		log.Printf("notifier: pull failed: %v", err)
		return
	}
	for _, n := range pending {
		msg := tgbotapi.NewMessage(n.TgID, n.Text)
		_, sendErr := bot.Send(msg)

		delivered := sendErr == nil
		reason := ""
		if sendErr != nil {
			reason = sendErr.Error()
			log.Printf("notifier: send to %d failed: %v", n.TgID, sendErr)
		}
		// Ack even on failure — an un-acked row would be retried forever,
		// and a user who blocked the bot would generate that retry every
		// minute until someone noticed.
		if err := client.AckNotification(serviceToken, n.ID, delivered, reason); err != nil {
			log.Printf("notifier: ack for %s failed: %v", n.ID, err)
		}
	}
}
```

- [ ] **Step 4: Start it from bot main**

In `bot/cmd/main.go`, after the bot and API client are constructed and before the update loop begins:

```go
	notifierCtx, stopNotifier := context.WithCancel(context.Background())
	defer stopNotifier()
	go notifier.Start(notifierCtx, bot, apiClient, cfg.ServiceToken, time.Minute)
```

Add `"context"`, `"time"` and `"github.com/zemdenalex/neuroboost-bot/internal/notifier"` to the imports. Use the actual variable names `main.go` already uses for the bot and client rather than these placeholders.

- [ ] **Step 5: Build the bot module**

```bash
cd "E:/Projects/007 - Ventures/V003 - NeuroBoost/bot" && go build ./...
```
Expected: clean build. Note this is a **separate module** — `api-go`'s build does not cover it, and CI may not either; check `.github/workflows/ci.yml` and add a bot build step if it is missing.

- [ ] **Step 6: End-to-end verification with a real Telegram message**

This is the only step that proves the feature. Delivery cannot be unit-tested (spec §9: "Доставка — интеграционно, вручную на staging").

1. Set `tg_id` on the dev user to your own Telegram numeric id.
2. Run the bot locally against the dev API with the same `SERVICE_TOKEN` and `API_BASE=http://localhost:8081/api`.
3. Create an event three minutes out with `reminder_offsets: [2]`.
4. Expect a Telegram message within ~60 seconds of the two-minute mark, and the row's status to become `SENT`.

If no message arrives, check in this order: is there a `PENDING` row (worker problem, Task 5); does `pending` return it (endpoint or token problem, Task 6); does the bot log a send error (Telegram reachability — this is exactly the RU-hosting blockage that step 10 solves by moving the bot to the foreign host).

- [ ] **Step 7: Commit**

```bash
git add bot/internal/notifier/ bot/internal/api/client.go bot/internal/config/config.go bot/cmd/main.go
git commit -m "feat(bot): notifier goroutine pulls, sends and acks reminders"
```

---

## After all seven tasks

Reminders now arrive. What remains from the spec (a later plan): buttons and callbacks (§6, step 7), the `<ReminderOffsets/>` component in the event editor, task form and Settings (§8, steps 8–9), and the foreign-host deploy (step 10) — which is what actually makes delivery work from Russia.

**Before any of that:** the `SERVICE_TOKEN` must exist in the Tracker App and in the staging environment, or the `/api/svc` endpoints return 503 on staging and the notifier logs "notifications disabled" and exits.

## Self-Review

**Spec coverage (§10 steps 1–6):** step 1 → Task 1; step 2 → Task 3; step 3 → Task 4; step 4 → Task 5; step 5 → Task 6; step 6 → Task 7. Task 2 has no step number in the spec — it is implied by §4.1 and is required for anything downstream to be reachable; without it `reminder_offsets` is unreachable and every scan finds nothing.

**Gaps knowingly left:** §9's "миграция: непустая `reminder` переживает добавление уникального индекса" is verified empirically in Task 1 steps 3–5 rather than by an automated test — this codebase has no DB-backed test harness, and adding one is out of scope here. §6's buttons are step 7, out of scope. The `-1` snooze sentinel is defined and defended against in Tasks 3 and 5 but not used until step 7.

**Type consistency:** `Due{OccurrenceStart, MinutesBefore, RemindAt}` is produced in Task 3 and consumed in Task 5. `Settings` / `ParseSettings` / `OffsetsForPreset` are produced in Task 4 and consumed in Task 5. `PendingNotification{ID, TgID, Text, SourceKind, SourceID}` is defined identically in Task 6 (Go, API side) and Task 7 (Go, bot side) — they are separate modules and cannot share the type. `ReminderOffsets` is the field name in Tasks 1, 2 and 5 throughout.

**One thing the implementer must not assume:** `OffsetsForPreset` exists and is tested, but **nothing calls it yet**. Applying the default preset at create time (spec §4.3: "Кто применяет пресет — бэкенд, при создании") is deliberately left to the UI plan, because until there is a settings editor every user has the same defaults and the distinction is untestable in a browser. If you want it now, it belongs in Task 2 step 4: when `req.ReminderOffsets == nil`, resolve `OffsetsForPreset(st, st.DefaultEventPreset)` instead of `[]int{}`.
