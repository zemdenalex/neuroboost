<!-- паспорт: тип=документ | статус=действует | строк=212 | ~токенов=5778 | обновлён=по git -->

# D1 «задачи дня» — итоги сборки (23.09.2026)

**Что это:** API задач дня собран по плану `docs/superpowers/plans/2026-09-22-day-tasks-d1-api.md`, коммиты `dcd80d8..da3b244` на `develop`, **не запушено**. Ниже — все решения, которые я принял сам (`Ruling:`), и отложенные мелкие замечания. Полный разбор ревьюера — секция в конце.

✅ **Решено Денисом 23.09:** смена N посреди дня к сегодняшнему дню **не применяется** — цель фиксируется в момент «✅ Беру», новая действует с завтра (как и собрано).

## Решения и починки (ledger)

- Task 1: complete (commits 956b16d..dcd80d8, tests: bash -c 'cd api-go && go test ./internal/daytasks/ -count=1' → ok  	neuroboost/api-go/internal/daytasks	0.538s)
- Task 2: Ruling: brief comment «sameOrder compares…» names no function → «ymd compares…» — comment matches code — cost if wrong: none.
- Task 2: Ruling: brief sabotage (now.Hour()) leaves loc unused → does not compile; used «_ = loc; return now.Hour() < 12» instead — sabotage must compile — cost if wrong: none, failed on «server already on the 23rd» as brief expects.
- Task 2: Ruling: gofmt realigned the map literal in rules_test.go — brief formatting was not gofmt'd — cost: none.
- Task 2: complete (commits dcd80d8..8e64030, tests: bash -c 'cd api-go && go test ./internal/daytasks/ -count=1' → ok  	neuroboost/api-go/internal/daytasks	0.590s)
- Task 3: Ruling: brief's extra check for sabotage 1 («closed today, List on yesterday not counted») can fail in no hour — NY's date is UTC's or one behind, never ahead — so replaced with TestADayIsTheUsersDateNotUTCs: zone picked by the hour (UTC−12 / UTC+14) so its date ≠ UTC's at run time — cost if wrong: one extra test.
- Task 3: Ruling: brief's sabotage 1 (drop «AT TIME ZONE $TZ») leaves $4 unbound → SQL error, all 8 DB tests red for the wrong reason; ran it as «AT TIME ZONE 'UTC' … AND $TZ::text IS NOT NULL» → only TestADayIsTheUsersDateNotUTCs red (at 04:14 UTC the NY test passed, as predicted) — cost: none.
- Task 3: Ruling: added "strings" import to store.go per brief note; gofmt -w store.go (doc-comment indentation) — cost: none.
- Task 3: complete (commits 8e64030..afafd4e, tests: bash -c 'cd api-go && DATABASE_URL="$DATABASE_URL" go test ./internal/daytasks/ -count=1 -v 2>&1 | grep -c "^--- PASS" && DATABASE_URL="$DATABASE_URL" go test ./internal/daytasks/ -count=1' → ok  	neuroboost/api-go/internal/daytasks	1.109s)
- Task 4: Ruling: brief contingency applied — $2=nil untyped (SQLSTATE 42P18), renumbered to $1…$5 exactly as the brief prescribes — cost: none.
- Task 4: complete (commits afafd4e..50065bb, tests: bash -c 'cd api-go && go test ./internal/daytasks/ -count=1 -v 2>&1 | grep -E "^--- (FAIL|SKIP)" ; go test ./internal/daytasks/ -count=1' → ok  	neuroboost/api-go/internal/daytasks	0.169s)
- Task 4: CORRECTION — the task-done run above had no DATABASE_URL (0.169s = DB tests skipped). Re-run with $DATABASE_URL: 14 PASS, 0 SKIP, 0 FAIL, ok ~2s. From Task 5 on, task-done command sets DATABASE_URL inline.
- Task 5: Ruling: added «bad task id» case (non-UUID → 404 TASK_NOT_FOUND); it was red (500) → Add maps SQLSTATE 22P02 to ErrTaskNotFound via local pgErrCode (same helper shape as calendars/crud.go) — the brief's ⚠ anticipated it — cost if wrong: none.
- Task 5: Ruling: full suite red on accounts.TestMergeKnowsEveryForeignKeyOnUser — day_commitment(.day_commitment_day).user_id unknown to the merge, rows would vanish by cascade. Added both as MoveOrDrop (survivor's row wins on key clash) + TestMergeKeepsDayTasksOfBothAccounts RED→GREEN; sabotage (dedupe off) → 23505 — plan did not foresee the guard — cost if wrong: a merged account loses/duplicates day promises.
- Task 5: Ruling: calendars.TestHandlersDoNotScopeQueriesByUserID red on proposal.go (c.user_id in a block with FROM task). Split reads: promises by user_id (personal table, no task ref), tasks only via calendar_id = ANY(CalendarIDsFor). Same split in List, which the scanner MISSED (fragment after `+replaceAll+` has no SELECT keyword) and which really leaked titles of tasks from calendars the user left: TestListShowsOnlyTasksStillVisible RED→GREEN, sabotage (filter off) red — cost if wrong: none; spec says access = CalendarIDsFor.
- Task 5: complete (commits 50065bb..f281ac5, tests: bash -c 'cd api-go && export DATABASE_URL="$DATABASE_URL"; go test ./internal/daytasks/ ./internal/accounts/ ./internal/calendars/ -count=1 -v 2>&1 | grep -cE "^--- (FAIL|SKIP)" ; go test ./internal/daytasks/ ./internal/accounts/ ./internal/calendars/ -count=1' → ok  	neuroboost/api-go/internal/calendars	4.813s)
- Final: fixed C1 reminder ✅ did not stamp completed_at → List 500 (NULL into bool) — TestAReminderDoneCountsForTheDay RED→GREEN (through reminders.ActionHandler), + doneOnDay made total — TestADoneTaskWithoutAStampDoesNotBreakTheDay (sabotage red).
- Final: fixed I1 target read live → day_commitment_day.target snapshot at Confirm (000022 amended; not on any remote branch — checked) — TestChangingTheTargetDoesNotRecolourATakenDay RED→GREEN.
- Final: fixed I2 Confirm non-atomic + refused a pinned task done since → one tx, kept-set skips the open check — TestConfirmKeepsAPinnedTaskThatIsAlreadyDone, TestConfirmWritesNothingWhenOneTaskIsRefused RED→GREEN.
- Final: fixed I3 past days writable → Add/Confirm refuse past day (ErrTooLate) — TestAPastDayCannotBeTakenOrAddedTo RED→GREEN; older tests needing yesterday now seed it directly (seedPromise).
- Final: fixed I4 CANCELLED treated as open → closed in checkOpen and Propose — TestACancelledTaskIsNeitherProposedNorAddable RED→GREEN.
- Final: fixed I5 series on non-occurrence days + done-yesterday carried → checkOpen checks occursOn (new ErrNotAnOccurrence → 409 NOT_AN_OCCURRENCE); Propose skips non-occurring series, carries only undone — TestASeriesEntersOnlyItsOwnDays, TestASeriesDoneYesterdayIsNotCarried, TestASeriesThatDoesNotOccurTodayIsNotCarried RED→GREEN.
- Final: sabotage battery 11/11 red on their own tests (BAD 0). Suites: api-go 15 ok (DB, -count=1), bot 11 ok.
- Final: Ruling: I1 snapshot freezes TODAY's target at the moment of ✅ Беру too — whether a mid-day N change should apply to today is Denis's call (reviewer flagged it) — cost if wrong: one line in Confirm/List.
- Final: minor (deferred): M1 DELETE non-UUID → 500 · M2 62-day limit off by one · M3 merge dedupe ignores removed_at · M4 (fixed in passing: confirmed.Err() now checked) · M5 zone change recolours history · M6 PATCH DONE re-stamps completed_at (outside D1) · M7 scoping scanner blind to concatenated queries.

## Разбор ревьюера (final review, дословно)


Reviewer: final whole-branch seat. Read-only: nothing edited, nothing run against a database or server.
Method: `requesting-code-review/code-reviewer.md`. One pass over the diff, then reads of the surrounding code
(`tasks/handlers.go`, `tasks/occurrence.go`, `reminders/action.go`, `recurrence/rrule.go`,
`calendars/store.go`, migrations 000001/000006/000017, bot due-date writers) wherever a D1 rule depends on it.

**Counts:** Critical 1 · Important 5 · Minor 7 · Declined to judge 6

## Strengths

- `Level` is one pure function with a table test for every N (3…7) plus the target 0 guard. That matches spec §3 exactly.
- `CanRemove` is tested at 11:59/12:00 in New York, including the case where the server is already on the next day in UTC. `Today` is computed in the user's zone.
- `TestADayIsTheUsersDateNotUTCs` picks a zone whose date differs from UTC's at run time. The ledger's ruling (line 13) is right: the brief's version could never fail.
- Tests go through the real doors (`tasks.CreateHandler`, `UpdateHandler`, `MarkOccurrence`), following the 20.09 lesson.
- Access split (ledger line 23): promises are read by `user_id` and tasks by `calendar_id = ANY(CalendarIDsFor)`. The executor found and fixed a real title leak in `List` that the static scanner missed.
- Add returns the same 404 for an invisible task and a non-existent one, so it does not reveal whether a task exists.
- The executor caught that the account merge would otherwise lose day-tasks rows by cascade, and added a test for it.
- Arrays: `List` pre-fills `Items: []Item{}`, `Propose` starts from `[]Item{}`, and `List` always returns at least one day.

## Issues

### Critical

**C1. A task closed from the reminder ✅ Done button makes `List` return 500 for every range containing that day. Its day never counts it either.**
- Where: `api-go/internal/daytasks/store.go:64` (the `doneOnDay` ELSE branch), `store.go:160` (Scan into `&it.Done`), and the root cause outside D1 at `api-go/internal/reminders/action.go:264`.
- What happens:
  1. `reminders/action.go:264` runs `UPDATE task SET status = 'DONE', updated_at = NOW()` and does not set `completed_at`. It is the only task-closing door besides PATCH, and PATCH does stamp it (`tasks/handlers.go:616-619`).
  2. For that row, `doneOnDay` evaluates `t.status = 'DONE' AND (NULL AT TIME ZONE tz)::date = day`, which is `TRUE AND NULL`, so the CASE yields **NULL**.
  3. `List` scans that column into a plain `bool` (`&it.Done`, `store.go:160`). pgx v5 (v5.5.5 in go.mod) refuses NULL into a non-pointer `bool`, so the error surfaces as `DB_ERROR` 500.
- Scenario:
  1. Promise task X for today and confirm the day.
  2. The reminder for X arrives, and the user presses ✅ Done in Telegram.
  3. `GET /api/day-tasks?from=today&to=today` returns 500.
  4. Every later write (`Add`, `Confirm`, `Remove`) on that day also returns 500, because each responds through `respondDay` → `List`.
  5. A month view (up to 62 days) containing that day returns 500 as well.
- Even if the scan tolerated NULL, the task would still count as "not done": the day goes red for a task the user closed.
- Not affected: `Add` (`NULL OR TRUE` gives TRUE, so it refuses with NOT_OPEN, which is correct) and `Propose` (`NOT NULL` filters the row out, and the DONE clause excludes it anyway).
- Confidence:
  - High on the three-valued-logic chain.
  - High that pgx v5 errors on NULL into `*bool` ("cannot scan NULL into *bool").
  - Not run against a database.
  - No test covers it, because every test closes tasks through PATCH.
- Fix, both halves:
  1. In `reminders/action.go:264`, add `completed_at = NOW()`. Series tasks should not be closed through this path at all; that is a pre-existing problem, not D1's.
  2. Make `doneOnDay` total, so a row without a stamp reads as false rather than NULL: wrap it in `COALESCE((…), false)`, or add `AND t.completed_at IS NOT NULL` in the ELSE branch.
  3. Add a DB test that closes a promised task through the reminder action (or through the same UPDATE) and then calls `List`.
- Related: `tasks.CreateHandler` (`tasks/handlers.go:525`) also accepts `status: DONE` without `completed_at`. Such a task cannot enter a day, because Add refuses it, so it is harmless here. The reminder door is the one that matters.

### Important

**I1. The target N is read live and applied to every day, past days included. Changing the setting recolours history and turns today green at 23:00.**
- Where: `store.go:75` (`userZoneAndTarget` → current `settings.day_tasks_target`), `store.go:175` (`Level(out[i].Done, out[i].Target)`), and migration `000022_day_commitment.up.sql` (`day_commitment_day` has no target column).
- Scenario:
  1. At 23:00 the user has done 3 of 5, so the day is 🟧.
  2. They PATCH `settings.day_tasks_target = 3`.
  3. Today becomes 🟩 (`Level(3,3) = 5`), and every past confirmed day is recomputed under N=3. A 3-of-5 day from last week turns green.
  4. The reverse (N=7) turns past green days orange.
- This is the real route around "the colour must mean something". The noon rule does not close it, and as implemented the noon rule protects nothing: `Level` divides by the fixed N, not by the set size, so removing a task can never raise a day's level. The spec §5 rationale ("выкинул невыполненное — и день позеленел") would only hold if the target were the set size.
- This is a spec gap as well as a code gap: §7's schema has no per-day target either.
- Fix:
  - Snapshot the target at Confirm: add a `target SMALLINT NOT NULL` column to `day_commitment_day`, written with the current N (on first confirm; decide whether a re-confirm may update it).
  - `List` uses the stored value for confirmed days and the current N only for unconfirmed ones (level 0 anyway).
  - `000022` is not on `origin/develop`: `git branch -r --contains afafd4e` returns nothing. So it can still be amended rather than needing a 000023 (the project rule against editing migrations protects applied ones). Check staging before choosing.
- Denis should decide whether a mid-day N change applies to today.

**I2. Confirm is not atomic, and it fails with NOT_OPEN on a pre-pinned task that has since been done. The ✅ Беру press then leaves the day unconfirmed.**
- Where: `store.go:234-243` (`Confirm` loops `Add`), and `store.go:195` (Add's open-check applies even to a row already in the set).
- Scenario 1, a stale proposal:
  1. The 08:00 digest proposes [A (pinned yesterday), B, C, D, E].
  2. At 09:00 the user closes A.
  3. At 09:30 they press ✅ Беру, which sends all five ids.
  4. `Add(A)` returns NOT_OPEN → 409.
  5. The day is never taken (no `day_commitment_day` row), so it stays ⬛ however much gets done. This is spec §1's "не взят → ⬛" triggered by a race, not by the user's choice.
- Scenario 2, a partial write:
  1. `task_ids = [B, C, "not-a-uuid"]`, or C is a task the user just lost access to.
  2. B is inserted, then 404 is returned.
  3. B is now in the day, the day is not confirmed, and the client was told the request failed.
- Fix:
  - Run Confirm in one transaction.
  - In Confirm, skip the open-check for ids already present in `day_commitment` for that day with `removed_at IS NULL`. Keeping a promised task is not adding a done one.
  - Validate every id (UUID, visible, open) before any write.
- Add a test: pin A, close A, then Confirm [A, B] → 200, day confirmed, A counted.

**I3. Past days can be written retroactively. `Confirm` on yesterday turns an untaken ⬛ day into a colour, and `Add` on a past day accepts series whose past occurrence can then be marked.**
- Where:
  - `handlers.go` `ConfirmHandler` / `AddHandler`: no check of `day` against the user's today.
  - `store.go:183` (`Add`) and `store.go:234` (`Confirm`) accept any date.
- Scenario 1:
  1. The user pins five tasks for Monday in advance (spec §1.1), never presses ✅ Беру on Monday, and does all five.
  2. On Tuesday, `POST /api/day-tasks/confirm {"day":"<Monday>","task_ids":[]}` makes Monday 🟩.
  3. Spec §1.3 and Denis 22.09 say an untaken day is ⬛ ("не выбрал = не сделал"). The server computes the colour, so it has to enforce this; the bot's UI cannot.
- Scenario 2:
  1. `Add(yesterday, dailySeries)` succeeds, because yesterday's occurrence was not answered.
  2. Then `POST /api/tasks/{id}/occurrences` for yesterday (a named date that is an occurrence is accepted by `MarkOccurrence`) raises yesterday's done count after the fact.
- Spec §5 speaks only of "after the start of the day" and future days; it says nothing about past days. A reasonable reader of "past day never" (the Remove rule) expects the day to be closed to all edits.
- Fix: in `Add` and `Confirm`, refuse `ymd(day) < ymd(Today(now, tz))` with `ErrTooLate` → 409 TOO_LATE. The zone is already loaded in `Add`. Test: Add or Confirm on yesterday → 409.

**I4. Cancelled tasks are proposed and can be promised.**
- Where: `proposal.go:70` (`NOT (non-series AND status = 'DONE')`) and `store.go:195` (the same check in Add).
- The enum is `('TODO','IN_PROGRESS','SCHEDULED','DONE','CANCELLED')` (`000001_baseline.up.sql:11`). The rest of the codebase treats CANCELLED as closed: `planning/handlers.go:158`, `reminders/scan.go:245`.
- Scenario:
  1. The user cancels "купить билеты" (status CANCELLED).
  2. The next morning's proposal offers it (rank 5 by priority, or rank 3 if it had a due date, which is likely for a cancelled errand).
  3. If accepted, it sits in the day and can never be done, so it holds the day below 🟩 unless removed before noon.
  4. `Add` also accepts it directly.
- Fix: treat `status IN ('DONE','CANCELLED')` as not open for one-offs, in both `Propose` and `Add`. Better still, define one `isOpen` SQL fragment next to `doneOnDay` so the two cannot drift. Add a test: a cancelled task is neither proposed nor addable.

**I5. A series can enter a day on which it does not occur, and then that day can never be completed. The proposal also carries a series done yesterday as "yesterday's undone".**
- Where:
  - `proposal.go:51-55`: every task promised yesterday, with no "undone yesterday" filter.
  - `proposal.go:90-93`: `case pinned` / `case yesterday` come before the series occurrence check.
  - `store.go:183-213`: `Add` never calls `recurrence.Occurs`.
- Scenario (weekly series, e.g. "FREQ=WEEKLY" anchored on a Monday):
  1. On Monday it is promised and marked done.
  2. Tuesday's proposal puts it at **rank 2, "yesterday's undone"**. That is wrong on both counts: it was done, and Tuesday is not in the series.
  3. The user confirms.
  4. Tuesday's item can never become done. `MarkOccurrence` refuses a named non-occurrence date (`tasks/occurrence.go:124`), and the bot's ✅ press slides to the next occurrence (`PressedDay`). So Tuesday is capped below 🟩 by a task the proposal itself chose.
- The spec (§4.2) says "вчерашние **невыполненные**". For one-offs the filter happens to work (a done one-off is excluded as DONE), but a series' "done yesterday" lives in `task_occurrence` for yesterday, which the query never checks. It checks `doneOnDay` for **today** (`$2`).
- Fix:
  - In `Propose`, build `yesterdayIDs` only from rows where `NOT doneOnDay(yesterday)`, for example by joining the fragment with `$DAY = $3::date` in the promises query (the task read must stay calendar-scoped).
  - For a series at rank 1 or 2, also require `occursOn(rrule, anchor, day)`.
  - In `Add`, refuse a series on a non-occurrence day. This is a new `ErrNotAnOccurrence` → 409, mirroring `tasks.ErrNotAnOccurrence`.
- Tests:
  - A weekly series done Monday is absent from Tuesday's proposal.
  - Add of a series on a non-occurrence day → 409.

### Minor

**M1. DELETE with a non-UUID `task_id` → 500.**
- Where: `handlers.go:170` → `store.go:226`.
- `DELETE /api/day-tasks/2026-09-23/abc` → `task_id = 'abc'` → 22P02 → `DB_ERROR` 500.
- Already in the ledger (line 25), deferred. Fix: map 22P02 to 404 as in `Add`, or check the UUID in the handler. Confirm needs the same treatment for I2.

**M2. The 62-day limit is off by one.**
- Where: `store.go:72`. `to.Sub(from) > 62*24h` allows `from=01.01, to=04.03`, which is 63 days inclusive.
- Spec §7: "≤ 62 дня". Fix: `> 61*24h`, or count days (`ymd` difference + 1 > 62). Add a boundary test at 62 and 63.

**M3. Merge dedupe ignores `removed_at`, so an active promise can lose to a removed one.**
- Where: `accounts/merge.go:265-268`.
- Scenario: the survivor promised shared task T for the 22nd and removed it before noon (`removed_at` set). The absorbed account promised T for the 22nd and kept it (active). The merge deletes the absorbed (active) row and keeps the survivor's removed row, so T silently drops out of the merged person's 22nd.
- Fix: before the delete, `UPDATE` the survivor's row with `removed_at = LEAST-or-NULL` (the row stays active if either was active, and `added_at = LEAST(...)`). Or delete whichever of the pair is removed.

**M4. The rows error from the confirmed-days query is not checked.**
- Where: `store.go:106`. `confirmed.Close()` is not followed by a `confirmed.Err()` check (the `prom` query does have one). A mid-stream error would silently mark days unconfirmed, which is level 0. Fix: add the `Err()` check.

**M5. A change of timezone recolours history.**
- Where: `store.go:64`, where `completed_at` is converted with the user's *current* zone.
- Scenario: a user who moves from Moscow to New York sees tasks closed at 01:00 MSK move to the previous day in every past day.
- Same family as I1: a day is judged by today's settings. If I1 adds a per-day snapshot, the zone can be snapshotted the same way. Otherwise document it.

**M6. PATCH `status: DONE` on an already-done task re-stamps `completed_at`.**
- Where: `tasks/handlers.go:616-619`, outside D1.
- A task closed Monday and re-sent `DONE` on Tuesday (any client that resends status) moves its count from Monday to Tuesday. The current web edit modal does not send `status` (`Tasks.tsx:268`), so this is latent. Fix: `completed_at = COALESCE(completed_at, $n)` when the status is already DONE, or only stamp on a TODO→DONE transition.

**M7. The static scoping scanner cannot see queries concatenated from fragments.**
- Already deferred in the ledger (line 26).
- In `store.go:146-151` and `proposal.go:63-70`, the scoped `WHERE` sits in a fragment the scanner does not classify. So a future edit that drops `calendar_id = ANY` from `List` would pass the guard.
- `TestListShowsOnlyTasksStillVisible` covers `List` behaviourally. Nothing behavioural covers `Propose`'s scoping. Worth one test: another user's open task is not proposed.

## Declined to judge

- **Other members' tasks from a shared calendar fill the proposal (rank 5).** Visibility equals calendar membership project-wide, and `TestHandlersDoNotScopeQueriesByUserID` forbids `user_id` scoping on `task`. Whether "my day" should prefer my own tasks is a product decision for Denis.
- **Subtasks (`parent_id`) are proposed alongside their parents.** The spec is silent, and whether a subtask is a "thing to do today" is a product call.
- **SCHEDULED (converted to an event) counts as open.** It is not done, so it is open by spec §2. Whether a task already on the calendar belongs in the proposal is a product call.
- **Another member closing a shared task or series occurrence counts for my day.** This follows from `task_occurrence` belonging to the series (000017) and from spec §7's "из видимого календаря". It is consistent, not a defect.
- **The noon rule's value.** As shown in I1, removal can never raise a level under a fixed N. I did not grade the spec's rule itself; if I1 is fixed with a snapshot, the rule remains harmless UX.
- **A skipped series day (`state='skipped'`) is still proposed at rank 4.** The spec does not say whether "skipped" means "not today's task".

## Recommendations

- Define one `openOn(day)` SQL fragment beside `doneOnDay`, used by both `Add` and `Propose`. I4 and I5 are both drift between two hand-written "is it open" checks.
- For every door that closes a task (PATCH, reminder action, future bot paths), add a test that closes a *promised* task through that door and reads `List`. C1 exists because the tests only used the PATCH door.

## Assessment

**Ready to merge: With fixes.** C1 blocks: one press of the reminder ✅ breaks the day screen, and every screen that contains that day, with a 500. I1–I5 make the colour wrong in ordinary use (settings change, stale ✅ Беру, cancelled or weekly tasks). All of them are local fixes with a test each. The core (Level, noon, user-zone dates, access split) is sound.
