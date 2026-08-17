<!-- паспорт: тип=план | статус=выполнен | строк=1532 | ~токенов=13724 | обновлён=по git -->

# P1 — Task Quick Capture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Creating a simple task costs exactly one action — type the title, press Enter — while the same row expands into the full task form when more context is needed.

**Architecture:** Pure-logic-first. Every decision (defaults, settings parsing, tree indentation) lives in a tested pure function under `web/src/lib/quickTask/`; the React component is a thin shell over them. The Tasks page already uses the snake_case API stack (`web/src/api/tasks.ts`), so quick-capture builds on that stack and needs **no type conversion and no migration** — `"user".settings` is already `JSONB` (migration `000005`) and `task.parent_id` already exists (migration `000001`).

**Tech Stack:** React 18 + TypeScript strict, Vitest + jsdom, Tailwind 3.4, Lucide icons, Go 1.22 + chi + pgx/v5.

**Spec:** `docs/superpowers/specs/2026-07-27-p1-task-quick-capture-design.md`

## Global Constraints

- **No `any` in TypeScript.** Project-wide rule (root `CLAUDE.md`).
- **Two parallel task API stacks exist.** `web/src/api/tasks.ts` is snake_case and is what `pages/Tasks/Tasks.tsx` imports. `web/src/api/index.ts` + `web/src/types/index.ts` are camelCase and are what `TaskSidebar` imports. **All P1 UI work uses the snake_case stack.** Task 1 fixes the camelCase read path; do not migrate surfaces between stacks in this plan.
- **Verify before done — frontend:** `cd web && pnpm typecheck && pnpm test --run && pnpm build`. **Backend:** `cd api-go && go build ./... && go test ./...`.
- **pnpm is pinned** via `packageManager` (10.12.0). If missing, run `corepack enable` once — never `npm i -g pnpm`.
- **Priority is inverted:** 1 = Emergency (highest), 5 = If Possible (lowest), 0 = Buffer. Default is 3.
- **Never commit unless Denis asks.** Steps below say "Commit" — run them only when he has said to commit; otherwise leave the work staged-but-uncommitted and tell him.
- **No new dependencies** without asking first.
- Tests are co-located: `foo.ts` → `foo.test.ts`.

---

### Task 1: Fix T1 — `getTasks` returns snake_case cast to a camelCase type

The camelCase `Task` (`web/src/types/index.ts:2-20`) declares `estimatedMinutes`, `dueDate`, `parentId`. The Go API emits `estimated_minutes`, `due_date`, `parent_id` (`api-go/internal/tasks/types.go:37-43`). `getTasks` (`web/src/api/index.ts:161-168`) casts with **no conversion**, so every consumer of that path reads `undefined`.

Real consequence today: `TaskSidebar/useDragTask.ts:13` passes `task.estimatedMinutes` → `Calendar.tsx:106` → `scheduleTask(..., undefined)` → `api/index.ts:208` falls back to **60 minutes for every task dragged onto the calendar**, regardless of its real estimate.

Only the **read** path is broken. Writes are fine; stored data is correct.

**Files:**
- Create: `web/src/api/toTask.ts`
- Create: `web/src/api/toTask.test.ts`
- Modify: `web/src/api/index.ts:160-168` (`getTasks`)

**Interfaces:**
- Consumes: nothing.
- Produces: `toTask(raw: RawTask): Task` and `interface RawTask` from `web/src/api/toTask.ts`, where `Task` is the camelCase type from `web/src/types`.

- [ ] **Step 1: Write the failing test**

```ts
// web/src/api/toTask.test.ts
import { describe, it, expect } from 'vitest'
import { toTask, type RawTask } from './toTask'

const raw: RawTask = {
  id: 't1',
  user_id: 'u1',
  title: 'Дописать отчёт',
  status: 'TODO',
  priority: 2,
  estimated_minutes: 15,
  due_date: '2026-07-28T09:00:00Z',
  tags: ['work'],
  contexts: ['@computer'],
  parent_id: 'p1',
  created_at: '2026-07-27T20:00:00Z',
  updated_at: '2026-07-27T20:00:00Z',
}

describe('toTask', () => {
  it('maps snake_case fields onto the camelCase Task type', () => {
    const task = toTask(raw)
    expect(task.estimatedMinutes).toBe(15)
    expect(task.dueDate).toBe('2026-07-28T09:00:00Z')
    expect(task.parentId).toBe('p1')
    expect(task.userId).toBe('u1')
    expect(task.createdAt).toBe('2026-07-27T20:00:00Z')
  })

  it('keeps fields that need no renaming', () => {
    const task = toTask(raw)
    expect(task.id).toBe('t1')
    expect(task.title).toBe('Дописать отчёт')
    expect(task.priority).toBe(2)
    expect(task.status).toBe('TODO')
  })

  it('defaults arrays to [] and leaves absent optionals undefined', () => {
    const task = toTask({
      id: 't2',
      user_id: 'u1',
      title: 'Купить хлеб',
      status: 'TODO',
      priority: 3,
      created_at: '2026-07-27T20:00:00Z',
      updated_at: '2026-07-27T20:00:00Z',
    })
    expect(task.tags).toEqual([])
    expect(task.contexts).toEqual([])
    expect(task.estimatedMinutes).toBeUndefined()
    expect(task.dueDate).toBeUndefined()
    expect(task.parentId).toBeUndefined()
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && pnpm test --run src/api/toTask.test.ts`
Expected: FAIL — cannot resolve `./toTask`.

- [ ] **Step 3: Write the minimal implementation**

```ts
// web/src/api/toTask.ts
import type { Task } from '../types'

/**
 * Shape the Go API actually sends for a task (snake_case).
 * Mirrors api-go/internal/tasks/types.go:30-48.
 */
export interface RawTask {
  id: string
  user_id: string
  title: string
  description?: string
  status: Task['status']
  priority: number
  estimated_minutes?: number
  actual_minutes?: number
  due_date?: string
  tags?: string[]
  contexts?: string[]
  energy?: number
  parent_id?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

/** Convert an API task into the camelCase Task used by TaskSidebar and Calendar. */
export function toTask(raw: RawTask): Task {
  return {
    id: raw.id,
    userId: raw.user_id,
    title: raw.title,
    description: raw.description,
    status: raw.status,
    priority: raw.priority,
    estimatedMinutes: raw.estimated_minutes,
    dueDate: raw.due_date,
    tags: raw.tags ?? [],
    contexts: raw.contexts ?? [],
    energy: raw.energy,
    parentId: raw.parent_id,
    completedAt: raw.completed_at,
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
  }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && pnpm test --run src/api/toTask.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Wire it into `getTasks`**

Replace the body of `getTasks` in `web/src/api/index.ts` (currently lines 160-168):

```ts
/** Fetch tasks with optional filters */
export async function getTasks(status?: string, priority?: number): Promise<import('../types').Task[]> {
  const params = new URLSearchParams();
  if (status) params.append('status', status);
  if (priority !== undefined) params.append('priority', String(priority));

  const response = await api.get<{ tasks: RawTask[] } | RawTask[]>(`/tasks?${params}`);
  const rows = Array.isArray(response) ? response : response.tasks || [];
  return rows.map(toTask);
}
```

Add at the top of the file, next to the other imports:

```ts
import { toTask, type RawTask } from './toTask';
```

- [ ] **Step 6: Verify the whole frontend is still green**

Run: `cd web && pnpm typecheck && pnpm test --run`
Expected: typecheck clean, all tests pass (previous count + 3).

- [ ] **Step 7: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/api/toTask.ts web/src/api/toTask.test.ts web/src/api/index.ts
git commit -m "fix(tasks): convert snake_case API rows to the camelCase Task type

getTasks cast raw API rows straight to Task, so estimatedMinutes, dueDate
and parentId were always undefined. Dragging a task onto the calendar
therefore always scheduled 60 minutes instead of its real estimate."
```

---

### Task 2: Quick-task settings — typed, defaulted, garbage-resistant

Settings live in `"user".settings` (JSONB, migration `000005`) and reach the frontend as `UserSettings` (`web/src/api/auth.ts:20-30`). The blob is user-writable and may hold anything, so parsing must never throw and never yield an out-of-range value.

**Files:**
- Modify: `web/src/api/auth.ts:20-30` (add `quick_task` to `UserSettings`)
- Create: `web/src/lib/quickTask/settings.ts`
- Create: `web/src/lib/quickTask/settings.test.ts`

**Interfaces:**
- Consumes: `UserSettings` from `web/src/api/auth.ts`.
- Produces: `resolveQuickTaskSettings(settings: UserSettings | null | undefined): QuickTaskSettings`, plus the exported types `QuickTaskSettings`, `QuickTaskDefaults`, `DefaultDue`, and the constant `QUICK_TASK_DEFAULTS`.

- [ ] **Step 1: Write the failing test**

```ts
// web/src/lib/quickTask/settings.test.ts
import { describe, it, expect } from 'vitest'
import { resolveQuickTaskSettings, QUICK_TASK_DEFAULTS } from './settings'

describe('resolveQuickTaskSettings', () => {
  it('returns the documented defaults when nothing is stored', () => {
    expect(resolveQuickTaskSettings(undefined)).toEqual(QUICK_TASK_DEFAULTS)
    expect(resolveQuickTaskSettings(null)).toEqual(QUICK_TASK_DEFAULTS)
    expect(resolveQuickTaskSettings({})).toEqual(QUICK_TASK_DEFAULTS)
  })

  it('defaults to tomorrow, priority 3, 15 minutes, filters off', () => {
    expect(QUICK_TASK_DEFAULTS.default_due).toBe('tomorrow')
    expect(QUICK_TASK_DEFAULTS.default_priority).toBe(3)
    expect(QUICK_TASK_DEFAULTS.default_estimate_minutes).toBe(15)
    expect(QUICK_TASK_DEFAULTS.inherit_filters).toBe(false)
  })

  it('accepts valid stored values', () => {
    const r = resolveQuickTaskSettings({
      quick_task: {
        default_due: 'today',
        default_priority: 1,
        default_estimate_minutes: 5,
        inherit_filters: true,
      },
    })
    expect(r.default_due).toBe('today')
    expect(r.default_priority).toBe(1)
    expect(r.default_estimate_minutes).toBe(5)
    expect(r.inherit_filters).toBe(true)
  })

  it('accepts null estimate as "do not set an estimate"', () => {
    const r = resolveQuickTaskSettings({ quick_task: { default_estimate_minutes: null } })
    expect(r.default_estimate_minutes).toBeNull()
  })

  it('falls back per-field on garbage without throwing', () => {
    const r = resolveQuickTaskSettings({
      // The JSONB blob is user-writable; a bad field must not poison the others.
      quick_task: {
        default_due: 'yesterday',
        default_priority: 99,
        default_estimate_minutes: -5,
        inherit_filters: 'yes',
      },
    } as never)
    expect(r).toEqual(QUICK_TASK_DEFAULTS)
  })

  it('rejects a non-object quick_task', () => {
    expect(resolveQuickTaskSettings({ quick_task: 'nope' } as never)).toEqual(QUICK_TASK_DEFAULTS)
  })

  it('merges stored keybindings over the defaults', () => {
    const r = resolveQuickTaskSettings({ quick_task: { keys: { expand: 'Ctrl+D' } } } as never)
    expect(r.keys.expand).toBe('Ctrl+D')
    expect(r.keys.submit).toBe(QUICK_TASK_DEFAULTS.keys.submit)
    expect(r.keys.indent).toBe('Alt+ArrowRight')
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && pnpm test --run src/lib/quickTask/settings.test.ts`
Expected: FAIL — cannot resolve `./settings`.

- [ ] **Step 3: Write the minimal implementation**

```ts
// web/src/lib/quickTask/settings.ts
import type { UserSettings } from '../../api/auth'

export type DefaultDue = 'today' | 'tomorrow' | 'none'

export interface QuickTaskKeys {
  submit: string
  submit_expanded: string
  expand: string
  global_capture: string
  indent: string
  outdent: string
}

export interface QuickTaskSettings {
  default_due: DefaultDue
  default_priority: number
  /** null means "do not set an estimate". */
  default_estimate_minutes: number | null
  inherit_filters: boolean
  keys: QuickTaskKeys
}

export const QUICK_TASK_DEFAULTS: QuickTaskSettings = {
  default_due: 'tomorrow',
  default_priority: 3,
  default_estimate_minutes: 15,
  inherit_filters: false,
  keys: {
    submit: 'Enter',
    submit_expanded: 'Ctrl+Enter',
    expand: 'Ctrl+E',
    global_capture: 'Ctrl+K',
    // Not Tab: Tab is the browser's focus key and capturing it would trap
    // keyboard users inside the input (WCAG 2.1.2).
    indent: 'Alt+ArrowRight',
    outdent: 'Alt+ArrowLeft',
  },
}

const DUE_VALUES: readonly DefaultDue[] = ['today', 'tomorrow', 'none']

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/**
 * Read quick-task preferences out of the user settings JSONB blob.
 * The blob is user-writable, so every field falls back independently
 * and nothing here may throw.
 */
export function resolveQuickTaskSettings(settings: UserSettings | null | undefined): QuickTaskSettings {
  const raw: unknown = settings?.quick_task
  if (!isRecord(raw)) return QUICK_TASK_DEFAULTS

  const due = raw.default_due
  const priority = raw.default_priority
  const estimate = raw.default_estimate_minutes
  const inherit = raw.inherit_filters
  const keys = isRecord(raw.keys) ? raw.keys : {}

  const validKeys: QuickTaskKeys = { ...QUICK_TASK_DEFAULTS.keys }
  for (const name of Object.keys(QUICK_TASK_DEFAULTS.keys) as (keyof QuickTaskKeys)[]) {
    const bound = keys[name]
    if (typeof bound === 'string' && bound.trim() !== '') validKeys[name] = bound
  }

  return {
    default_due: DUE_VALUES.includes(due as DefaultDue) ? (due as DefaultDue) : QUICK_TASK_DEFAULTS.default_due,
    default_priority:
      typeof priority === 'number' && Number.isInteger(priority) && priority >= 0 && priority <= 5
        ? priority
        : QUICK_TASK_DEFAULTS.default_priority,
    default_estimate_minutes:
      estimate === null
        ? null
        : typeof estimate === 'number' && Number.isInteger(estimate) && estimate > 0 && estimate <= 1440
          ? estimate
          : QUICK_TASK_DEFAULTS.default_estimate_minutes,
    inherit_filters: typeof inherit === 'boolean' ? inherit : QUICK_TASK_DEFAULTS.inherit_filters,
    keys: validKeys,
  }
}
```

- [ ] **Step 4: Add `quick_task` to the `UserSettings` interface**

In `web/src/api/auth.ts`, inside `interface UserSettings` (currently lines 20-30), add:

```ts
  quick_task?: {
    default_due?: 'today' | 'tomorrow' | 'none'
    default_priority?: number
    default_estimate_minutes?: number | null
    inherit_filters?: boolean
    keys?: Partial<Record<'submit' | 'submit_expanded' | 'expand' | 'global_capture' | 'indent' | 'outdent', string>>
  }
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd web && pnpm test --run src/lib/quickTask/settings.test.ts && pnpm typecheck`
Expected: PASS (7 tests), typecheck clean.

- [ ] **Step 6: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/lib/quickTask/settings.ts web/src/lib/quickTask/settings.test.ts web/src/api/auth.ts
git commit -m "feat(tasks): typed quick-task preferences with per-field fallback"
```

---

### Task 3: `buildQuickTask` — turn a title into a full CreateTaskRequest

The one place that decides what a one-keystroke task actually contains. Pure, so it is fully testable without React.

**Files:**
- Create: `web/src/lib/quickTask/buildQuickTask.ts`
- Create: `web/src/lib/quickTask/buildQuickTask.test.ts`

**Interfaces:**
- Consumes: `QuickTaskSettings` from `./settings`; `CreateTaskRequest` from `web/src/api/tasks`.
- Produces: `buildQuickTask(input: QuickTaskInput): CreateTaskRequest | null` and `interface QuickTaskInput`.

Due dates are stored as `TIMESTAMPTZ` and parsed by Go with `time.Parse(time.RFC3339, ...)` (`api-go/internal/tasks/handlers.go:81`), so the value must be a full RFC3339 string. "Tomorrow" means **the start of the user's next local day**, expressed in UTC — computing it off a UTC date would land Denis's tasks on the wrong day, which is exactly bug class the v0.4.10 due-date fix already dealt with once.

- [ ] **Step 1: Write the failing test**

```ts
// web/src/lib/quickTask/buildQuickTask.test.ts
import { describe, it, expect } from 'vitest'
import { buildQuickTask } from './buildQuickTask'
import { QUICK_TASK_DEFAULTS } from './settings'

// 2026-07-27 23:40 in Europe/Moscow (UTC+3) — late enough that a naive
// UTC-based "tomorrow" would still say the 28th while local time says the 28th too;
// 21:40Z is deliberately chosen to sit on the other side of local midnight.
const NOW = new Date('2026-07-27T21:40:00Z')

describe('buildQuickTask', () => {
  it('returns null for an empty or whitespace-only title', () => {
    expect(buildQuickTask({ title: '', settings: QUICK_TASK_DEFAULTS, now: NOW })).toBeNull()
    expect(buildQuickTask({ title: '   ', settings: QUICK_TASK_DEFAULTS, now: NOW })).toBeNull()
  })

  it('trims the title', () => {
    const task = buildQuickTask({ title: '  купить хлеб  ', settings: QUICK_TASK_DEFAULTS, now: NOW })
    expect(task?.title).toBe('купить хлеб')
  })

  it('applies the default priority and estimate', () => {
    const task = buildQuickTask({ title: 'позвонить', settings: QUICK_TASK_DEFAULTS, now: NOW })
    expect(task?.priority).toBe(3)
    expect(task?.estimated_minutes).toBe(15)
    expect(task?.status).toBe('TODO')
  })

  it('omits the estimate when the setting is null', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: { ...QUICK_TASK_DEFAULTS, default_estimate_minutes: null },
      now: NOW,
    })
    expect(task?.estimated_minutes).toBeUndefined()
  })

  it('sets due_date to the start of the next local day for "tomorrow"', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: QUICK_TASK_DEFAULTS,
      now: NOW,
      timeZone: 'Europe/Moscow',
    })
    // Local time is 2026-07-28 00:40 (+03), so "tomorrow" is the 29th local,
    // i.e. 2026-07-28T21:00:00Z.
    expect(task?.due_date).toBe('2026-07-28T21:00:00.000Z')
  })

  it('sets due_date to the start of the current local day for "today"', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: { ...QUICK_TASK_DEFAULTS, default_due: 'today' },
      now: NOW,
      timeZone: 'Europe/Moscow',
    })
    expect(task?.due_date).toBe('2026-07-27T21:00:00.000Z')
  })

  it('omits due_date entirely for "none"', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: { ...QUICK_TASK_DEFAULTS, default_due: 'none' },
      now: NOW,
      timeZone: 'Europe/Moscow',
    })
    expect(task?.due_date).toBeUndefined()
  })

  it('ignores the active filters unless inherit_filters is on', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: QUICK_TASK_DEFAULTS,
      now: NOW,
      filters: { tags: ['work'], contexts: ['@computer'] },
    })
    expect(task?.tags).toEqual([])
    expect(task?.contexts).toEqual([])
  })

  it('inherits the active filters when inherit_filters is on', () => {
    const task = buildQuickTask({
      title: 'позвонить',
      settings: { ...QUICK_TASK_DEFAULTS, inherit_filters: true },
      now: NOW,
      filters: { tags: ['work'], contexts: ['@computer'] },
    })
    expect(task?.tags).toEqual(['work'])
    expect(task?.contexts).toEqual(['@computer'])
  })

  it('attaches a parent when one is given', () => {
    const task = buildQuickTask({
      title: 'собрать цифры',
      settings: QUICK_TASK_DEFAULTS,
      now: NOW,
      parentId: 'parent-uuid',
    })
    expect(task?.parent_id).toBe('parent-uuid')
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && pnpm test --run src/lib/quickTask/buildQuickTask.test.ts`
Expected: FAIL — cannot resolve `./buildQuickTask`.

- [ ] **Step 3: Write the minimal implementation**

```ts
// web/src/lib/quickTask/buildQuickTask.ts
import type { CreateTaskRequest } from '../../api/tasks'
import type { QuickTaskSettings } from './settings'

export interface QuickTaskFilters {
  tags?: string[]
  contexts?: string[]
}

export interface QuickTaskInput {
  title: string
  settings: QuickTaskSettings
  now: Date
  /** IANA zone; defaults to the browser's. */
  timeZone?: string
  filters?: QuickTaskFilters
  parentId?: string
}

/**
 * Midnight at the start of the local day `offsetDays` away from `now`,
 * returned as an absolute instant. Computed from the zone's own wall-clock
 * parts rather than from UTC, so a task created at 00:40 Moscow time is not
 * filed under the previous day.
 */
function startOfLocalDay(now: Date, timeZone: string, offsetDays: number): Date {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).formatToParts(now)

  const get = (type: Intl.DateTimeFormatPartTypes): number =>
    Number(parts.find(p => p.type === type)?.value ?? '0')

  const localAsUtc = Date.UTC(get('year'), get('month') - 1, get('day'), get('hour'), get('minute'), get('second'))
  // How far the zone runs ahead of UTC at this instant.
  const zoneOffsetMs = localAsUtc - Math.floor(now.getTime() / 1000) * 1000
  const midnightLocalAsUtc = Date.UTC(get('year'), get('month') - 1, get('day') + offsetDays)
  return new Date(midnightLocalAsUtc - zoneOffsetMs)
}

/**
 * Expand a typed title into a full create request.
 * Returns null when there is nothing to create, so an accidental Enter on an
 * empty field cannot produce a blank task.
 */
export function buildQuickTask(input: QuickTaskInput): CreateTaskRequest | null {
  const title = input.title.trim()
  if (title === '') return null

  const { settings } = input
  const timeZone = input.timeZone ?? Intl.DateTimeFormat().resolvedOptions().timeZone

  const request: CreateTaskRequest = {
    title,
    status: 'TODO',
    priority: settings.default_priority,
    tags: settings.inherit_filters ? (input.filters?.tags ?? []) : [],
    contexts: settings.inherit_filters ? (input.filters?.contexts ?? []) : [],
  }

  if (settings.default_estimate_minutes !== null) {
    request.estimated_minutes = settings.default_estimate_minutes
  }

  if (settings.default_due !== 'none') {
    const offset = settings.default_due === 'tomorrow' ? 1 : 0
    request.due_date = startOfLocalDay(input.now, timeZone, offset).toISOString()
  }

  if (input.parentId) request.parent_id = input.parentId

  return request
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && pnpm test --run src/lib/quickTask/buildQuickTask.test.ts`
Expected: PASS (11 tests).

If the two timezone assertions fail, do **not** loosen the assertion — the expected values are correct for Europe/Moscow (UTC+3, no DST since 2014). Fix `startOfLocalDay`.

- [ ] **Step 5: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/lib/quickTask/buildQuickTask.ts web/src/lib/quickTask/buildQuickTask.test.ts
git commit -m "feat(tasks): buildQuickTask expands a title into a full create request"
```

---

### Task 4: The `QuickAddRow` component on the Tasks page

The visible payoff: a permanently-focused input at the top of the list. Enter creates and **keeps the focus**, so the next title can be typed immediately.

**Files:**
- Create: `web/src/components/QuickAdd/QuickAddRow.tsx`
- Create: `web/src/components/QuickAdd/index.ts`
- Modify: `web/src/pages/Tasks/Tasks.tsx` (render it above the list; add `handleQuickCreate`)
- Modify: `web/src/i18n/locales/en/tasks.json`, `web/src/i18n/locales/ru/tasks.json`

**Interfaces:**
- Consumes: `buildQuickTask` (Task 3), `resolveQuickTaskSettings` (Task 2), `createTask` + `Task` + `CreateTaskRequest` from `web/src/api/tasks`, `useAuthContext` from `web/src/contexts/AuthContext`.
- Produces: `<QuickAddRow onCreate={(req: CreateTaskRequest) => Promise<void>} onOpenFull={(title: string) => void} filters={QuickTaskFilters} />`.

- [ ] **Step 1: Write the component**

```tsx
// web/src/components/QuickAdd/QuickAddRow.tsx
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Loader2 } from 'lucide-react'
import { useAuthContext } from '../../contexts/AuthContext'
import { resolveQuickTaskSettings } from '../../lib/quickTask/settings'
import { buildQuickTask, type QuickTaskFilters } from '../../lib/quickTask/buildQuickTask'
import type { CreateTaskRequest } from '../../api/tasks'

interface QuickAddRowProps {
  onCreate: (request: CreateTaskRequest) => Promise<void>
  onOpenFull: (title: string) => void
  filters?: QuickTaskFilters
  /** Take focus on mount. The Tasks page passes true. */
  autoFocus?: boolean
}

export function QuickAddRow({ onCreate, onOpenFull, filters, autoFocus = false }: QuickAddRowProps) {
  const { t } = useTranslation('tasks')
  const { user } = useAuthContext()
  const [title, setTitle] = useState('')
  const [busy, setBusy] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const settings = resolveQuickTaskSettings(user?.settings)

  useEffect(() => {
    if (autoFocus) inputRef.current?.focus()
  }, [autoFocus])

  async function submit() {
    const request = buildQuickTask({ title, settings, now: new Date(), filters })
    // Empty input: nothing to do, and the focus must not move.
    if (!request || busy) return
    setBusy(true)
    // Clear optimistically so the next title can be typed while the request flies.
    setTitle('')
    try {
      await onCreate(request)
    } catch {
      // Put the text back rather than losing what was typed.
      setTitle(request.title)
    } finally {
      setBusy(false)
      inputRef.current?.focus()
    }
  }

  return (
    <div className="flex items-stretch gap-2 mb-4">
      <div className="flex flex-1 items-center gap-2 rounded-lg border border-zinc-700 bg-zinc-900 px-3 focus-within:border-blue-500">
        <Plus className="h-4 w-4 shrink-0 text-zinc-500" aria-hidden="true" />
        <input
          ref={inputRef}
          value={title}
          onChange={e => setTitle(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter') {
              e.preventDefault()
              void submit()
            }
          }}
          placeholder={t('quickAdd.placeholder')}
          aria-label={t('quickAdd.placeholder')}
          className="w-full bg-transparent py-2 text-zinc-100 outline-none placeholder:text-zinc-600"
        />
        {busy && <Loader2 className="h-4 w-4 shrink-0 animate-spin text-zinc-500" aria-hidden="true" />}
      </div>
      <button
        type="button"
        onClick={() => onOpenFull(title)}
        className="rounded-lg border border-zinc-700 px-3 text-sm text-zinc-400 hover:border-blue-500 hover:text-zinc-100"
      >
        {t('quickAdd.full')}
      </button>
    </div>
  )
}
```

```ts
// web/src/components/QuickAdd/index.ts
export { QuickAddRow } from './QuickAddRow'
```

- [ ] **Step 2: Add the translation strings**

In `web/src/i18n/locales/en/tasks.json`:

```json
  "quickAdd": {
    "placeholder": "New task — type and press Enter",
    "full": "Full task"
  }
```

In `web/src/i18n/locales/ru/tasks.json`:

```json
  "quickAdd": {
    "placeholder": "Новая задача — введи и нажми Enter",
    "full": "Полная задача"
  }
```

- [ ] **Step 3: Render it on the Tasks page**

In `web/src/pages/Tasks/Tasks.tsx`, add the import next to the existing ones:

```ts
import { QuickAddRow } from '../../components/QuickAdd'
```

Add the handler alongside the other handlers in the component body:

```ts
  async function handleQuickCreate(request: CreateTaskRequest) {
    const created = await createTask(request)
    setTasks(prev => [created, ...prev])
  }
```

`CreateTaskRequest` must be added to the existing import from `'../../api/tasks'`.

Render `<QuickAddRow>` immediately above the task list (just before the `loading ? ... : filteredTasks.length === 0 ? ...` block around line 280):

```tsx
        <QuickAddRow
          autoFocus
          onCreate={handleQuickCreate}
          onOpenFull={title => {
            setEditingTask({ title })
            setShowEditor(true)
          }}
        />
```

- [ ] **Step 4: Verify**

Run: `cd web && pnpm typecheck && pnpm test --run && pnpm build`
Expected: all three clean.

- [ ] **Step 5: Check it by hand**

Run `cd web && pnpm dev`, open `/tasks` and confirm:
- The input holds focus on page load.
- Typing a title and pressing Enter creates the task, clears the field, keeps the focus.
- Three tasks can be entered back-to-back without touching the mouse.
- Pressing Enter on an empty field does nothing.
- **`Tab` still moves focus to "Full task"** — there must be no keyboard trap.
- "Full task" opens the existing modal with the typed title already in it.

- [ ] **Step 6: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/components/QuickAdd web/src/pages/Tasks/Tasks.tsx web/src/i18n/locales/en/tasks.json web/src/i18n/locales/ru/tasks.json
git commit -m "feat(tasks): quick-add row — one action per simple task"
```

---

**Tasks 1–4 ship a usable product.** Everything below adds range on top of a working core; stop and re-evaluate with Denis before continuing.

---

### Task 5: Progressive field levels (0 → 1 → 2)

**Files:**
- Modify: `web/src/components/QuickAdd/QuickAddRow.tsx`
- Create: `web/src/components/QuickAdd/QuickAddFields.tsx`
- Modify: `web/src/i18n/locales/{en,ru}/tasks.json`

**Interfaces:**
- Consumes: `CreateTaskRequest`, `QuickTaskSettings`.
- Produces: `<QuickAddFields level={1 | 2} draft={Partial<CreateTaskRequest>} onChange={(patch: Partial<CreateTaskRequest>) => void} />`.

The Enter rule is the part that must not be improvised: **`Enter` submits only from the title input.** In every other field `Enter` keeps its native behaviour (newline in `description`, option choice in the priority `<select>`, confirm in the date input) and only `Ctrl+Enter` submits.

- [ ] **Step 1: Add the level state and the chevron to `QuickAddRow`**

```tsx
  const [level, setLevel] = useState<0 | 1 | 2>(0)
  const [draft, setDraft] = useState<Partial<CreateTaskRequest>>({})
```

Chevron button, placed inside the bordered container after the input:

```tsx
        <button
          type="button"
          onClick={() => setLevel(l => (l === 2 ? 0 : ((l + 1) as 0 | 1 | 2)))}
          aria-expanded={level > 0}
          aria-label={t('quickAdd.expand')}
          className="shrink-0 rounded p-1 text-zinc-500 hover:text-zinc-200"
        >
          <ChevronDown className={`h-4 w-4 transition-transform ${level > 0 ? 'rotate-180' : ''}`} />
        </button>
```

Import `ChevronDown` from `lucide-react`.

- [ ] **Step 2: Handle Ctrl+E and Ctrl+Enter on the container**

Add to the wrapper `div`:

```tsx
      onKeyDown={e => {
        if (e.ctrlKey && (e.key === 'e' || e.key === 'E')) {
          e.preventDefault()
          setLevel(l => (l === 2 ? 0 : ((l + 1) as 0 | 1 | 2)))
        }
        if (e.ctrlKey && e.key === 'Enter') {
          e.preventDefault()
          void submit()
        }
        if (e.key === 'Escape' && level > 0) {
          e.preventDefault()
          setLevel(0)
        }
      }}
```

- [ ] **Step 3: Write `QuickAddFields`**

```tsx
// web/src/components/QuickAdd/QuickAddFields.tsx
import { useTranslation } from 'react-i18next'
import type { CreateTaskRequest } from '../../api/tasks'
import { PRIORITY_LABELS } from '../../lib/priority'

interface QuickAddFieldsProps {
  level: 1 | 2
  draft: Partial<CreateTaskRequest>
  onChange: (patch: Partial<CreateTaskRequest>) => void
}

export function QuickAddFields({ level, draft, onChange }: QuickAddFieldsProps) {
  const { t } = useTranslation('tasks')
  return (
    <div className="mb-4 grid gap-3 rounded-lg border border-zinc-800 bg-zinc-900/60 p-3 sm:grid-cols-2">
      <label className="grid gap-1 text-sm text-zinc-400">
        {t('form.priority')}
        <select
          value={draft.priority ?? 3}
          onChange={e => onChange({ priority: Number(e.target.value) })}
          className="rounded border border-zinc-700 bg-zinc-900 px-2 py-1 text-zinc-100"
        >
          {[1, 2, 3, 4, 5].map(p => (
            <option key={p} value={p}>{t(PRIORITY_LABELS[p])}</option>
          ))}
        </select>
      </label>

      <label className="grid gap-1 text-sm text-zinc-400">
        {t('form.estimatedTime')}
        <input
          type="number"
          min={1}
          value={draft.estimated_minutes ?? ''}
          onChange={e => onChange({ estimated_minutes: e.target.value === '' ? undefined : Number(e.target.value) })}
          className="rounded border border-zinc-700 bg-zinc-900 px-2 py-1 text-zinc-100"
        />
      </label>

      <label className="grid gap-1 text-sm text-zinc-400">
        {t('form.dueDate')}
        <input
          type="datetime-local"
          onChange={e => onChange({ due_date: e.target.value === '' ? undefined : new Date(e.target.value).toISOString() })}
          className="rounded border border-zinc-700 bg-zinc-900 px-2 py-1 text-zinc-100"
        />
      </label>

      <label className="grid gap-1 text-sm text-zinc-400">
        {t('form.tags')}
        <input
          value={(draft.tags ?? []).join(', ')}
          onChange={e => onChange({ tags: e.target.value.split(',').map(s => s.trim()).filter(Boolean) })}
          className="rounded border border-zinc-700 bg-zinc-900 px-2 py-1 text-zinc-100"
        />
      </label>

      {level === 2 && (
        <label className="grid gap-1 text-sm text-zinc-400 sm:col-span-2">
          {t('form.description')}
          <textarea
            rows={3}
            value={draft.description ?? ''}
            onChange={e => onChange({ description: e.target.value })}
            className="rounded border border-zinc-700 bg-zinc-900 px-2 py-1 text-zinc-100"
          />
        </label>
      )}
    </div>
  )
}
```

Verify `PRIORITY_LABELS` keys before wiring: they come from `web/src/lib/priority.ts` and are already used by `Tasks.tsx`. Match whatever shape that module exports rather than assuming.

- [ ] **Step 4: Merge the draft into the built request**

In `submit()`, after `buildQuickTask` returns:

```ts
    const request = { ...built, ...draft, title: built.title }
```

so an explicitly typed field always beats the default, but the trimmed title always wins.

- [ ] **Step 5: Verify**

Run: `cd web && pnpm typecheck && pnpm test --run && pnpm build`

Then by hand: `Ctrl+E` expands, `Enter` inside `description` inserts a newline (does **not** submit), `Ctrl+Enter` submits from any field, `Esc` collapses without losing the typed title.

- [ ] **Step 6: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/components/QuickAdd web/src/i18n/locales/en/tasks.json web/src/i18n/locales/ru/tasks.json
git commit -m "feat(tasks): progressive field levels on the quick-add row"
```

---

### Task 6: Tree building with Alt+→ / Alt+←

**Files:**
- Create: `web/src/lib/quickTask/indent.ts`
- Create: `web/src/lib/quickTask/indent.test.ts`
- Modify: `web/src/components/QuickAdd/QuickAddRow.tsx`

**Interfaces:**
- Consumes: nothing.
- Produces: `nextParentId(trail: TrailEntry[], direction: 'in' | 'out'): string | undefined` and `interface TrailEntry { id: string; parentId?: string }`.

The trail is the list of tasks created in the current quick-add session, most recent last. Indenting means "the next task becomes a child of the last created one"; outdenting means "go up one level".

- [ ] **Step 1: Write the failing test**

```ts
// web/src/lib/quickTask/indent.test.ts
import { describe, it, expect } from 'vitest'
import { nextParentId, type TrailEntry } from './indent'

describe('nextParentId', () => {
  it('returns undefined when nothing has been created yet', () => {
    expect(nextParentId([], 'in')).toBeUndefined()
    expect(nextParentId([], 'out')).toBeUndefined()
  })

  it('indenting makes the last created task the parent', () => {
    const trail: TrailEntry[] = [{ id: 'a' }]
    expect(nextParentId(trail, 'in')).toBe('a')
  })

  it('indenting twice nests under the newest child', () => {
    const trail: TrailEntry[] = [{ id: 'a' }, { id: 'b', parentId: 'a' }]
    expect(nextParentId(trail, 'in')).toBe('b')
  })

  it('outdenting climbs to the grandparent', () => {
    const trail: TrailEntry[] = [{ id: 'a' }, { id: 'b', parentId: 'a' }, { id: 'c', parentId: 'b' }]
    expect(nextParentId(trail, 'out')).toBe('a')
  })

  it('outdenting from the top level stays at the top level', () => {
    const trail: TrailEntry[] = [{ id: 'a' }]
    expect(nextParentId(trail, 'out')).toBeUndefined()
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && pnpm test --run src/lib/quickTask/indent.test.ts`
Expected: FAIL — cannot resolve `./indent`.

- [ ] **Step 3: Write the minimal implementation**

```ts
// web/src/lib/quickTask/indent.ts

/** A task created during the current quick-add session. */
export interface TrailEntry {
  id: string
  parentId?: string
}

/**
 * Which parent the next quick-added task should get.
 * 'in'  — nest under the most recently created task.
 * 'out' — climb one level from the most recently created task.
 */
export function nextParentId(trail: TrailEntry[], direction: 'in' | 'out'): string | undefined {
  const last = trail[trail.length - 1]
  if (!last) return undefined
  if (direction === 'in') return last.id
  const parent = trail.find(entry => entry.id === last.parentId)
  return parent?.parentId
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && pnpm test --run src/lib/quickTask/indent.test.ts`
Expected: PASS (5 tests).

- [ ] **Step 5: Wire the keys into `QuickAddRow`**

Add state and the handler:

```ts
  const [trail, setTrail] = useState<TrailEntry[]>([])
  const [parentId, setParentId] = useState<string | undefined>(undefined)
```

In the input's `onKeyDown`:

```tsx
            if (e.altKey && e.key === 'ArrowRight') {
              e.preventDefault()
              setParentId(nextParentId(trail, 'in'))
            }
            if (e.altKey && e.key === 'ArrowLeft') {
              e.preventDefault()
              setParentId(nextParentId(trail, 'out'))
            }
```

Pass `parentId` into `buildQuickTask`, and after a successful create push the new task onto the trail. `onCreate` must therefore return the created task: change its type to `(request: CreateTaskRequest) => Promise<Task>` and have `handleQuickCreate` in `Tasks.tsx` return `created`.

Show the current nesting depth next to the input so the state is visible — otherwise nobody can tell what `Enter` will do:

```tsx
        {parentId && <span className="shrink-0 text-xs text-zinc-500">↳ {t('quickAdd.subtask')}</span>}
```

- [ ] **Step 6: Verify**

Run: `cd web && pnpm typecheck && pnpm test --run && pnpm build`

By hand: type a parent, Enter, `Alt+→`, type two children, Enter each — both appear nested. `Alt+←` returns to the top level. `Tab` still moves focus out of the input.

- [ ] **Step 7: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/lib/quickTask/indent.ts web/src/lib/quickTask/indent.test.ts web/src/components/QuickAdd
git commit -m "feat(tasks): build a task tree from the quick-add row with Alt+arrows"
```

---

### Task 7: `POST /tasks/batch` with partial success

**Files:**
- Modify: `api-go/internal/tasks/types.go` (add `BatchCreateRequest`, `BatchCreateResponse`, `BatchRowError`)
- Modify: `api-go/internal/tasks/handlers.go` (add `BatchCreateHandler`, extract `insertTask`)
- Create: `api-go/internal/tasks/batch_test.go`
- Modify: `api-go/cmd/api/main.go` (register the route)
- Modify: `web/src/api/tasks.ts` (add `createTasksBatch`)

**Interfaces:**
- Consumes: the existing `CreateTaskRequest` and `validateTaskMutation`.
- Produces: `POST /api/tasks/batch` and `createTasksBatch(requests: CreateTaskRequest[]): Promise<BatchCreateResponse>`.

**Partial success, not all-or-nothing.** One bad row out of twenty must not discard the other nineteen — valid rows are created, invalid rows come back as errors carrying their index.

- [ ] **Step 1: Add the types**

```go
// BatchCreateRequest creates many tasks in one round-trip.
type BatchCreateRequest struct {
	Tasks []CreateTaskRequest `json:"tasks"`
}

// BatchRowError reports a single row that could not be created.
// Index refers to the position in the submitted Tasks slice.
type BatchRowError struct {
	Index   int    `json:"index"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BatchCreateResponse carries the tasks that were created plus the rows
// that failed. Rows are independent: a bad row does not discard good ones.
type BatchCreateResponse struct {
	Tasks  []Task          `json:"tasks"`
	Errors []BatchRowError `json:"errors"`
}

// MaxBatchTasks caps one batch request. Guards against an accidental
// paste of a very large document.
const MaxBatchTasks = 100
```

- [ ] **Step 2: Write the failing test for row validation**

```go
// api-go/internal/tasks/batch_test.go
package tasks

import "testing"

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestValidateBatchRow(t *testing.T) {
	tests := []struct {
		name     string
		req      CreateTaskRequest
		wantCode string
	}{
		{"valid minimal", CreateTaskRequest{Title: "купить хлеб"}, ""},
		{"empty title", CreateTaskRequest{Title: ""}, "MISSING_TITLE"},
		{"whitespace title", CreateTaskRequest{Title: "   "}, "MISSING_TITLE"},
		{"bad due date", CreateTaskRequest{Title: "x", DueDate: strPtr("28-07-2026")}, "INVALID_DUE_DATE"},
		{"good due date", CreateTaskRequest{Title: "x", DueDate: strPtr("2026-07-28T21:00:00Z")}, ""},
		{"priority out of range", CreateTaskRequest{Title: "x", Priority: intPtr(9)}, "INVALID_PRIORITY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _ := validateBatchRow(tt.req)
			if code != tt.wantCode {
				t.Errorf("validateBatchRow() code = %q, want %q", code, tt.wantCode)
			}
		})
	}
}

func TestBatchSizeLimit(t *testing.T) {
	if MaxBatchTasks != 100 {
		t.Errorf("MaxBatchTasks = %d, want 100", MaxBatchTasks)
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd api-go && go test ./internal/tasks/ -run TestValidateBatchRow -v`
Expected: FAIL — `validateBatchRow` undefined.

- [ ] **Step 4: Implement `validateBatchRow`**

In `api-go/internal/tasks/handlers.go`, reusing the existing `validateTaskMutation`:

```go
// validateBatchRow checks one row of a batch create without touching the DB.
// Returns ("", "") when the row is acceptable.
func validateBatchRow(req CreateTaskRequest) (string, string) {
	if strings.TrimSpace(req.Title) == "" {
		return "MISSING_TITLE", "Title is required"
	}
	if req.DueDate != nil && *req.DueDate != "" {
		if _, err := time.Parse(time.RFC3339, *req.DueDate); err != nil {
			return "INVALID_DUE_DATE", "Invalid due date format"
		}
	}
	return validateTaskMutation(req.Status, req.Priority, req.Category)
}
```

Add `"strings"` to the imports if it is not already there. Confirm `validateTaskMutation` returns `INVALID_PRIORITY` for an out-of-range priority; if it uses a different code, update the test to match the real code rather than changing the handler.

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd api-go && go test ./internal/tasks/ -v`
Expected: PASS.

- [ ] **Step 6: Implement the handler**

```go
// BatchCreateHandler creates many tasks in one request. Rows are independent:
// valid rows are created and invalid rows are returned with their index, so a
// single bad line does not discard a whole pasted list.
func BatchCreateHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	var req BatchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if len(req.Tasks) == 0 {
		util.RespondError(w, http.StatusBadRequest, "EMPTY_BATCH", "No tasks provided")
		return
	}
	if len(req.Tasks) > MaxBatchTasks {
		util.RespondError(w, http.StatusBadRequest, "BATCH_TOO_LARGE", "Too many tasks in one request")
		return
	}

	resp := BatchCreateResponse{Tasks: []Task{}, Errors: []BatchRowError{}}
	for i, row := range req.Tasks {
		if code, msg := validateBatchRow(row); code != "" {
			resp.Errors = append(resp.Errors, BatchRowError{Index: i, Code: code, Message: msg})
			continue
		}
		task, err := insertTask(r.Context(), userID, row)
		if err != nil {
			resp.Errors = append(resp.Errors, BatchRowError{Index: i, Code: "CREATE_FAILED", Message: "Could not create task"})
			continue
		}
		resp.Tasks = append(resp.Tasks, *task)
	}

	util.RespondJSON(w, http.StatusCreated, resp)
}
```

Extract the INSERT from the existing `CreateHandler` into `insertTask(ctx context.Context, userID string, req CreateTaskRequest) (*Task, error)` and make `CreateHandler` call it, so both paths share one query and one set of defaults. Do not duplicate the SQL.

- [ ] **Step 7: Register the route**

In `api-go/cmd/api/main.go`, next to the existing `/tasks` routes:

```go
			r.Post("/batch", tasks.BatchCreateHandler)
```

- [ ] **Step 8: Add the frontend call**

In `web/src/api/tasks.ts`:

```ts
export interface BatchRowError {
  index: number
  code: string
  message: string
}

export interface BatchCreateResponse {
  tasks: Task[]
  errors: BatchRowError[]
}

/** Create many tasks in one round-trip. Rows fail independently. */
export async function createTasksBatch(tasks: CreateTaskRequest[]): Promise<BatchCreateResponse> {
  return api.post<BatchCreateResponse>('/tasks/batch', { tasks })
}
```

- [ ] **Step 9: Verify**

Run: `cd api-go && go build ./... && go test ./...` then `cd web && pnpm typecheck && pnpm test --run && pnpm build`

- [ ] **Step 10: Commit** *(only if Denis has asked for commits)*

```bash
git add api-go/internal/tasks web/src/api/tasks.ts api-go/cmd/api/main.go
git commit -m "feat(tasks): POST /tasks/batch with per-row partial success"
```

---

### Task 8: One-click close with undo

**Files:**
- Modify: `web/src/pages/Tasks/Tasks.tsx`
- Modify: `web/src/i18n/locales/{en,ru}/tasks.json`

**Interfaces:**
- Consumes: `updateTask` from `web/src/api/tasks`; the existing toast component at `web/src/components/ui/Toast.tsx` (already used by Pomodoro — follow its API rather than inventing one).

- [ ] **Step 1: Add the toggle handler**

```ts
  async function toggleDone(task: Task) {
    const previous = task.status
    const next = task.status === 'DONE' ? 'TODO' : 'DONE'
    // Optimistic: the list must not wait on the network for a checkbox.
    setTasks(prev => prev.map(t => (t.id === task.id ? { ...t, status: next } : t)))
    try {
      await updateTask(task.id, { status: next })
      showToast(t('toast.closed', { title: task.title }), {
        action: { label: t('toast.undo'), onClick: () => void toggleDoneBack(task.id, previous) },
      })
    } catch {
      setTasks(prev => prev.map(t => (t.id === task.id ? { ...t, status: previous } : t)))
    }
  }

  async function toggleDoneBack(id: string, status: TaskStatus) {
    setTasks(prev => prev.map(t => (t.id === id ? { ...t, status } : t)))
    await updateTask(id, { status })
  }
```

Read `web/src/components/ui/Toast.tsx` first and match its real signature — the `showToast(...)` call above is illustrative, not a promise about its API.

- [ ] **Step 2: Make the status circle a real button**

The list already renders `CheckCircle` / `Circle` icons. Wrap the icon in a `<button type="button" onClick={() => void toggleDone(task)} aria-label={...}>` so it is keyboard-reachable and Space activates it.

- [ ] **Step 3: Add the strings**

`en`: `"toast": { "closed": "Closed \"{{title}}\"", "undo": "Undo" }`
`ru`: `"toast": { "closed": "Закрыто «{{title}}»", "undo": "Вернуть" }`

- [ ] **Step 4: Verify**

Run: `cd web && pnpm typecheck && pnpm test --run && pnpm build`
By hand: one click closes, the toast appears, Undo restores; Space on the focused checkbox works.

- [ ] **Step 5: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/pages/Tasks/Tasks.tsx web/src/i18n/locales/en/tasks.json web/src/i18n/locales/ru/tasks.json
git commit -m "feat(tasks): close a task in one click, with undo"
```

---

### Task 9: Multi-select and bulk actions

**Files:**
- Create: `web/src/lib/quickTask/selectRange.ts`
- Create: `web/src/lib/quickTask/selectRange.test.ts`
- Modify: `web/src/pages/Tasks/Tasks.tsx`

**Interfaces:**
- Produces: `selectRange(ids: string[], anchorId: string, targetId: string): string[]`.

- [ ] **Step 1: Write the failing test**

```ts
// web/src/lib/quickTask/selectRange.test.ts
import { describe, it, expect } from 'vitest'
import { selectRange } from './selectRange'

const ids = ['a', 'b', 'c', 'd', 'e']

describe('selectRange', () => {
  it('selects forwards inclusively', () => {
    expect(selectRange(ids, 'b', 'd')).toEqual(['b', 'c', 'd'])
  })

  it('selects backwards inclusively', () => {
    expect(selectRange(ids, 'd', 'b')).toEqual(['b', 'c', 'd'])
  })

  it('selects a single item when anchor and target match', () => {
    expect(selectRange(ids, 'c', 'c')).toEqual(['c'])
  })

  it('returns just the target when the anchor is unknown', () => {
    expect(selectRange(ids, 'zzz', 'c')).toEqual(['c'])
  })

  it('returns an empty selection when the target is unknown', () => {
    expect(selectRange(ids, 'a', 'zzz')).toEqual([])
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && pnpm test --run src/lib/quickTask/selectRange.test.ts`
Expected: FAIL — cannot resolve `./selectRange`.

- [ ] **Step 3: Write the minimal implementation**

```ts
// web/src/lib/quickTask/selectRange.ts

/**
 * Ids between anchor and target inclusive, in list order.
 * `ids` must be in the order the list is rendered, so Shift+click
 * selects what the eye sees rather than what the data happens to hold.
 */
export function selectRange(ids: string[], anchorId: string, targetId: string): string[] {
  const target = ids.indexOf(targetId)
  if (target === -1) return []
  const anchor = ids.indexOf(anchorId)
  if (anchor === -1) return [targetId]
  const [from, to] = anchor <= target ? [anchor, target] : [target, anchor]
  return ids.slice(from, to + 1)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && pnpm test --run src/lib/quickTask/selectRange.test.ts`
Expected: PASS (5 tests).

- [ ] **Step 5: Wire selection into the list**

Add `const [selected, setSelected] = useState<Set<string>>(new Set())` and an anchor ref. Shift+click on a row calls `selectRange(filteredTasks.map(t => t.id), anchor, task.id)`. When `selected.size > 0`, show a bar above the list with **Закрыть** and **Перенести на завтра**, both applying to every selected id, and a count. "Перенести на завтра" reuses `buildQuickTask`'s local-day logic — extract `startOfLocalDay` into its own exported helper in `web/src/lib/quickTask/localDay.ts` and import it in both places rather than duplicating the arithmetic.

- [ ] **Step 6: Verify**

Run: `cd web && pnpm typecheck && pnpm test --run && pnpm build`

- [ ] **Step 7: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/lib/quickTask web/src/pages/Tasks/Tasks.tsx
git commit -m "feat(tasks): multi-select with bulk close and push-to-tomorrow"
```

---

### Task 10: Settings section

**Files:**
- Modify: `web/src/pages/Settings/Settings.tsx`
- Modify: `web/src/i18n/locales/{en,ru}/settings.json`

**Interfaces:**
- Consumes: `QUICK_TASK_DEFAULTS`, `resolveQuickTaskSettings`, and `updateSettings` from `AuthContext`.

- [ ] **Step 1: Add the section**

A "Быстрое создание задач" block with: default due (`today` / `tomorrow` / `none`), default priority (1–5), default estimate in minutes (empty = no estimate), and an **off-by-default** switch labelled in plain language — «Наследовать активные фильтры страницы», not "apply current filters". Persist through the existing `updateSettings({ quick_task: {...} })`; the context already merges over previous settings (`AuthContext.tsx:204-210`), so partial writes are safe.

Follow the hybrid auto-save pattern already used on this page (shipped in v0.4.9) rather than adding a Save button.

- [ ] **Step 2: Verify**

Run: `cd web && pnpm typecheck && pnpm test --run && pnpm build`
By hand: change the default priority, reload, create a quick task — it uses the new default. Set the estimate field empty — the created task has no estimate.

- [ ] **Step 3: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/pages/Settings/Settings.tsx web/src/i18n/locales/en/settings.json web/src/i18n/locales/ru/settings.json
git commit -m "feat(settings): quick-task defaults"
```

---

### Task 11: Global Ctrl+K capture

**Files:**
- Create: `web/src/components/QuickAdd/QuickAddModal.tsx`
- Modify: `web/src/App.tsx` (mount the listener once, above the router)

**Interfaces:**
- Consumes: `QuickAddRow` (Task 4), `createTask` from `web/src/api/tasks`, `resolveQuickTaskSettings` for the configured binding.

- [ ] **Step 1: Write the modal**

It renders the same `<QuickAddRow>` plus a running list of what was created this session. `Esc` closes it. Nothing about the row is re-implemented — one component, two mount points.

- [ ] **Step 2: Mount the global listener**

In `App.tsx`, a `useEffect` binding `keydown` on `window` that opens the modal on the configured `global_capture` binding (default `Ctrl+K`) and calls `preventDefault()`. Skip the shortcut when the event target is already a text input, so it cannot steal typing.

- [ ] **Step 3: Verify**

Run: `cd web && pnpm typecheck && pnpm test --run && pnpm build`
By hand: `Ctrl+K` from `/calendar` opens the overlay, a task is created, `Esc` closes, and the task appears on `/tasks` after navigation.

- [ ] **Step 4: Commit** *(only if Denis has asked for commits)*

```bash
git add web/src/components/QuickAdd web/src/App.tsx
git commit -m "feat(tasks): global Ctrl+K quick capture"
```

---

## Self-review notes

**Spec coverage:** §4.1 → Tasks 4, 6. §4.2 → Task 5. §4.3 → Task 11. §4.4 → Tasks 2, 3.
§4.5 → Tasks 2, 10. §4.6 → Tasks 8, 9. §5 → Task 7. §5.1 (T1) → Task 1. §6 → tests inside
each task.

**Deliberately deferred, from spec §8:** ordering of tasks inside a priority group; level-2
"place / links / sources" fields (they need a migration); a React component-test harness.
None of these block Tasks 1–11.

**Known risk:** Tasks 4–6, 8–11 have no automated UI coverage, because the project has no
React component-test harness. Every one of those tasks therefore carries an explicit manual
check step. That is a stopgap, not a substitute.
