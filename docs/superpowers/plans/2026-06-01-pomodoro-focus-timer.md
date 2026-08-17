<!-- паспорт: тип=план | статус=архив | строк=2220 | ~токенов=18354 | обновлён=по git -->

# Pomodoro / Focus Timer Rebuild — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the Pomodoro tool as a global, timestamp-based focus timer that survives navigation/reload, advances phases deterministically, and records each completed work block as a calendar event + actual minutes on the linked task (with undo), plus a 3-style floating widget visible on every page.

**Architecture:** A new global `PomodoroContext` (React Context, mirroring `AuthContext`) owns the timer. The source of truth is `endsAt` (epoch ms); `remainingMs = endsAt − now` is recomputed each tick, making it immune to reload/navigation/background-tab throttling. Phase transitions fire from a `useEffect` (never inside a `setState` updater). Completion writes a calendar event via the existing events API and logs minutes via a new `POST /api/tasks/:id/log-time` endpoint (backend migration adds `task.actual_minutes`). A floating widget rendered in `AppLayout` reflects the same context in one of three user-selectable styles.

**Tech Stack:** Go 1.22 + chi + pgx (backend), React 18 + TypeScript + Tailwind (frontend), PostgreSQL 16 + golang-migrate, Vitest + jsdom (new — frontend unit tests), i18next (en/ru).

**Spec:** `docs/superpowers/specs/2026-06-01-pomodoro-focus-timer-design.md`

**Conventions for this plan:**
- DB tables are singular: `task`, `event`.
- Commit messages follow conventional-commits and **never** co-author Claude/Anthropic.
- Frontend: run `cd web && pnpm typecheck` after TS changes; `pnpm build` before declaring a UI task done. Backend: `cd api-go && go build ./cmd/api`.
- Vitest single-file run: `cd web && pnpm exec vitest run <path>`.

---

## File Structure

**Backend**
- Create `api-go/migrations/000009_add_task_actual_minutes.up.sql` / `.down.sql` — adds `task.actual_minutes`.
- Modify `api-go/internal/tasks/types.go` — `ActualMinutes` field + `LogTimeRequest`.
- Modify `api-go/internal/tasks/handlers.go` — include `actual_minutes` in all task scans; add `logTaskTime` + `LogTimeHandler`.
- Modify `api-go/cmd/api/main.go` — register the route.
- Create `api-go/internal/tasks/handlers_test.go` — integration test for log-time (skips without `DATABASE_URL`).

**Frontend — test tooling**
- Modify `web/package.json` — add `vitest`, `jsdom`, `test` script.
- Modify `web/vite.config.ts` — add Vitest `test` block (jsdom).

**Frontend — pure logic (unit-tested)**
- Create `web/src/lib/pomodoro/types.ts` — shared types + defaults.
- Create `web/src/lib/pomodoro/machine.ts` (+ `machine.test.ts`) — phase transitions, durations.
- Create `web/src/lib/pomodoro/storage.ts` (+ `storage.test.ts`) — persistence + remaining/stale computation.
- Create `web/src/lib/pomodoro/tracking.ts` (+ `tracking.test.ts`) — focus-event builder + completion/undo orchestration.
- Create `web/src/lib/pomodoro/notify.ts` — beep + Notification helpers (browser side-effects).
- Modify `web/src/api/tasks.ts` — `actual_minutes` field + `logTaskTime()`.

**Frontend — wiring & UI**
- Create `web/src/contexts/PomodoroContext.tsx` — provider + `usePomodoro()`.
- Modify `web/src/App.tsx` — mount `PomodoroProvider`.
- Modify `web/src/router.tsx` — render `<PomodoroWidget/>` in `AppLayout`.
- Create `web/src/components/Pomodoro/PomodoroWidget.tsx` + `PillWidget.tsx` + `CardWidget.tsx` + `BarWidget.tsx`.
- Rewrite `web/src/pages/Tools/Pomodoro.tsx` — consumes context (pips, toast, interrupted note, widgetStyle setting).
- Modify `web/src/i18n/locales/en/tools.json` + `web/src/i18n/locales/ru/tools.json` — new keys.
- Modify `docs/ROADMAP.md` — mark focus-timer slice.

---

## Phase 1 — Backend

### Task 1: Migration — add `task.actual_minutes`

**Files:**
- Create: `api-go/migrations/000009_add_task_actual_minutes.up.sql`
- Create: `api-go/migrations/000009_add_task_actual_minutes.down.sql`

- [ ] **Step 1: Write the up migration**

`api-go/migrations/000009_add_task_actual_minutes.up.sql`:
```sql
-- Actual focused minutes logged against a task (e.g. by the Pomodoro timer).
-- Complements estimated_minutes for estimate-vs-reality comparison.
ALTER TABLE task ADD COLUMN IF NOT EXISTS actual_minutes INTEGER NOT NULL DEFAULT 0;
```

- [ ] **Step 2: Write the down migration**

`api-go/migrations/000009_add_task_actual_minutes.down.sql`:
```sql
ALTER TABLE task DROP COLUMN IF EXISTS actual_minutes;
```

- [ ] **Step 3: Apply locally if a DB is available (otherwise CI applies it)**

Run (only if you have a local DB):
```bash
cd api-go && migrate -path migrations -database "$DATABASE_URL" up
```
Expected: `9/u add_task_actual_minutes` applied, no error. If no local DB, skip — CI's "Run migrations" step applies it before tests.

- [ ] **Step 4: Commit**

```bash
git add api-go/migrations/000009_add_task_actual_minutes.up.sql api-go/migrations/000009_add_task_actual_minutes.down.sql
git commit -m "feat(db): add task.actual_minutes column (migration 000009)"
```

---

### Task 2: Task model — `ActualMinutes` field + `LogTimeRequest`

**Files:**
- Modify: `api-go/internal/tasks/types.go`
- Modify: `api-go/internal/tasks/handlers.go` (all 4 task SELECT/scan sites)

- [ ] **Step 1: Add the field to the `Task` struct**

In `api-go/internal/tasks/types.go`, add `ActualMinutes` immediately after `EstimatedMinutes` in the `Task` struct:
```go
	EstimatedMinutes *int          `json:"estimated_minutes,omitempty"`
	ActualMinutes    int           `json:"actual_minutes"`
```

- [ ] **Step 2: Add the `LogTimeRequest` type**

Append to `api-go/internal/tasks/types.go` (after `ScheduleTaskRequest`):
```go
// LogTimeRequest adds (or, with a negative value, removes) focused minutes
// to a task's actual_minutes total. Used by the Pomodoro timer + undo.
type LogTimeRequest struct {
	Minutes int `json:"minutes"`
}
```

- [ ] **Step 3: Include `actual_minutes` in all four task queries + scans**

In `api-go/internal/tasks/handlers.go`, for each of `listTasks`, `getTask`, `createTask`, `updateTask`: add `actual_minutes` to the SELECT/RETURNING column list (right after `updated_at`) and `&t.ActualMinutes` to the matching `rows.Scan`/`.Scan(...)` (right after `&t.UpdatedAt`).

`listTasks` — SELECT (line ~277) and Scan (line ~317):
```go
		       energy, parent_id, completed_at, created_at, updated_at, actual_minutes
```
```go
			&t.Energy, &t.ParentID, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &t.ActualMinutes,
```

`getTask` — SELECT (line ~366) and Scan (line ~372): same two additions (`, actual_minutes` in SELECT, `, &t.ActualMinutes` after `&t.UpdatedAt`).

`createTask` — RETURNING (line ~343) and Scan (line ~347): same two additions.

`updateTask` — RETURNING (line ~474) and Scan (line ~483): same two additions.

- [ ] **Step 4: Build to verify**

Run:
```bash
cd api-go && go build ./cmd/api
```
Expected: builds with no errors.

- [ ] **Step 5: Commit**

```bash
git add api-go/internal/tasks/types.go api-go/internal/tasks/handlers.go
git commit -m "feat(tasks): expose actual_minutes on Task and add LogTimeRequest"
```

---

### Task 3: `logTaskTime` DB function + `LogTimeHandler` + route

**Files:**
- Modify: `api-go/internal/tasks/handlers.go`
- Modify: `api-go/cmd/api/main.go:120`

- [ ] **Step 1: Add the DB function**

Append to `api-go/internal/tasks/handlers.go` (after `scheduleTask`):
```go
// logTaskTime adds delta minutes to a task's actual_minutes, clamped at >= 0,
// and returns the updated task. A negative delta is used for undo.
func logTaskTime(ctx context.Context, userID, taskID string, delta int) (*Task, error) {
	var t Task
	var tags, contexts []string

	err := db.Pool.QueryRow(ctx, `
		UPDATE task
		SET actual_minutes = GREATEST(0, actual_minutes + $1), updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, title, description, status, category, priority,
		          estimated_minutes, due_date, COALESCE(tags, '{}'), COALESCE(contexts, '{}'),
		          energy, parent_id, completed_at, created_at, updated_at, actual_minutes
	`, delta, taskID, userID).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Category,
		&t.Priority, &t.EstimatedMinutes, &t.DueDate, &tags, &contexts,
		&t.Energy, &t.ParentID, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &t.ActualMinutes,
	)
	if err != nil {
		return nil, err
	}
	t.Tags = tags
	t.Contexts = contexts
	return &t, nil
}
```

- [ ] **Step 2: Add the handler**

Append to `api-go/internal/tasks/handlers.go` (after `ScheduleHandler`, before `// Database operations`):
```go
// LogTimeHandler adds (or removes, if negative) focused minutes on a task.
func LogTimeHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Task ID is required")
		return
	}

	var req LogTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	task, err := logTaskTime(r.Context(), userID, taskID, req.Minutes)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Task not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "LOG_TIME_ERROR", "Failed to log time")
		return
	}

	util.RespondJSON(w, http.StatusOK, task)
}
```

- [ ] **Step 3: Register the route**

In `api-go/cmd/api/main.go`, in the Tasks group (after line 120, the `schedule` route):
```go
		r.Post("/api/tasks/{id}/schedule", t.ScheduleHandler)
		r.Post("/api/tasks/{id}/log-time", t.LogTimeHandler)
```

- [ ] **Step 4: Build to verify**

Run:
```bash
cd api-go && go build ./cmd/api
```
Expected: builds with no errors.

- [ ] **Step 5: Commit**

```bash
git add api-go/internal/tasks/handlers.go api-go/cmd/api/main.go
git commit -m "feat(tasks): add POST /api/tasks/:id/log-time endpoint"
```

---

### Task 4: Go integration test for `log-time`

**Files:**
- Create: `api-go/internal/tasks/handlers_test.go`

This test connects to the `DATABASE_URL` Postgres (the one CI already provisions and migrates) and **skips** when `DATABASE_URL` is unset, so it's safe to run locally without a DB.

- [ ] **Step 1: Write the failing test**

`api-go/internal/tasks/handlers_test.go`:
```go
package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
)

// setupTestDB connects to DATABASE_URL and seeds a user + task, returning the
// task ID, user ID, and a cleanup func. Skips the test if DATABASE_URL is unset.
// Seeds only baseline-guaranteed columns (id/email on "user", user_id/title on
// task) and relies on column defaults for everything else, so it survives
// regardless of which later migrations have run.
func setupTestDB(t *testing.T) (taskID, userID string, cleanup func()) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	InitDB(d)
	ctx := context.Background()

	// Unique email avoids UNIQUE collisions if a prior run's cleanup was skipped.
	email := fmt.Sprintf("pomodoro-test-%d@example.com", time.Now().UnixNano())
	err = d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`,
		email,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	err = d.Pool.QueryRow(ctx,
		`INSERT INTO task (user_id, title) VALUES ($1, $2) RETURNING id`,
		userID, "Test task",
	).Scan(&taskID)
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	cleanup = func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE user_id = $1`, userID)
		_, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID)
		d.Close()
	}
	return taskID, userID, cleanup
}

// callLogTime issues a log-time request authenticated as userID.
func callLogTime(taskID, userID string, minutes int) *httptest.ResponseRecorder {
	body, _ := json.Marshal(LogTimeRequest{Minutes: minutes})
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/log-time", bytes.NewReader(body))
	// Inject chi URL param + authenticated user into context.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", taskID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	LogTimeHandler(rec, req)
	return rec
}

func TestLogTimeHandler_AddsAndClampsMinutes(t *testing.T) {
	taskID, userID, cleanup := setupTestDB(t)
	defer cleanup()

	// Add 25 minutes.
	rec := callLogTime(taskID, userID, 25)
	if rec.Code != http.StatusOK {
		t.Fatalf("add: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data Task `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.ActualMinutes != 25 {
		t.Fatalf("expected 25 actual_minutes, got %d", resp.Data.ActualMinutes)
	}

	// Undo (-25) should bring it to 0.
	rec = callLogTime(taskID, userID, -25)
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.ActualMinutes != 0 {
		t.Fatalf("expected 0 after undo, got %d", resp.Data.ActualMinutes)
	}

	// Over-subtract clamps at 0, never negative.
	rec = callLogTime(taskID, userID, -100)
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.ActualMinutes != 0 {
		t.Fatalf("expected clamp at 0, got %d", resp.Data.ActualMinutes)
	}
}

func TestLogTimeHandler_NotOwnedReturns404(t *testing.T) {
	taskID, _, cleanup := setupTestDB(t)
	defer cleanup()

	rec := callLogTime(taskID, "00000000-0000-0000-0000-000000000000", 25)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner, got %d", rec.Code)
	}
}
```

> Note: `util.RespondJSON` wraps payloads as `{"data": ...}` (verified in `internal/util/response.go`), so the test decodes `resp.Data`.

- [ ] **Step 2: Run the test to verify it fails (or skips locally)**

Run:
```bash
cd api-go && go test -v ./internal/tasks/
```
Expected: with no `DATABASE_URL`, both tests print `SKIP`. With a DB, they currently **PASS** (Task 3 already implemented the handler) — this test guards against regressions. If Task 3 were missing, it would FAIL to compile/return 404.

- [ ] **Step 3: Commit**

```bash
git add api-go/internal/tasks/handlers_test.go
git commit -m "test(tasks): integration test for log-time add/clamp/ownership"
```

---

## Phase 2 — Frontend test tooling

### Task 5: Add Vitest + jsdom

**Files:**
- Modify: `web/package.json`
- Modify: `web/vite.config.ts`
- Create: `web/src/lib/pomodoro/smoke.test.ts` (temporary sanity test, deleted at end of task)

- [ ] **Step 1: Add dev-dependencies + test script**

In `web/package.json`, add to `scripts`:
```json
    "typecheck": "tsc --noEmit",
    "test": "vitest run",
    "test:watch": "vitest"
```
Add to `devDependencies`:
```json
    "jsdom": "^25.0.0",
    "vitest": "^2.1.0"
```

- [ ] **Step 2: Install**

Run:
```bash
cd web && pnpm install
```
Expected: lockfile updates; `vitest` and `jsdom` installed.

- [ ] **Step 3: Configure Vitest (jsdom env) in vite.config.ts**

Replace `web/vite.config.ts` with:
```ts
/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: { port: 5173 },
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
```

- [ ] **Step 4: Write a smoke test**

`web/src/lib/pomodoro/smoke.test.ts`:
```ts
import { describe, it, expect } from 'vitest'

describe('vitest smoke', () => {
  it('runs and sees jsdom localStorage', () => {
    localStorage.setItem('k', 'v')
    expect(localStorage.getItem('k')).toBe('v')
  })
})
```

- [ ] **Step 5: Run it**

Run:
```bash
cd web && pnpm exec vitest run src/lib/pomodoro/smoke.test.ts
```
Expected: 1 passed.

- [ ] **Step 6: Delete the smoke test, commit the setup**

```bash
rm web/src/lib/pomodoro/smoke.test.ts
git add web/package.json web/pnpm-lock.yaml web/vite.config.ts
git commit -m "build(web): add Vitest + jsdom for unit tests"
```

---

## Phase 3 — Frontend pure logic (TDD)

### Task 6: Shared types

**Files:**
- Create: `web/src/lib/pomodoro/types.ts`

- [ ] **Step 1: Write the types + defaults**

`web/src/lib/pomodoro/types.ts`:
```ts
export type TimerMode = 'work' | 'shortBreak' | 'longBreak'

export type WidgetStyle = 'pill' | 'card' | 'bar'

export interface PomodoroSettings {
  workMinutes: number
  shortBreakMinutes: number
  longBreakMinutes: number
  autoStartBreaks: boolean
  soundEnabled: boolean
  sessionsBeforeLong: number
  widgetStyle: WidgetStyle
}

export const DEFAULT_SETTINGS: PomodoroSettings = {
  workMinutes: 25,
  shortBreakMinutes: 5,
  longBreakMinutes: 15,
  autoStartBreaks: true,
  soundEnabled: true,
  sessionsBeforeLong: 4,
  widgetStyle: 'card',
}

export interface PersistedTimerState {
  phase: TimerMode
  endsAt: number | null // epoch ms; null when idle/paused
  isRunning: boolean
  remainingWhenPaused: number | null // ms
  sessionsCompleted: number
  linkedTaskId: string | null
  linkedTaskTitle: string | null
  blockStartedAt: string | null // ISO; start of the current work block
}

export interface SessionRecord {
  date: string // YYYY-MM-DD
  mode: TimerMode
  durationSeconds: number
  taskId?: string
  completedAt: string // ISO
}

export interface Completion {
  eventId: string | null
  taskId: string | null
  minutes: number
  failed: boolean
}
```

- [ ] **Step 2: Typecheck**

Run:
```bash
cd web && pnpm typecheck
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/pomodoro/types.ts
git commit -m "feat(pomodoro): shared timer types and default settings"
```

---

### Task 7: Phase machine (TDD)

**Files:**
- Create: `web/src/lib/pomodoro/machine.test.ts`
- Create: `web/src/lib/pomodoro/machine.ts`

- [ ] **Step 1: Write the failing tests**

`web/src/lib/pomodoro/machine.test.ts`:
```ts
import { describe, it, expect } from 'vitest'
import { nextPhase, minutesForMode, durationMsForMode } from './machine'
import { DEFAULT_SETTINGS } from './types'

describe('nextPhase', () => {
  it('work → shortBreak when the completed count is not a multiple of sessionsBeforeLong', () => {
    expect(nextPhase('work', 1, 4)).toBe('shortBreak')
    expect(nextPhase('work', 2, 4)).toBe('shortBreak')
    expect(nextPhase('work', 3, 4)).toBe('shortBreak')
  })

  it('work → longBreak when the completed count hits a multiple of sessionsBeforeLong', () => {
    expect(nextPhase('work', 4, 4)).toBe('longBreak')
    expect(nextPhase('work', 8, 4)).toBe('longBreak')
  })

  it('any break → work', () => {
    expect(nextPhase('shortBreak', 2, 4)).toBe('work')
    expect(nextPhase('longBreak', 4, 4)).toBe('work')
  })

  it('walks a full set: 4th work block tips into the long break', () => {
    const before = 4
    const seq: string[] = []
    let completed = 0
    for (let i = 0; i < 4; i++) {
      completed += 1
      seq.push(nextPhase('work', completed, before))
    }
    expect(seq).toEqual(['shortBreak', 'shortBreak', 'shortBreak', 'longBreak'])
  })
})

describe('durations', () => {
  it('minutesForMode reads the matching setting', () => {
    expect(minutesForMode('work', DEFAULT_SETTINGS)).toBe(25)
    expect(minutesForMode('shortBreak', DEFAULT_SETTINGS)).toBe(5)
    expect(minutesForMode('longBreak', DEFAULT_SETTINGS)).toBe(15)
  })

  it('durationMsForMode converts minutes to ms', () => {
    expect(durationMsForMode('work', DEFAULT_SETTINGS)).toBe(25 * 60 * 1000)
  })
})
```

- [ ] **Step 2: Run to verify it fails**

Run:
```bash
cd web && pnpm exec vitest run src/lib/pomodoro/machine.test.ts
```
Expected: FAIL — `Failed to resolve import "./machine"`.

- [ ] **Step 3: Implement the machine**

`web/src/lib/pomodoro/machine.ts`:
```ts
import type { TimerMode, PomodoroSettings } from './types'

/**
 * Given the phase that just finished, returns the next phase.
 * For work, pass the sessionsCompleted count AFTER incrementing it.
 */
export function nextPhase(
  finishedPhase: TimerMode,
  sessionsCompletedAfter: number,
  sessionsBeforeLong: number
): TimerMode {
  if (finishedPhase === 'work') {
    return sessionsCompletedAfter % sessionsBeforeLong === 0 ? 'longBreak' : 'shortBreak'
  }
  return 'work'
}

export function minutesForMode(mode: TimerMode, settings: PomodoroSettings): number {
  switch (mode) {
    case 'work':
      return settings.workMinutes
    case 'shortBreak':
      return settings.shortBreakMinutes
    case 'longBreak':
      return settings.longBreakMinutes
  }
}

export function durationMsForMode(mode: TimerMode, settings: PomodoroSettings): number {
  return minutesForMode(mode, settings) * 60 * 1000
}
```

- [ ] **Step 4: Run to verify it passes**

Run:
```bash
cd web && pnpm exec vitest run src/lib/pomodoro/machine.test.ts
```
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/pomodoro/machine.ts web/src/lib/pomodoro/machine.test.ts
git commit -m "feat(pomodoro): pure phase-transition machine with tests"
```

---

### Task 8: Persistence + remaining/stale (TDD)

**Files:**
- Create: `web/src/lib/pomodoro/storage.test.ts`
- Create: `web/src/lib/pomodoro/storage.ts`

- [ ] **Step 1: Write the failing tests**

`web/src/lib/pomodoro/storage.test.ts`:
```ts
import { describe, it, expect, beforeEach } from 'vitest'
import {
  loadSettings,
  saveSettings,
  loadTimerState,
  saveTimerState,
  clearTimerState,
  computeRemainingMs,
  isStale,
} from './storage'
import { DEFAULT_SETTINGS, type PersistedTimerState } from './types'

const runningState = (endsAt: number): PersistedTimerState => ({
  phase: 'work',
  endsAt,
  isRunning: true,
  remainingWhenPaused: null,
  sessionsCompleted: 1,
  linkedTaskId: 't1',
  linkedTaskTitle: 'Task one',
  blockStartedAt: '2026-06-01T10:00:00.000Z',
})

describe('settings persistence', () => {
  beforeEach(() => localStorage.clear())

  it('returns defaults when nothing stored', () => {
    expect(loadSettings()).toEqual(DEFAULT_SETTINGS)
  })

  it('round-trips and merges unknown-missing keys onto defaults', () => {
    saveSettings({ ...DEFAULT_SETTINGS, workMinutes: 50, widgetStyle: 'pill' })
    const loaded = loadSettings()
    expect(loaded.workMinutes).toBe(50)
    expect(loaded.widgetStyle).toBe('pill')
    expect(loaded.shortBreakMinutes).toBe(DEFAULT_SETTINGS.shortBreakMinutes)
  })
})

describe('timer state persistence', () => {
  beforeEach(() => localStorage.clear())

  it('round-trips state', () => {
    const s = runningState(1000)
    saveTimerState(s)
    expect(loadTimerState()).toEqual(s)
  })

  it('returns null when nothing stored, and after clear', () => {
    expect(loadTimerState()).toBeNull()
    saveTimerState(runningState(1000))
    clearTimerState()
    expect(loadTimerState()).toBeNull()
  })
})

describe('computeRemainingMs', () => {
  it('running: endsAt - now, floored at 0', () => {
    expect(computeRemainingMs(runningState(5000), 2000)).toBe(3000)
    expect(computeRemainingMs(runningState(5000), 9000)).toBe(0)
  })

  it('paused: returns remainingWhenPaused', () => {
    const paused: PersistedTimerState = {
      ...runningState(0),
      isRunning: false,
      endsAt: null,
      remainingWhenPaused: 4200,
    }
    expect(computeRemainingMs(paused, 999999)).toBe(4200)
  })
})

describe('isStale', () => {
  it('true when running and endsAt already passed', () => {
    expect(isStale(runningState(1000), 2000)).toBe(true)
  })
  it('false when running and endsAt still in the future', () => {
    expect(isStale(runningState(5000), 2000)).toBe(false)
  })
  it('false when not running', () => {
    const paused = { ...runningState(1000), isRunning: false, endsAt: null }
    expect(isStale(paused, 999999)).toBe(false)
  })
})
```

- [ ] **Step 2: Run to verify it fails**

Run:
```bash
cd web && pnpm exec vitest run src/lib/pomodoro/storage.test.ts
```
Expected: FAIL — cannot resolve `./storage`.

- [ ] **Step 3: Implement storage**

`web/src/lib/pomodoro/storage.ts`:
```ts
import { DEFAULT_SETTINGS, type PomodoroSettings, type PersistedTimerState } from './types'

const SETTINGS_KEY = 'nb-pomodoro-settings'
const STATE_KEY = 'nb-pomodoro-state'

export function loadSettings(): PomodoroSettings {
  try {
    const raw = localStorage.getItem(SETTINGS_KEY)
    if (!raw) return DEFAULT_SETTINGS
    return { ...DEFAULT_SETTINGS, ...(JSON.parse(raw) as Partial<PomodoroSettings>) }
  } catch {
    return DEFAULT_SETTINGS
  }
}

export function saveSettings(s: PomodoroSettings): void {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(s))
}

export function loadTimerState(): PersistedTimerState | null {
  try {
    const raw = localStorage.getItem(STATE_KEY)
    if (!raw) return null
    return JSON.parse(raw) as PersistedTimerState
  } catch {
    return null
  }
}

export function saveTimerState(s: PersistedTimerState): void {
  localStorage.setItem(STATE_KEY, JSON.stringify(s))
}

export function clearTimerState(): void {
  localStorage.removeItem(STATE_KEY)
}

/** Remaining ms: derived from the wall clock when running, else the paused value. */
export function computeRemainingMs(state: PersistedTimerState, now: number): number {
  if (!state.isRunning) return state.remainingWhenPaused ?? 0
  if (state.endsAt == null) return 0
  return Math.max(0, state.endsAt - now)
}

/** A running block whose endsAt is already in the past — elapsed while the app was closed. */
export function isStale(state: PersistedTimerState, now: number): boolean {
  return state.isRunning && state.endsAt != null && state.endsAt <= now
}
```

- [ ] **Step 4: Run to verify it passes**

Run:
```bash
cd web && pnpm exec vitest run src/lib/pomodoro/storage.test.ts
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/pomodoro/storage.ts web/src/lib/pomodoro/storage.test.ts
git commit -m "feat(pomodoro): timer persistence + remaining/stale computation with tests"
```

---

### Task 9: API client — `logTaskTime` + `actual_minutes`

**Files:**
- Modify: `web/src/api/tasks.ts`

- [ ] **Step 1: Add `actual_minutes` to the `Task` interface**

In `web/src/api/tasks.ts`, add to the `Task` interface after `estimated_minutes`:
```ts
  estimated_minutes?: number
  actual_minutes: number
```

- [ ] **Step 2: Add the `logTaskTime` function**

In `web/src/api/tasks.ts`, after `scheduleTask`:
```ts
export async function logTaskTime(id: string, minutes: number): Promise<Task> {
  return api.post<Task>(`/tasks/${id}/log-time`, { minutes })
}
```

- [ ] **Step 3: Typecheck**

Run:
```bash
cd web && pnpm typecheck
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add web/src/api/tasks.ts
git commit -m "feat(api): logTaskTime client + actual_minutes on Task"
```

---

### Task 10: Tracking — focus-event builder + completion/undo (TDD)

**Files:**
- Create: `web/src/lib/pomodoro/tracking.test.ts`
- Create: `web/src/lib/pomodoro/tracking.ts`

- [ ] **Step 1: Write the failing tests**

`web/src/lib/pomodoro/tracking.test.ts`:
```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../api/events', () => ({
  createEvent: vi.fn(),
  deleteEvent: vi.fn(),
}))
vi.mock('../../api/tasks', () => ({
  logTaskTime: vi.fn(),
}))

import { createEvent, deleteEvent } from '../../api/events'
import { logTaskTime } from '../../api/tasks'
import { buildFocusEvent, recordWorkCompletion, undoWorkCompletion } from './tracking'

const baseParams = {
  startedAtISO: '2026-06-01T10:00:00.000Z',
  endsAtISO: '2026-06-01T10:25:00.000Z',
  minutes: 25,
}

describe('buildFocusEvent', () => {
  it('titles with the task and links the task id', () => {
    const ev = buildFocusEvent({ ...baseParams, taskId: 't1', taskTitle: 'Write brief' })
    expect(ev.title).toBe('Focus: Write brief')
    expect(ev.task_id).toBe('t1')
    expect(ev.starts_at).toBe(baseParams.startedAtISO)
    expect(ev.ends_at).toBe(baseParams.endsAtISO)
    expect(ev.is_work_event).toBe(true)
    expect(ev.tags).toEqual(['focus'])
    expect(ev.color).toBe('#ef4444')
  })

  it('falls back to a generic title and no task id when unlinked', () => {
    const ev = buildFocusEvent({ ...baseParams, taskId: null, taskTitle: null })
    expect(ev.title).toBe('Focus')
    expect(ev.task_id).toBeUndefined()
  })
})

describe('recordWorkCompletion', () => {
  beforeEach(() => vi.clearAllMocks())

  it('creates an event and logs task time when a task is linked', async () => {
    vi.mocked(createEvent).mockResolvedValue({ id: 'ev1' } as never)
    vi.mocked(logTaskTime).mockResolvedValue({} as never)
    const res = await recordWorkCompletion({ ...baseParams, taskId: 't1', taskTitle: 'X' })
    expect(createEvent).toHaveBeenCalledOnce()
    expect(logTaskTime).toHaveBeenCalledWith('t1', 25)
    expect(res).toEqual({ eventId: 'ev1', taskId: 't1', minutes: 25, failed: false })
  })

  it('creates the event but does NOT log time when unlinked', async () => {
    vi.mocked(createEvent).mockResolvedValue({ id: 'ev2' } as never)
    const res = await recordWorkCompletion({ ...baseParams, taskId: null, taskTitle: null })
    expect(logTaskTime).not.toHaveBeenCalled()
    expect(res.eventId).toBe('ev2')
    expect(res.failed).toBe(false)
  })

  it('returns failed=true (and does not throw) when the API errors', async () => {
    vi.mocked(createEvent).mockRejectedValue(new Error('offline'))
    const res = await recordWorkCompletion({ ...baseParams, taskId: 't1', taskTitle: 'X' })
    expect(res.failed).toBe(true)
    expect(res.eventId).toBeNull()
  })
})

describe('undoWorkCompletion', () => {
  beforeEach(() => vi.clearAllMocks())

  it('deletes the event and compensates the logged minutes', async () => {
    vi.mocked(deleteEvent).mockResolvedValue(undefined as never)
    vi.mocked(logTaskTime).mockResolvedValue({} as never)
    await undoWorkCompletion({ eventId: 'ev1', taskId: 't1', minutes: 25, failed: false })
    expect(deleteEvent).toHaveBeenCalledWith('ev1')
    expect(logTaskTime).toHaveBeenCalledWith('t1', -25)
  })

  it('skips deletion when there was no event', async () => {
    await undoWorkCompletion({ eventId: null, taskId: null, minutes: 25, failed: true })
    expect(deleteEvent).not.toHaveBeenCalled()
    expect(logTaskTime).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run to verify it fails**

Run:
```bash
cd web && pnpm exec vitest run src/lib/pomodoro/tracking.test.ts
```
Expected: FAIL — cannot resolve `./tracking`.

- [ ] **Step 3: Implement tracking**

`web/src/lib/pomodoro/tracking.ts`:
```ts
import { createEvent, deleteEvent, type CreateEventRequest } from '../../api/events'
import { logTaskTime } from '../../api/tasks'
import type { Completion } from './types'

const WORK_COLOR = '#ef4444'

export interface FocusEventParams {
  taskId: string | null
  taskTitle: string | null
  startedAtISO: string
  endsAtISO: string
}

export function buildFocusEvent(params: FocusEventParams): CreateEventRequest {
  return {
    title: params.taskTitle ? `Focus: ${params.taskTitle}` : 'Focus',
    starts_at: params.startedAtISO,
    ends_at: params.endsAtISO,
    task_id: params.taskId ?? undefined,
    color: WORK_COLOR,
    is_work_event: true,
    tags: ['focus'],
  }
}

export interface CompletionParams extends FocusEventParams {
  minutes: number
}

/**
 * Records a completed work block: creates a calendar event and, if a task is
 * linked, logs the focused minutes. Never throws — on failure returns
 * { failed: true } so the timer can advance and the UI can show an honest toast.
 */
export async function recordWorkCompletion(params: CompletionParams): Promise<Completion> {
  try {
    const event = await createEvent(buildFocusEvent(params))
    if (params.taskId) {
      await logTaskTime(params.taskId, params.minutes)
    }
    return { eventId: event.id, taskId: params.taskId, minutes: params.minutes, failed: false }
  } catch {
    return { eventId: null, taskId: params.taskId, minutes: params.minutes, failed: true }
  }
}

/** Reverses a recorded completion: deletes the event and compensates the minutes. */
export async function undoWorkCompletion(c: Completion): Promise<void> {
  if (c.eventId) await deleteEvent(c.eventId)
  if (c.taskId) await logTaskTime(c.taskId, -c.minutes)
}
```

- [ ] **Step 4: Run to verify it passes**

Run:
```bash
cd web && pnpm exec vitest run src/lib/pomodoro/tracking.test.ts
```
Expected: all pass.

- [ ] **Step 5: Run the whole suite + typecheck**

Run:
```bash
cd web && pnpm test && pnpm typecheck
```
Expected: machine + storage + tracking suites all pass; no type errors.

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/pomodoro/tracking.ts web/src/lib/pomodoro/tracking.test.ts
git commit -m "feat(pomodoro): focus-event builder + completion/undo tracking with tests"
```

---

### Task 11: Notification + beep helpers

**Files:**
- Create: `web/src/lib/pomodoro/notify.ts`

- [ ] **Step 1: Implement the helpers (moved from the old page)**

`web/src/lib/pomodoro/notify.ts`:
```ts
export function playBeep(enabled: boolean): void {
  if (!enabled) return
  try {
    const ctx = new AudioContext()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.type = 'sine'
    osc.frequency.setValueAtTime(800, ctx.currentTime)
    gain.gain.setValueAtTime(0.4, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.4)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.4)
    osc.onended = () => ctx.close()
  } catch {
    // AudioContext unavailable — ignore
  }
}

export function requestNotificationPermission(): void {
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission()
  }
}

export function showNotification(title: string, body: string): void {
  if ('Notification' in window && Notification.permission === 'granted') {
    new Notification(title, { body, icon: '/favicon.ico' })
  }
}
```

- [ ] **Step 2: Typecheck**

Run:
```bash
cd web && pnpm typecheck
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/pomodoro/notify.ts
git commit -m "feat(pomodoro): beep + notification helpers"
```

---

## Phase 4 — Context provider

### Task 12: `PomodoroContext` + mount in App

**Files:**
- Create: `web/src/contexts/PomodoroContext.tsx`
- Modify: `web/src/App.tsx`

- [ ] **Step 1: Write the provider**

`web/src/contexts/PomodoroContext.tsx`:
```tsx
import {
  createContext,
  useContext,
  useState,
  useEffect,
  useRef,
  useCallback,
  type ReactNode,
} from 'react'
import {
  type TimerMode,
  type PomodoroSettings,
  type PersistedTimerState,
  type Completion,
  type SessionRecord,
} from '../lib/pomodoro/types'
import {
  loadSettings,
  saveSettings,
  loadTimerState,
  saveTimerState,
  isStale,
} from '../lib/pomodoro/storage'
import { nextPhase, durationMsForMode, minutesForMode } from '../lib/pomodoro/machine'
import { recordWorkCompletion, undoWorkCompletion } from '../lib/pomodoro/tracking'
import { playBeep, requestNotificationPermission, showNotification } from '../lib/pomodoro/notify'

interface PomodoroContextValue {
  phase: TimerMode
  remainingMs: number
  isRunning: boolean
  isActive: boolean
  sessionsCompleted: number
  linkedTaskId: string | null
  linkedTaskTitle: string | null
  settings: PomodoroSettings
  lastCompletion: Completion | null
  breakOver: boolean
  interruptedWhileAway: boolean
  start: () => void
  pause: () => void
  reset: () => void
  skip: () => void
  selectPhase: (m: TimerMode) => void
  setLinkedTask: (id: string | null, title: string | null) => void
  updateSettings: (patch: Partial<PomodoroSettings>) => void
  dismissCompletion: () => void
  undoLastCompletion: () => void
  dismissBreakOver: () => void
  dismissInterrupted: () => void
}

const PomodoroContext = createContext<PomodoroContextValue | null>(null)

const HISTORY_KEY = 'nb-pomodoro-history'

function appendHistory(record: SessionRecord): void {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    const list: SessionRecord[] = raw ? JSON.parse(raw) : []
    list.push(record)
    localStorage.setItem(HISTORY_KEY, JSON.stringify(list))
  } catch {
    // ignore persistence failure
  }
}

/**
 * Computes the corrected initial state ONCE, synchronously, before first render.
 * A stale persisted block (endsAt already in the past — elapsed while the app
 * was closed) is resolved to idle here so it is NEVER committed as isRunning:true.
 * This is critical: a mount EFFECT cannot do this safely, because the completion
 * effect runs in the same passive-effect flush and would still see isRunning:true
 * and fire completePhase() — fabricating a calendar event/time that decision (d)
 * of the spec forbids.
 */
function computeInitial(
  p: PersistedTimerState | null,
  s: PomodoroSettings
): PersistedTimerState & { interrupted: boolean } {
  if (!p) {
    return {
      phase: 'work',
      endsAt: null,
      isRunning: false,
      remainingWhenPaused: durationMsForMode('work', s),
      sessionsCompleted: 0,
      linkedTaskId: null,
      linkedTaskTitle: null,
      blockStartedAt: null,
      interrupted: false,
    }
  }
  if (isStale(p, Date.now())) {
    return {
      ...p,
      isRunning: false,
      endsAt: null,
      remainingWhenPaused: durationMsForMode(p.phase, s),
      blockStartedAt: null,
      interrupted: true,
    }
  }
  return { ...p, interrupted: false }
}

export function PomodoroProvider({ children }: { children: ReactNode }) {
  const initialSettings = useRef(loadSettings()).current
  const init = useRef(computeInitial(loadTimerState(), initialSettings)).current

  const [settings, setSettings] = useState<PomodoroSettings>(initialSettings)
  const [phase, setPhase] = useState<TimerMode>(init.phase)
  const [endsAt, setEndsAt] = useState<number | null>(init.endsAt)
  const [isRunning, setIsRunning] = useState<boolean>(init.isRunning)
  const [remainingWhenPaused, setRemainingWhenPaused] = useState<number | null>(init.remainingWhenPaused)
  const [sessionsCompleted, setSessionsCompleted] = useState<number>(init.sessionsCompleted)
  const [linkedTaskId, setLinkedTaskId] = useState<string | null>(init.linkedTaskId)
  const [linkedTaskTitle, setLinkedTaskTitle] = useState<string | null>(init.linkedTaskTitle)
  const [blockStartedAt, setBlockStartedAt] = useState<string | null>(init.blockStartedAt)

  const [now, setNow] = useState<number>(() => Date.now())
  const [lastCompletion, setLastCompletion] = useState<Completion | null>(null)
  const [breakOver, setBreakOver] = useState(false)
  const [interruptedWhileAway, setInterruptedWhileAway] = useState(init.interrupted)

  const handledEndsAtRef = useRef<number | null>(null)

  const remainingMs = isRunning
    ? Math.max(0, (endsAt ?? now) - now)
    : remainingWhenPaused ?? durationMsForMode(phase, settings)

  const fullMsForPhase = durationMsForMode(phase, settings)
  const isActive = isRunning || (remainingWhenPaused != null && remainingWhenPaused < fullMsForPhase)

  // (Stale-block resolution happens synchronously in computeInitial above —
  // never in an effect, to avoid the completion effect firing on a stale block.)

  // Display tick — re-render once per second while running. Display-only;
  // remainingMs is always derived from the wall clock, not this interval.
  useEffect(() => {
    if (!isRunning) return
    setNow(Date.now())
    const id = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(id)
  }, [isRunning])

  // Persist whenever any durable field changes.
  useEffect(() => {
    saveTimerState({
      phase,
      endsAt,
      isRunning,
      remainingWhenPaused,
      sessionsCompleted,
      linkedTaskId,
      linkedTaskTitle,
      blockStartedAt,
    })
  }, [phase, endsAt, isRunning, remainingWhenPaused, sessionsCompleted, linkedTaskId, linkedTaskTitle, blockStartedAt])

  const seedPhase = useCallback(
    (mode: TimerMode, autoStart: boolean) => {
      const durMs = durationMsForMode(mode, settings)
      setPhase(mode)
      if (autoStart) {
        setEndsAt(Date.now() + durMs)
        setRemainingWhenPaused(null)
        setIsRunning(true)
        setBlockStartedAt(new Date().toISOString())
      } else {
        setEndsAt(null)
        setRemainingWhenPaused(durMs)
        setIsRunning(false)
        setBlockStartedAt(null)
      }
    },
    [settings]
  )

  const completePhase = useCallback(() => {
    const finishedPhase = phase
    const finishedEndsAt = endsAt
    const endedISO = finishedEndsAt != null ? new Date(finishedEndsAt).toISOString() : new Date().toISOString()

    if (finishedPhase === 'work') {
      const minutes = minutesForMode('work', settings)
      appendHistory({
        date: endedISO.slice(0, 10),
        mode: 'work',
        durationSeconds: minutes * 60,
        taskId: linkedTaskId ?? undefined,
        completedAt: endedISO,
      })
      showNotification('Work session complete', 'Time for a break')
      const startedISO = blockStartedAt ?? new Date(Date.now() - minutes * 60 * 1000).toISOString()
      void recordWorkCompletion({
        taskId: linkedTaskId,
        taskTitle: linkedTaskTitle,
        startedAtISO: startedISO,
        endsAtISO: endedISO,
        minutes,
      }).then(setLastCompletion)

      const newCompleted = sessionsCompleted + 1
      setSessionsCompleted(newCompleted)
      seedPhase(nextPhase('work', newCompleted, settings.sessionsBeforeLong), settings.autoStartBreaks)
    } else {
      appendHistory({
        date: endedISO.slice(0, 10),
        mode: finishedPhase,
        durationSeconds: minutesForMode(finishedPhase, settings) * 60,
        completedAt: endedISO,
      })
      showNotification('Break over', 'Back to focus?')
      setBreakOver(true)
      seedPhase('work', false)
    }
  }, [phase, endsAt, settings, linkedTaskId, linkedTaskTitle, blockStartedAt, sessionsCompleted, seedPhase])

  // Completion detector — fires exactly once per block. Never inside a setState updater.
  useEffect(() => {
    if (!isRunning || endsAt == null) return
    if (now < endsAt) return
    if (handledEndsAtRef.current === endsAt) return
    handledEndsAtRef.current = endsAt
    playBeep(settings.soundEnabled)
    completePhase()
  }, [now, isRunning, endsAt, settings.soundEnabled, completePhase])

  const start = useCallback(() => {
    requestNotificationPermission()
    const durMs = remainingWhenPaused ?? durationMsForMode(phase, settings)
    setEndsAt(Date.now() + durMs)
    setRemainingWhenPaused(null)
    setIsRunning(true)
    setBlockStartedAt((prev) => prev ?? new Date().toISOString())
    setInterruptedWhileAway(false)
  }, [remainingWhenPaused, phase, settings])

  const pause = useCallback(() => {
    setRemainingWhenPaused(endsAt != null ? Math.max(0, endsAt - Date.now()) : remainingWhenPaused)
    setEndsAt(null)
    setIsRunning(false)
  }, [endsAt, remainingWhenPaused])

  const reset = useCallback(() => {
    setIsRunning(false)
    setEndsAt(null)
    setRemainingWhenPaused(durationMsForMode(phase, settings))
    setBlockStartedAt(null)
  }, [phase, settings])

  const skip = useCallback(() => {
    // Advance phase only. No tracking, no increment — you didn't complete this block.
    const np: TimerMode = phase === 'work' ? 'shortBreak' : 'work'
    seedPhase(np, false)
  }, [phase, seedPhase])

  const selectPhase = useCallback(
    (m: TimerMode) => {
      seedPhase(m, false)
    },
    [seedPhase]
  )

  const setLinkedTask = useCallback((id: string | null, title: string | null) => {
    setLinkedTaskId(id)
    setLinkedTaskTitle(title)
  }, [])

  const updateSettings = useCallback(
    (patch: Partial<PomodoroSettings>) => {
      setSettings((prev) => {
        const next = { ...prev, ...patch }
        saveSettings(next)
        return next
      })
    },
    []
  )

  const dismissCompletion = useCallback(() => setLastCompletion(null), [])
  const undoLastCompletion = useCallback(() => {
    if (lastCompletion) void undoWorkCompletion(lastCompletion)
    setLastCompletion(null)
  }, [lastCompletion])
  const dismissBreakOver = useCallback(() => setBreakOver(false), [])
  const dismissInterrupted = useCallback(() => setInterruptedWhileAway(false), [])

  const value: PomodoroContextValue = {
    phase,
    remainingMs,
    isRunning,
    isActive,
    sessionsCompleted,
    linkedTaskId,
    linkedTaskTitle,
    settings,
    lastCompletion,
    breakOver,
    interruptedWhileAway,
    start,
    pause,
    reset,
    skip,
    selectPhase,
    setLinkedTask,
    updateSettings,
    dismissCompletion,
    undoLastCompletion,
    dismissBreakOver,
    dismissInterrupted,
  }

  return <PomodoroContext.Provider value={value}>{children}</PomodoroContext.Provider>
}

export function usePomodoro(): PomodoroContextValue {
  const ctx = useContext(PomodoroContext)
  if (!ctx) throw new Error('usePomodoro must be used within PomodoroProvider')
  return ctx
}
```

- [ ] **Step 2: Mount the provider in `App.tsx`**

Replace `web/src/App.tsx` with:
```tsx
import { RouterProvider } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import { PomodoroProvider } from './contexts/PomodoroContext'
import { ErrorBoundary } from './components/ErrorBoundary'
import { router } from './router'
import { useUIScale } from './hooks/useUIScale'

function AppShell() {
  useUIScale()
  return <RouterProvider router={router} />
}

export default function App() {
  return (
    <ErrorBoundary>
      <AuthProvider>
        <PomodoroProvider>
          <AppShell />
        </PomodoroProvider>
      </AuthProvider>
    </ErrorBoundary>
  )
}
```

- [ ] **Step 3: Typecheck**

Run:
```bash
cd web && pnpm typecheck
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add web/src/contexts/PomodoroContext.tsx web/src/App.tsx
git commit -m "feat(pomodoro): global timer context (timestamp model, single-fire transitions)"
```

---

## Phase 5 — UI

### Task 13: i18n keys (en + ru)

**Files:**
- Modify: `web/src/i18n/locales/en/tools.json`
- Modify: `web/src/i18n/locales/ru/tools.json`

- [ ] **Step 1: Add the new keys to `en/tools.json`**

Insert after the existing `"pomodoro.noHistory"` line:
```json
  "pomodoro.cycle": "Cycle",
  "pomodoro.cycleProgress": "{{done}} done · long break after {{total}}",
  "pomodoro.focusBlock": "Focus · {{minutes}} min block",
  "pomodoro.widgetStyle": "Floating widget",
  "pomodoro.widget.pill": "Pill",
  "pomodoro.widget.card": "Card",
  "pomodoro.widget.bar": "Bar",
  "pomodoro.loggedToTask": "Logged {{minutes}} min to {{task}} · added to calendar",
  "pomodoro.loggedNoTask": "Added {{minutes}} min focus block to your calendar",
  "pomodoro.saveFailed": "Timer kept going — couldn't save to calendar",
  "pomodoro.undo": "Undo",
  "pomodoro.breakOver": "Break over — back to focus?",
  "pomodoro.interrupted": "Your timer was interrupted while you were away",
  "pomodoro.dismiss": "Dismiss",
  "pomodoro.todayBlocks": "Today: {{count}} focus blocks · {{hours}}h {{minutes}}m focused"
```

- [ ] **Step 2: Add the same keys (Russian) to `ru/tools.json`**

Insert after the existing `"pomodoro.noHistory"` line:
```json
  "pomodoro.cycle": "Цикл",
  "pomodoro.cycleProgress": "{{done}} выполнено · длинный перерыв после {{total}}",
  "pomodoro.focusBlock": "Фокус · блок {{minutes}} мин",
  "pomodoro.widgetStyle": "Плавающий виджет",
  "pomodoro.widget.pill": "Пилюля",
  "pomodoro.widget.card": "Карточка",
  "pomodoro.widget.bar": "Полоса",
  "pomodoro.loggedToTask": "Записано {{minutes}} мин в «{{task}}» · добавлено в календарь",
  "pomodoro.loggedNoTask": "Блок фокуса {{minutes}} мин добавлен в календарь",
  "pomodoro.saveFailed": "Таймер продолжил идти — не удалось сохранить в календарь",
  "pomodoro.undo": "Отменить",
  "pomodoro.breakOver": "Перерыв окончен — вернуться к работе?",
  "pomodoro.interrupted": "Таймер был прерван, пока вас не было",
  "pomodoro.dismiss": "Скрыть",
  "pomodoro.todayBlocks": "Сегодня: {{count}} блоков фокуса · {{hours}}ч {{minutes}}м"
```

- [ ] **Step 3: Validate JSON + typecheck**

Run:
```bash
cd web && node -e "require('./src/i18n/locales/en/tools.json');require('./src/i18n/locales/ru/tools.json');console.log('json ok')" && pnpm typecheck
```
Expected: `json ok`, no type errors.

- [ ] **Step 4: Commit**

```bash
git add web/src/i18n/locales/en/tools.json web/src/i18n/locales/ru/tools.json
git commit -m "i18n(pomodoro): keys for cycle, widget style, completion toast, interrupted note"
```

---

### Task 14: Floating widget (3 styles) + mount in layout

**Files:**
- Create: `web/src/components/Pomodoro/formatTime.ts`
- Create: `web/src/components/Pomodoro/PillWidget.tsx`
- Create: `web/src/components/Pomodoro/CardWidget.tsx`
- Create: `web/src/components/Pomodoro/BarWidget.tsx`
- Create: `web/src/components/Pomodoro/PomodoroWidget.tsx`
- Modify: `web/src/router.tsx:71-82` (AppLayout)

- [ ] **Step 1: Shared formatter + pip/color constants**

`web/src/components/Pomodoro/formatTime.ts`:
```ts
import type { TimerMode } from '../../lib/pomodoro/types'

export function formatMs(ms: number): string {
  const total = Math.max(0, Math.round(ms / 1000))
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

export const MODE_HEX: Record<TimerMode, string> = {
  work: '#ef4444',
  shortBreak: '#22c55e',
  longBreak: '#3b82f6',
}

export const MODE_BTN: Record<TimerMode, string> = {
  work: 'bg-red-500',
  shortBreak: 'bg-green-500',
  longBreak: 'bg-blue-500',
}
```

- [ ] **Step 2: Pill widget**

`web/src/components/Pomodoro/PillWidget.tsx`:
```tsx
import { Play, Pause } from 'lucide-react'
import { Link } from 'react-router-dom'
import { usePomodoro } from '../../contexts/PomodoroContext'
import { formatMs, MODE_HEX } from './formatTime'

export function PillWidget() {
  const { phase, remainingMs, isRunning, start, pause } = usePomodoro()
  return (
    <div
      className="fixed bottom-4 right-4 z-40 flex items-center gap-2 rounded-full bg-zinc-900/95 px-3 py-2 shadow-xl backdrop-blur"
      style={{ border: `1px solid ${MODE_HEX[phase]}` }}
    >
      <Link to="/tools/pomodoro" className="flex items-center gap-2">
        <span className="h-2.5 w-2.5 rounded-full" style={{ background: MODE_HEX[phase] }} />
        <span className="font-mono text-base font-bold tabular-nums text-zinc-100">{formatMs(remainingMs)}</span>
      </Link>
      <button
        onClick={isRunning ? pause : start}
        className="grid h-6 w-6 place-items-center rounded-full bg-zinc-800 text-zinc-300 hover:text-zinc-100"
        aria-label={isRunning ? 'Pause' : 'Start'}
      >
        {isRunning ? <Pause className="h-3 w-3" /> : <Play className="h-3 w-3" />}
      </button>
    </div>
  )
}
```

- [ ] **Step 3: Card widget**

`web/src/components/Pomodoro/CardWidget.tsx`:
```tsx
import { Play, Pause, SkipForward } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePomodoro } from '../../contexts/PomodoroContext'
import { formatMs, MODE_HEX, MODE_BTN } from './formatTime'

export function CardWidget() {
  const { t } = useTranslation('tools')
  const { phase, remainingMs, isRunning, sessionsCompleted, linkedTaskTitle, settings, start, pause, skip } =
    usePomodoro()
  const filled = sessionsCompleted % settings.sessionsBeforeLong
  return (
    <div
      className="fixed bottom-4 right-4 z-40 w-56 rounded-xl bg-zinc-900/95 p-3 shadow-2xl backdrop-blur"
      style={{ borderLeft: `3px solid ${MODE_HEX[phase]}`, border: '1px solid #27272a' }}
    >
      <div className="mb-2 flex items-center justify-between">
        <span className="text-[10px] font-bold uppercase tracking-wide" style={{ color: MODE_HEX[phase] }}>
          {t(`pomodoro.${phase}`)}
        </span>
        <span className="flex gap-1">
          {Array.from({ length: settings.sessionsBeforeLong }).map((_, i) => (
            <span
              key={i}
              className="h-1.5 w-1.5 rounded-full"
              style={{ background: i < filled ? MODE_HEX.work : '#3f3f46' }}
            />
          ))}
        </span>
      </div>
      <div className="font-mono text-3xl font-extrabold tabular-nums text-zinc-100">{formatMs(remainingMs)}</div>
      {linkedTaskTitle && <div className="mb-2 truncate text-xs text-zinc-400">↳ {linkedTaskTitle}</div>}
      <div className="mt-2 flex gap-2">
        <button
          onClick={isRunning ? pause : start}
          className={`flex h-8 flex-[1.6] items-center justify-center rounded-lg text-white ${MODE_BTN[phase]}`}
          aria-label={isRunning ? t('pomodoro.pause') : t('pomodoro.start')}
        >
          {isRunning ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
        </button>
        <button
          onClick={skip}
          className="flex h-8 flex-1 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800 text-zinc-300"
          aria-label={t('pomodoro.skip')}
        >
          <SkipForward className="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Bar widget**

`web/src/components/Pomodoro/BarWidget.tsx`:
```tsx
import { Play, Pause } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePomodoro } from '../../contexts/PomodoroContext'
import { formatMs, MODE_HEX, MODE_BTN } from './formatTime'
import { durationMsForMode } from '../../lib/pomodoro/machine'

export function BarWidget() {
  const { t } = useTranslation('tools')
  const { phase, remainingMs, isRunning, settings, start, pause } = usePomodoro()
  const full = durationMsForMode(phase, settings)
  const pct = full > 0 ? Math.min(100, Math.max(0, ((full - remainingMs) / full) * 100)) : 0
  return (
    <div className="fixed left-1/2 top-2 z-40 flex -translate-x-1/2 items-center gap-3 rounded-full border border-zinc-700 bg-zinc-900/95 py-1.5 pl-4 pr-2 shadow-xl backdrop-blur">
      <span className="text-[11px] font-bold" style={{ color: MODE_HEX[phase] }}>
        {t(`pomodoro.${phase}`)}
      </span>
      <span className="relative h-1 w-20 overflow-hidden rounded bg-zinc-700">
        <span className="absolute left-0 top-0 h-full" style={{ width: `${pct}%`, background: MODE_HEX[phase] }} />
      </span>
      <span className="font-mono text-sm font-bold tabular-nums text-zinc-100">{formatMs(remainingMs)}</span>
      <button
        onClick={isRunning ? pause : start}
        className={`grid h-6 w-6 place-items-center rounded-full text-white ${MODE_BTN[phase]}`}
        aria-label={isRunning ? t('pomodoro.pause') : t('pomodoro.start')}
      >
        {isRunning ? <Pause className="h-3 w-3" /> : <Play className="h-3 w-3" />}
      </button>
    </div>
  )
}
```

- [ ] **Step 5: Widget switcher (hides on the Pomodoro page / when idle)**

`web/src/components/Pomodoro/PomodoroWidget.tsx`:
```tsx
import { useLocation } from 'react-router-dom'
import { usePomodoro } from '../../contexts/PomodoroContext'
import { PillWidget } from './PillWidget'
import { CardWidget } from './CardWidget'
import { BarWidget } from './BarWidget'

export function PomodoroWidget() {
  const { isActive, settings } = usePomodoro()
  const { pathname } = useLocation()

  if (!isActive) return null
  if (pathname.startsWith('/tools/pomodoro')) return null

  switch (settings.widgetStyle) {
    case 'pill':
      return <PillWidget />
    case 'bar':
      return <BarWidget />
    case 'card':
    default:
      return <CardWidget />
  }
}
```

- [ ] **Step 6: Render the widget in `AppLayout`**

In `web/src/router.tsx`, update the `AppLayout` function (lines 71–82):
```tsx
import { PomodoroWidget } from './components/Pomodoro/PomodoroWidget'
// ...
function AppLayout() {
  return (
    <>
      <Layout>
        <Suspense fallback={<PageLoader />}>
          <Outlet />
        </Suspense>
      </Layout>
      <PomodoroWidget />
      <FeedbackButton />
    </>
  )
}
```
(Add the `PomodoroWidget` import alongside the existing `FeedbackButton` import at the top of the file.)

- [ ] **Step 7: Typecheck + build**

Run:
```bash
cd web && pnpm typecheck && pnpm build
```
Expected: no errors; build succeeds.

- [ ] **Step 8: Commit**

```bash
git add web/src/components/Pomodoro/ web/src/router.tsx
git commit -m "feat(pomodoro): floating cross-page widget with pill/card/bar styles"
```

---

### Task 15: Rewrite the Pomodoro page to consume the context

**Files:**
- Rewrite: `web/src/pages/Tools/Pomodoro.tsx`

The page no longer owns timer state — it reads `usePomodoro()`, renders the cycle pips, the completion/break/interrupted toasts, the task linker (fetched here), the settings panel (now including `widgetStyle`), and today's stats from history.

- [ ] **Step 1: Replace the page**

`web/src/pages/Tools/Pomodoro.tsx`:
```tsx
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Timer,
  Play,
  Pause,
  RotateCcw,
  SkipForward,
  Settings as SettingsIcon,
  Volume2,
  VolumeX,
  ChevronDown,
  ChevronUp,
  Link as LinkIcon,
  Check,
  X,
} from 'lucide-react'
import { listTasks, type Task } from '../../api/tasks'
import { usePomodoro } from '../../contexts/PomodoroContext'
import type { TimerMode, WidgetStyle, SessionRecord } from '../../lib/pomodoro/types'
import { formatMs, MODE_HEX } from '../../components/Pomodoro/formatTime'

function loadTodayWork(): SessionRecord[] {
  try {
    const raw = localStorage.getItem('nb-pomodoro-history')
    const list: SessionRecord[] = raw ? JSON.parse(raw) : []
    const today = new Date().toISOString().slice(0, 10)
    return list.filter((r) => r.date === today && r.mode === 'work')
  } catch {
    return []
  }
}

export default function Pomodoro() {
  const { t } = useTranslation('tools')
  const {
    phase,
    remainingMs,
    isRunning,
    sessionsCompleted,
    linkedTaskId,
    settings,
    lastCompletion,
    breakOver,
    interruptedWhileAway,
    start,
    pause,
    reset,
    skip,
    selectPhase,
    setLinkedTask,
    updateSettings,
    dismissCompletion,
    undoLastCompletion,
    dismissBreakOver,
    dismissInterrupted,
  } = usePomodoro()

  const [tasks, setTasks] = useState<Task[]>([])
  const [showSettings, setShowSettings] = useState(false)

  useEffect(() => {
    listTasks()
      .then((data) => setTasks(data.filter((t) => t.status !== 'DONE' && t.status !== 'CANCELLED')))
      .catch(console.error)
  }, [])

  const today = loadTodayWork()
  const todayCount = today.length
  const todayMinutes = today.reduce((acc, r) => acc + Math.floor(r.durationSeconds / 60), 0)
  const filled = sessionsCompleted % settings.sessionsBeforeLong

  const onSelectTask = (id: string) => {
    const task = tasks.find((tk) => tk.id === id) ?? null
    setLinkedTask(id || null, task?.title ?? null)
  }

  return (
    <div className="min-h-screen bg-zinc-950 px-4 py-8 text-zinc-100">
      <div className="mx-auto flex w-full max-w-lg flex-col items-center gap-6">
        <div className="flex w-full items-center gap-2">
          <Timer className="h-5 w-5 text-zinc-400" />
          <h1 className="text-xl font-semibold">{t('pomodoro.title')}</h1>
        </div>

        {/* Mode selector */}
        <div className="flex w-full gap-2 rounded-lg bg-zinc-800 p-1">
          {(['work', 'shortBreak', 'longBreak'] as TimerMode[]).map((m) => (
            <button
              key={m}
              onClick={() => selectPhase(m)}
              className={`flex-1 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                phase === m ? 'bg-zinc-700 text-zinc-100' : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              {t(`pomodoro.${m}`)}
            </button>
          ))}
        </div>

        {/* Cycle pips */}
        <div className="flex items-center gap-2.5 text-xs text-zinc-500">
          <span className="uppercase tracking-wide">{t('pomodoro.cycle')}</span>
          <span className="flex gap-1.5">
            {Array.from({ length: settings.sessionsBeforeLong }).map((_, i) => (
              <span
                key={i}
                className="h-2.5 w-2.5 rounded-full border"
                style={{
                  background: i < filled ? MODE_HEX.work : 'transparent',
                  borderColor: i < filled ? MODE_HEX.work : '#3f3f46',
                  boxShadow: i === filled && phase === 'work' ? `0 0 0 3px rgba(239,68,68,0.18)` : undefined,
                }}
              />
            ))}
          </span>
          <span>{t('pomodoro.cycleProgress', { done: sessionsCompleted, total: settings.sessionsBeforeLong })}</span>
        </div>

        {/* Time */}
        <div className="flex flex-col items-center gap-1">
          <span className="font-mono text-6xl font-bold tabular-nums" style={{ color: MODE_HEX[phase] }}>
            {formatMs(remainingMs)}
          </span>
          <span className="text-sm text-zinc-500">{t(`pomodoro.${phase}`)}</span>
        </div>

        {/* Controls */}
        <div className="flex items-center gap-3">
          <button
            onClick={reset}
            className="rounded-full bg-zinc-800 p-3 text-zinc-400 hover:text-zinc-200"
            aria-label={t('pomodoro.reset')}
          >
            <RotateCcw className="h-5 w-5" />
          </button>
          <button
            onClick={isRunning ? pause : start}
            className="flex items-center gap-2 rounded-full px-8 py-3 font-semibold text-white"
            style={{ background: MODE_HEX[phase] }}
          >
            {isRunning ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
            {isRunning ? t('pomodoro.pause') : t('pomodoro.start')}
          </button>
          <button
            onClick={skip}
            className="rounded-full bg-zinc-800 p-3 text-zinc-400 hover:text-zinc-200"
            aria-label={t('pomodoro.skip')}
          >
            <SkipForward className="h-5 w-5" />
          </button>
        </div>

        {/* Sound toggle */}
        <button
          onClick={() => updateSettings({ soundEnabled: !settings.soundEnabled })}
          className="flex items-center gap-2 text-sm text-zinc-400 hover:text-zinc-200"
        >
          {settings.soundEnabled ? <Volume2 className="h-4 w-4" /> : <VolumeX className="h-4 w-4" />}
          {t('pomodoro.sound')}
        </button>

        {/* Today stats */}
        {todayCount > 0 && (
          <div className="w-full rounded-lg border border-zinc-700 bg-zinc-800/60 px-4 py-2 text-center text-sm text-zinc-400">
            {t('pomodoro.todayBlocks', {
              count: todayCount,
              hours: Math.floor(todayMinutes / 60),
              minutes: todayMinutes % 60,
            })}
          </div>
        )}

        {/* Task linker */}
        <div className="w-full">
          <label className="mb-1 block text-xs text-zinc-500">{t('pomodoro.linkTask')}</label>
          <select
            className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-200"
            value={linkedTaskId ?? ''}
            onChange={(e) => onSelectTask(e.target.value)}
          >
            <option value="">{t('pomodoro.noTask')}</option>
            {tasks.map((task) => (
              <option key={task.id} value={task.id}>
                {task.title}
              </option>
            ))}
          </select>
        </div>

        {/* Settings panel */}
        <div className="w-full overflow-hidden rounded-xl border border-zinc-700">
          <button
            onClick={() => setShowSettings((v) => !v)}
            className="flex w-full items-center justify-between bg-zinc-800 px-4 py-3 text-sm font-medium hover:bg-zinc-700"
          >
            <span className="flex items-center gap-2">
              <SettingsIcon className="h-4 w-4 text-zinc-400" />
              {t('pomodoro.settings')}
            </span>
            {showSettings ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </button>
          {showSettings && (
            <div className="flex flex-col gap-4 bg-zinc-900 px-4 py-4">
              {(
                [
                  ['workMinutes', 'pomodoro.workDuration'],
                  ['shortBreakMinutes', 'pomodoro.shortBreakDuration'],
                  ['longBreakMinutes', 'pomodoro.longBreakDuration'],
                  ['sessionsBeforeLong', 'pomodoro.sessionsBeforeLong'],
                ] as [keyof typeof settings, string][]
              ).map(([key, labelKey]) => (
                <div key={key} className="flex items-center justify-between gap-4">
                  <label className="text-sm text-zinc-300">{t(labelKey)}</label>
                  <input
                    type="number"
                    min={1}
                    max={key === 'sessionsBeforeLong' ? 10 : 120}
                    value={settings[key] as number}
                    onChange={(e) => {
                      const v = parseInt(e.target.value, 10)
                      if (!isNaN(v) && v > 0) updateSettings({ [key]: v })
                    }}
                    className="w-20 rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-center text-sm text-zinc-200"
                  />
                </div>
              ))}

              {/* Widget style */}
              <div className="flex items-center justify-between gap-4">
                <label className="text-sm text-zinc-300">{t('pomodoro.widgetStyle')}</label>
                <div className="flex gap-1 rounded-lg bg-zinc-800 p-1">
                  {(['pill', 'card', 'bar'] as WidgetStyle[]).map((style) => (
                    <button
                      key={style}
                      onClick={() => updateSettings({ widgetStyle: style })}
                      className={`rounded-md px-2.5 py-1 text-xs ${
                        settings.widgetStyle === style ? 'bg-zinc-700 text-zinc-100' : 'text-zinc-400'
                      }`}
                    >
                      {t(`pomodoro.widget.${style}`)}
                    </button>
                  ))}
                </div>
              </div>

              {/* Auto-start toggle */}
              <div className="flex items-center justify-between">
                <label className="text-sm text-zinc-300">{t('pomodoro.autoStartBreaks')}</label>
                <button
                  role="switch"
                  aria-checked={settings.autoStartBreaks}
                  onClick={() => updateSettings({ autoStartBreaks: !settings.autoStartBreaks })}
                  className={`relative h-6 w-10 rounded-full transition-colors ${
                    settings.autoStartBreaks ? 'bg-blue-600' : 'bg-zinc-700'
                  }`}
                >
                  <span
                    className={`absolute top-1 h-4 w-4 rounded-full bg-white transition-transform ${
                      settings.autoStartBreaks ? 'left-5' : 'left-1'
                    }`}
                  />
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Interrupted note */}
      {interruptedWhileAway && (
        <div className="fixed bottom-4 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-xl border border-zinc-700 bg-zinc-900 px-4 py-3 shadow-2xl">
          <span className="text-sm text-zinc-300">{t('pomodoro.interrupted')}</span>
          <button onClick={dismissInterrupted} className="text-zinc-500 hover:text-zinc-300" aria-label={t('pomodoro.dismiss')}>
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Break-over toast */}
      {breakOver && (
        <div className="fixed bottom-4 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-xl border border-zinc-700 bg-zinc-900 px-4 py-3 shadow-2xl">
          <span className="text-sm text-zinc-300">{t('pomodoro.breakOver')}</span>
          <button onClick={() => { dismissBreakOver(); start() }} className="rounded-lg bg-red-600 px-3 py-1 text-sm font-semibold text-white">
            {t('pomodoro.start')}
          </button>
          <button onClick={dismissBreakOver} className="text-zinc-500 hover:text-zinc-300" aria-label={t('pomodoro.dismiss')}>
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Completion toast */}
      {lastCompletion && (
        <div className="fixed bottom-4 right-4 z-50 flex max-w-sm items-center gap-3 rounded-xl border border-zinc-700 bg-zinc-900 px-4 py-3 shadow-2xl" style={{ borderLeft: `3px solid ${lastCompletion.failed ? '#f59e0b' : '#22c55e'}` }}>
          <span className="grid h-7 w-7 place-items-center rounded-lg" style={{ background: lastCompletion.failed ? 'rgba(245,158,11,0.15)' : 'rgba(34,197,94,0.15)', color: lastCompletion.failed ? '#f59e0b' : '#22c55e' }}>
            {lastCompletion.failed ? <X className="h-4 w-4" /> : <Check className="h-4 w-4" />}
          </span>
          <span className="flex-1 text-xs text-zinc-300">
            {lastCompletion.failed
              ? t('pomodoro.saveFailed')
              : lastCompletion.taskId
              ? t('pomodoro.loggedToTask', {
                  minutes: lastCompletion.minutes,
                  task: tasks.find((tk) => tk.id === lastCompletion.taskId)?.title ?? '',
                })
              : t('pomodoro.loggedNoTask', { minutes: lastCompletion.minutes })}
          </span>
          {!lastCompletion.failed && (
            <button onClick={undoLastCompletion} className="whitespace-nowrap rounded-lg border border-blue-500/40 px-3 py-1 text-xs font-bold text-blue-400">
              {t('pomodoro.undo')}
            </button>
          )}
          <button onClick={dismissCompletion} className="text-zinc-500 hover:text-zinc-300" aria-label={t('pomodoro.dismiss')}>
            <X className="h-4 w-4" />
          </button>
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 2: Typecheck + build**

Run:
```bash
cd web && pnpm typecheck && pnpm build
```
Expected: no errors; build succeeds.

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/Tools/Pomodoro.tsx
git commit -m "feat(pomodoro): rebuild page on global context — cycle pips, toasts, widget-style setting"
```

---

## Phase 6 — Verification & wrap-up

### Task 16: Full verification, manual checklist, roadmap

**Files:**
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: Run the full automated suite**

Run:
```bash
cd web && pnpm test && pnpm typecheck && pnpm build
cd ../api-go && go build ./cmd/api && go test -v ./internal/tasks/
```
Expected: all Vitest suites pass; typecheck clean; web build succeeds; Go builds; Go tests pass (or SKIP locally without `DATABASE_URL`).

- [ ] **Step 2: Manual test checklist (against `docker compose up` or dev)**

Verify each:
1. Start a work block → navigate to Calendar/Tasks → the floating widget shows the **correct, still-counting** time (try all three widget styles via the setting).
2. Reload the page mid-block → timer resumes at the correct remaining time (not reset).
3. Let a work block hit 0 → it advances to a break, a cycle pip fills, and (auto-start on) the break begins.
4. Complete a work block linked to a task → a "Focus: <task>" event appears on the calendar for the block window; the task's `actual_minutes` increased by the work duration; the toast shows "Logged … · Undo".
5. Click **Undo** → the calendar event is gone and `actual_minutes` is back down.
6. Complete a work block with **no task** → a generic "Focus" event is created; no task time logged.
7. **Skip** a work block → no calendar event, no logged time, cycle does not increment.
8. Complete 4 work blocks → the 4th tips into a **long** break.
9. Start a block, close the tab for longer than the block, reopen → **no** event/time written; a gentle "interrupted while away" note shows; timer is idle.
10. Switch language to RU → all new strings (cycle, widget styles, toast, interrupted) are translated.

- [ ] **Step 3: Update the roadmap**

In `docs/ROADMAP.md`, under "Next Sprint: v0.4.10 — Intelligent Tools", add a row to the feature table:
```markdown
| Focus Timer rebuild | Pomodoro: global timestamp-based timer (survives nav/reload), deterministic phase machine, calendar-event + task actual-minutes tracking with undo, 3-style floating widget. Fixes the reset + non-advancing bugs. Spec: `docs/superpowers/specs/2026-06-01-pomodoro-focus-timer-design.md`. |
```

- [ ] **Step 4: Commit**

```bash
git add docs/ROADMAP.md
git commit -m "docs(roadmap): record Focus Timer rebuild under v0.4.10"
```
