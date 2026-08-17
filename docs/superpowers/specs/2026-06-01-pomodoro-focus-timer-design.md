<!-- паспорт: тип=спека | статус=справка | строк=214 | ~токенов=3523 | обновлён=по git -->

# Pomodoro / Focus Timer — Rebuild Design

> **Status:** Approved design, pending implementation plan
> **Date:** 2026-06-01
> **Version slot:** v0.4.10 (Intelligent Tools era) — first concrete slice
> **Scope chosen:** Everything (rebuild + tracking + 3-style widget + backend)

---

## 1. Problem

The current `web/src/pages/Tools/Pomodoro.tsx` has three defects that make the tool effectively unusable:

1. **Resets on navigation and reload.** All live timer state (`mode`, `timeLeft`, `isRunning`, `sessionsCompleted`, `linkedTaskId`) is component `useState`. Navigating to another route unmounts the page; reload reboots the app. Only `settings` + `history` persist to localStorage. Nothing records "you were N minutes into a block."

2. **Phases do not advance.** The interval calls `handleTimerEnd()` *from inside the `setTimeLeft` updater* (which itself dispatches ~6 `setState` calls including another `setTimeLeft`), then the same updater `return 0`. Calling state setters from inside a state-setter's updater is React undefined behavior; the race between `return 0` and the queued `setTimeLeft(nextTotal)` means the work block ends but neither switches to the break nor increments the cycle counter — it snaps back to the start of work. The session indicator already existed; it never moved because the underlying transition never ran correctly.

3. **No tracking anywhere.** A completed block is appended only to a localStorage array (`nb-pomodoro-history`). No calendar event, no actual-time on the task, nothing the rest of the app can see.

Root cause of (1) and (2) is the same architectural mistake: **the source of truth is a number being decremented every second.** A correct timer stores the moment it should end and measures against the wall clock.

## 2. Goals / Non-goals

**Goals**
- Timer keeps correct time across navigation, reload, and backgrounded tabs.
- Phases advance deterministically; the cycle visibly progresses to the long break.
- A completed work block writes a calendar event **and** logs actual minutes onto the linked task, with one-click undo.
- The running timer is visible and controllable from any in-app page via a floating widget; its style is user-selectable.

**Non-goals (this release)**
- Backend/server-side session storage or cross-device sync (YAGNI).
- Exact at-the-moment notification while the tab is closed (needs a service worker — deferred).
- Reworking history/stats analytics beyond the today summary.

## 3. Locked decisions (from brainstorm)

| Decision | Choice |
|----------|--------|
| Timer scope | **Global** React Context + floating widget; survives nav + reload |
| Tracking | Calendar **event** + **actual minutes** logged on the task |
| On completion | **Auto-save** with a quiet **Undo** toast |
| Widget styles | All three (**pill / card / bar**), selectable in settings, default **card** |
| Cycle display | New **pips** row (`●●◉○`) replacing the old "session X of Y" text |
| Release scope | Everything in one slice |

## 4. Architecture

### 4.1 Global timer context

A new `PomodoroContext` (mirrors `AuthContext`/`ThemeContext` — **no new dependency**, Context not Zustand) mounted inside `AppLayout` so it persists across route changes for authenticated pages.

Exposes via `usePomodoro()`:

```ts
{
  phase: 'work' | 'shortBreak' | 'longBreak'
  remainingMs: number              // derived: endsAt - now (or remainingWhenPaused)
  isRunning: boolean
  sessionsCompleted: number        // total completed work blocks this set-cycle
  linkedTaskId: string | null
  settings: PomodoroSettings
  lastCompletion: Completion | null // drives the undo toast
  // actions
  start(): void; pause(): void; reset(): void; skip(): void
  selectPhase(p): void; setLinkedTask(id): void
  updateSettings(patch): void
  undoLastCompletion(): void
}
```

### 4.2 Timestamp model

Source of truth is **`endsAt` (epoch ms)**, not a countdown.

- `remainingMs = isRunning ? max(0, endsAt - Date.now()) : remainingWhenPaused`
- A single `setInterval(…, 1000)` in the provider only forces a re-render (display refresh). It is **not** authoritative — if it's throttled in a background tab, `remainingMs` is still correct on next fire/focus.
- Pause: `remainingWhenPaused = endsAt - Date.now()`, then clear running.
- Resume: `endsAt = Date.now() + remainingWhenPaused`.

### 4.3 Phase state machine (bug-2 fix)

A `useEffect` watches `remainingMs`. When `isRunning && remainingMs <= 0`, it invokes `completePhase()` **once**, guarded by a ref holding the `endsAt` already consumed (so a re-render can't double-fire). `completePhase()`:

1. Build the completed block record (`phase`, `blockStartedAt`, `endedAt = endsAt` — the **scheduled** end, never `Date.now()`; see §5 for why), `taskId`).
2. If `phase === 'work'`: `sessionsCompleted += 1`, then fire **tracking** (§5).
3. Compute next phase from the transition table using the **post-increment** `sessionsCompleted`.
4. Set next `endsAt` (or `remainingWhenPaused` if not auto-starting).
5. Persist (§4.4). Auto-start the next block if the relevant setting is on.

Transition table (with `n` = `sessionsCompleted` after the increment in step 2):

```
work            → n % sessionsBeforeLong === 0 ? longBreak : shortBreak
short/longBreak → work
```

**`skip()` and `reset()` reuse only the transition/seed logic — they do NOT record a block, do NOT increment `sessionsCompleted`, and do NOT fire §5 tracking.** Skipping a block you didn't actually work must never write a calendar event or log task time (matches the old behavior). `reset()` re-seeds the current phase to its full duration. `skip()` advances to the next phase. **No `setState` is ever called inside another `setState` updater.**

### 4.4 Persistence

One localStorage key `nb-pomodoro-state`:

```ts
interface PersistedTimerState {
  phase: TimerMode
  endsAt: number | null          // epoch ms; null when idle/paused
  isRunning: boolean
  remainingWhenPaused: number | null
  sessionsCompleted: number
  linkedTaskId: string | null
  blockStartedAt: string | null  // ISO; start of the current/last block, for the calendar event
}
```

Settings stay in `nb-pomodoro-settings`; history stays in `nb-pomodoro-history`. On provider mount, rehydrate and recompute `remainingMs`:
- **`endsAt` in the future** → the timer simply keeps running; `remainingMs` is correct against the wall clock. This is the normal reload/return-to-page case and the primary fix for bug 1.
- **`endsAt` already in the past** → the block elapsed while the app was closed/asleep, so we **cannot confirm any focus actually happened.** Per honest-mirror principle we do **not** fabricate: skip §5 tracking entirely, do **not** increment `sessionsCompleted`, do **not** auto-start. Set the timer to **idle at the same phase**, re-seeded to full duration, and show a gentle, non-actionable note ("Your timer was interrupted while you were away"). The live in-tab completion effect (§4.3) is the only path that records a block; a block must have genuinely run to zero *with the app open* to be tracked.

## 5. Tracking on completion (work blocks)

On a completed **work** block:

1. `createEvent({ title: linkedTask ? \`Focus: ${linkedTask.title}\` : 'Focus', starts_at: blockStartedAt, ends_at: endedAt, task_id: linkedTaskId ?? undefined, color: '#ef4444', is_work_event: true, tags: ['focus'] })` → keep `event.id`.
2. If `linkedTaskId`: `logTaskTime(linkedTaskId, workMinutes)`.
3. Set `lastCompletion = { eventId, taskId, minutes }` → render the Undo toast.

**Undo** (`undoLastCompletion`): `deleteEvent(eventId)` and, if `taskId`, `logTaskTime(taskId, -minutes)`; clear `lastCompletion`.

**Defaults (assumptions, confirmed in brainstorm):**
- A work block with **no linked task** still creates a generic `Focus` calendar event (keeps the calendar truthful) but logs no task time.
- **Calendar event span = the real session window: `starts_at = blockStartedAt`, `ends_at = endsAt`** (the scheduled end). Because resume pushes `endsAt = now + remainingWhenPaused`, this window naturally includes any pause time — it answers "when were you in this session?" If the block was never paused (the common case) the span is exactly the configured duration.
- **Logged task minutes = the configured work duration** (e.g. 25), i.e. focused time, *not* the wall-clock span. This is an intentional, documented distinction: the calendar event records *when* the session happened (including pauses); `actual_minutes` records *how much focus* it represents. A paused 25-min block can therefore show a 32-min calendar event but +25 task minutes — by design.
- **Break** completion writes nothing; it shows a softer "Break over — back to focus?" toast.

## 6. Backend change

### 6.1 Migration `000009_add_task_actual_minutes`

```sql
-- up
ALTER TABLE tasks ADD COLUMN actual_minutes INTEGER NOT NULL DEFAULT 0;
-- down
ALTER TABLE tasks DROP COLUMN actual_minutes;
```

### 6.2 Endpoint `POST /api/tasks/:id/log-time`

- Body: `{ "minutes": int }` (may be negative for undo).
- Effect (atomic, parameterized): `UPDATE tasks SET actual_minutes = GREATEST(0, actual_minutes + $1), updated_at = now() WHERE id = $2 AND user_id = $3 RETURNING …`
- Returns the updated task. Auth via existing JWT middleware; ownership enforced by `user_id`.

### 6.3 Types / wiring

- `internal/tasks/types.go`: add `ActualMinutes int \`json:"actual_minutes"\`` to `Task`; add `LogTimeRequest { Minutes int \`json:"minutes"\` }`.
- `internal/tasks/handlers.go`: add `LogTimeHandler`; include `actual_minutes` in every SELECT/scan.
- `cmd/api/main.go`: register `r.Post("/tasks/{id}/log-time", tasks.LogTimeHandler)` alongside the other task routes.
- Frontend `web/src/api/tasks.ts`: add `actual_minutes: number` to `Task`; `export async function logTaskTime(id: string, minutes: number): Promise<Task>`.

## 7. Error handling (no silent failures)

- Tracking API calls are wrapped; **the timer always advances regardless** (local state is truth). On failure the toast is honest: *"Timer kept going — couldn't save to calendar. Retry?"* with a retry action; the local history block is still recorded. We never show the success toast when a write failed.
- Stale block resolved on mount (§4.4) writes **nothing** — no event, no task time, no Undo toast — because focus during the closed period cannot be confirmed. It only re-seeds the phase to idle with a gentle note.
- `logTaskTime` clamps at 0 server-side so a double-undo can't push actual time negative.

## 8. Floating widget

- `components/Pomodoro/PomodoroWidget.tsx` rendered once in `AppLayout`.
- Visible only when a timer is active (`isRunning` or paused mid-block) **and** the user is not already on `/tools/pomodoro`.
- Reads `usePomodoro()`; three pure-presentation variants chosen by `settings.widgetStyle`:
  - **pill** — corner pill: phase dot, time, pause; click to expand / go to page.
  - **card** (default) — corner card: phase, cycle pips, linked task, pause + skip. (Auto-collapse-to-pill after inactivity is optional polish, not required for this release.)
  - **bar** — slim centered top bar with live progress track, phase, time, pause.
- Mode color: work red `#ef4444`, short-break green `#22c55e`, long-break blue `#3b82f6` (existing palette).

## 9. Settings additions

`PomodoroSettings` gains `widgetStyle: 'pill' | 'card' | 'bar'` (default `'card'`), surfaced as a 3-way control in the existing settings panel. All current fields (durations, `autoStartBreaks`, `soundEnabled`, `sessionsBeforeLong`) are retained.

## 10. Files

**Frontend — new**
- `web/src/contexts/PomodoroContext.tsx` — provider + `usePomodoro()`
- `web/src/lib/pomodoro/machine.ts` — pure phase-transition + seed helpers (unit-testable)
- `web/src/lib/pomodoro/storage.ts` — persisted-state load/save
- `web/src/lib/pomodoro/tracking.ts` — completion → event + time-log + undo
- `web/src/lib/pomodoro/types.ts` — shared types
- `web/src/components/Pomodoro/PomodoroWidget.tsx` (+ `PillWidget`, `CardWidget`, `BarWidget` subcomponents)

**Frontend — modified**
- `web/src/pages/Tools/Pomodoro.tsx` — becomes a consumer of the context (full-page UI, cycle pips, completion toast)
- `web/src/api/tasks.ts` — `actual_minutes` field + `logTaskTime()`
- `web/src/App.tsx` / `router.tsx` — mount `PomodoroProvider` + `PomodoroWidget` in `AppLayout`
- settings UI — add `widgetStyle` control

**Backend — new**
- `api-go/migrations/000009_add_task_actual_minutes.{up,down}.sql`

**Backend — modified**
- `api-go/internal/tasks/types.go`, `api-go/internal/tasks/handlers.go`, `api-go/cmd/api/main.go`

## 11. Testing

- **Machine unit tests** (`machine.ts`): work→short→work→short→…→long across a full set; `sessionsCompleted` increments correctly; `skip` and `reset` seed the right next phase/duration.
- **Persistence round-trip** (`storage.ts`): save → load reproduces state; running block with future `endsAt` yields correct `remainingMs`.
- **Background/elapsed simulation**: with a fake clock, advancing past `endsAt` fires `completePhase` exactly once (no double-fire); a past `endsAt` on mount resolves the block.
- **Undo**: completion then undo issues `deleteEvent` and a compensating `logTaskTime(-minutes)`; double-undo cannot drive `actual_minutes` below 0.
- **Backend**: `log-time` increments/decrements with clamping, enforces ownership, rejects non-owned/unknown ids.
- **Manual**: start a work block, navigate to Calendar/Tasks, confirm widget shows correct time; reload mid-block; let a block complete and verify the calendar event + task `actual_minutes` + undo.

## 12. Out of scope / follow-ups
- Service-worker notifications for exact background firing.
- Per-session backend history table / cross-device sync.
- Estimate-vs-actual analytics surface on the task/Tasks page (the data will exist after this; visualizing it is later).
- **Multi-tab:** two open tabs each run a provider; both could fire `completePhase` on the same block and double-write the event/time. Acknowledged and punted for this release (single-tab is the assumed usage). A `BroadcastChannel`/storage-event lock is the future fix.
