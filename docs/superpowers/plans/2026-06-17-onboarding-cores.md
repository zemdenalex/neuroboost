<!-- паспорт: тип=план | статус=архив | строк=317 | ~токенов=2984 | обновлён=по git -->

# Onboarding Pure Cores — Implementation Plan (Phase 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the three pure, fully-tested logic cores that the onboarding + contextual-help feature is built on, with zero React/UI yet.

**Architecture:** Per the design spec (`docs/superpowers/specs/2026-06-17-onboarding-and-contextual-help-design.md`), all branching logic lives in pure functions so it can be unit-tested with vitest (this codebase has no `@testing-library/react`; web tests are pure-fn on jsdom). The React shells and `OnboardingContext` consume these cores and are planned separately once these land.

**Tech Stack:** TypeScript (strict), vitest. Files under `web/src/lib/onboarding/`.

**Scope note:** This plan covers spec Phase 1 (the cores) only. Remaining phases — WelcomeCard + GuidedChecklist + OnboardingContext (2), tap-Schedule path (3), HelpPanel (4), HintsLayer (5), empty states (6) — each get their own plan when reached, because they require reading the real components (header, pages, router, i18n index) to write accurate, placeholder-free code.

---

### Task 1: `onboardingFlag` — first-run flags over localStorage

**Files:**
- Create: `web/src/lib/onboarding/onboardingFlag.ts`
- Test: `web/src/lib/onboarding/onboardingFlag.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect } from 'vitest'
import { getOnboardingFlags, setWelcomeSeen, setChecklistDismissed } from './onboardingFlag'

function fakeStorage(initial: Record<string, string> = {}): Storage {
  const map = new Map(Object.entries(initial))
  return {
    getItem: (k) => (map.has(k) ? (map.get(k) as string) : null),
    setItem: (k, v) => { map.set(k, String(v)) },
    removeItem: (k) => { map.delete(k) },
    clear: () => map.clear(),
    key: (i) => Array.from(map.keys())[i] ?? null,
    get length() { return map.size },
  } as Storage
}

describe('onboardingFlag', () => {
  it('defaults to all-false when storage is empty', () => {
    expect(getOnboardingFlags(fakeStorage())).toEqual({ welcomeSeen: false, checklistDismissed: false })
  })

  it('reads true flags written by the setters', () => {
    const s = fakeStorage()
    setWelcomeSeen(s)
    setChecklistDismissed(s)
    expect(getOnboardingFlags(s)).toEqual({ welcomeSeen: true, checklistDismissed: true })
  })

  it('treats malformed values as false', () => {
    const s = fakeStorage({
      'neuroboost-onboarding-welcome-seen': 'yes',
      'neuroboost-onboarding-checklist-dismissed': '1',
    })
    expect(getOnboardingFlags(s)).toEqual({ welcomeSeen: false, checklistDismissed: false })
  })

  it('never throws when storage access throws', () => {
    const throwing = {
      getItem: () => { throw new Error('blocked') },
      setItem: () => { throw new Error('quota') },
      removeItem: () => {}, clear: () => {}, key: () => null, length: 0,
    } as unknown as Storage
    expect(() => getOnboardingFlags(throwing)).not.toThrow()
    expect(getOnboardingFlags(throwing)).toEqual({ welcomeSeen: false, checklistDismissed: false })
    expect(() => setWelcomeSeen(throwing)).not.toThrow()
    expect(() => setChecklistDismissed(throwing)).not.toThrow()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm vitest run src/lib/onboarding/onboardingFlag.test.ts`
Expected: FAIL — cannot find module `./onboardingFlag`.

- [ ] **Step 3: Write minimal implementation**

```ts
const WELCOME_SEEN_KEY = 'neuroboost-onboarding-welcome-seen'
const CHECKLIST_DISMISSED_KEY = 'neuroboost-onboarding-checklist-dismissed'

export interface OnboardingFlags {
  welcomeSeen: boolean
  checklistDismissed: boolean
}

function readBool(storage: Storage, key: string): boolean {
  try {
    return storage.getItem(key) === 'true'
  } catch {
    return false
  }
}

function writeTrue(storage: Storage, key: string): void {
  try {
    storage.setItem(key, 'true')
  } catch {
    // flags are non-critical; ignore quota/availability errors
  }
}

export function getOnboardingFlags(storage: Storage = localStorage): OnboardingFlags {
  return {
    welcomeSeen: readBool(storage, WELCOME_SEEN_KEY),
    checklistDismissed: readBool(storage, CHECKLIST_DISMISSED_KEY),
  }
}

export function setWelcomeSeen(storage: Storage = localStorage): void {
  writeTrue(storage, WELCOME_SEEN_KEY)
}

export function setChecklistDismissed(storage: Storage = localStorage): void {
  writeTrue(storage, CHECKLIST_DISMISSED_KEY)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm vitest run src/lib/onboarding/onboardingFlag.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit** — batched with Tasks 2 & 3 at the end (single phase commit).

---

### Task 2: `checklistProgress` — derive step state from counts

**Files:**
- Create: `web/src/lib/onboarding/checklistProgress.ts`
- Test: `web/src/lib/onboarding/checklistProgress.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect } from 'vitest'
import { computeChecklistProgress } from './checklistProgress'

describe('computeChecklistProgress', () => {
  it('all steps incomplete when every count is zero', () => {
    const p = computeChecklistProgress({ taskCount: 0, eventCount: 0, completedTaskCount: 0, reflectionCount: 0 })
    expect(p.steps).toEqual({ createTask: false, scheduleTask: false, completeAndReflect: false })
    expect(p.currentStep).toBe('createTask')
    expect(p.completedCount).toBe(0)
  })

  it('createTask done, next step is scheduleTask', () => {
    const p = computeChecklistProgress({ taskCount: 1, eventCount: 0, completedTaskCount: 0, reflectionCount: 0 })
    expect(p.steps.createTask).toBe(true)
    expect(p.currentStep).toBe('scheduleTask')
    expect(p.completedCount).toBe(1)
  })

  it('completeAndReflect satisfied by a reflection alone', () => {
    const p = computeChecklistProgress({ taskCount: 1, eventCount: 1, completedTaskCount: 0, reflectionCount: 2 })
    expect(p.steps.completeAndReflect).toBe(true)
    expect(p.currentStep).toBe('done')
    expect(p.completedCount).toBe(3)
  })

  it('completeAndReflect satisfied by a completed task alone', () => {
    const p = computeChecklistProgress({ taskCount: 1, eventCount: 1, completedTaskCount: 1, reflectionCount: 0 })
    expect(p.steps.completeAndReflect).toBe(true)
    expect(p.currentStep).toBe('done')
  })

  it('treats missing/undefined counts as zero', () => {
    const p = computeChecklistProgress({})
    expect(p.currentStep).toBe('createTask')
    expect(p.completedCount).toBe(0)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm vitest run src/lib/onboarding/checklistProgress.test.ts`
Expected: FAIL — cannot find module `./checklistProgress`.

- [ ] **Step 3: Write minimal implementation**

```ts
export interface ChecklistCounts {
  taskCount: number
  eventCount: number
  completedTaskCount: number
  reflectionCount: number
}

export type ChecklistStepId = 'createTask' | 'scheduleTask' | 'completeAndReflect'

export interface ChecklistProgress {
  steps: Record<ChecklistStepId, boolean>
  currentStep: ChecklistStepId | 'done'
  completedCount: number
}

const STEP_ORDER: ChecklistStepId[] = ['createTask', 'scheduleTask', 'completeAndReflect']

export function computeChecklistProgress(counts: Partial<ChecklistCounts>): ChecklistProgress {
  const taskCount = counts.taskCount ?? 0
  const eventCount = counts.eventCount ?? 0
  const completedTaskCount = counts.completedTaskCount ?? 0
  const reflectionCount = counts.reflectionCount ?? 0

  const steps: Record<ChecklistStepId, boolean> = {
    createTask: taskCount >= 1,
    scheduleTask: eventCount >= 1,
    completeAndReflect: completedTaskCount >= 1 || reflectionCount >= 1,
  }
  const currentStep: ChecklistStepId | 'done' = STEP_ORDER.find((s) => !steps[s]) ?? 'done'
  const completedCount = STEP_ORDER.filter((s) => steps[s]).length

  return { steps, currentStep, completedCount }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm vitest run src/lib/onboarding/checklistProgress.test.ts`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit** — batched at the end.

---

### Task 3: `helpContent` — resolve a help i18n key from the current path

**Files:**
- Create: `web/src/lib/onboarding/helpContent.ts`
- Test: `web/src/lib/onboarding/helpContent.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect } from 'vitest'
import { resolveHelpKey } from './helpContent'

describe('resolveHelpKey', () => {
  it('maps a known top-level route to its help key', () => {
    expect(resolveHelpKey('/tasks')).toBe('help.tasks')
    expect(resolveHelpKey('/calendar')).toBe('help.calendar')
  })

  it('ignores trailing segments and query/sub-paths', () => {
    expect(resolveHelpKey('/tasks/123')).toBe('help.tasks')
    expect(resolveHelpKey('/calendar/2026-06-17')).toBe('help.calendar')
  })

  it('falls back to the default key for the root path', () => {
    expect(resolveHelpKey('/')).toBe('help.default')
    expect(resolveHelpKey('')).toBe('help.default')
  })

  it('falls back to the default key for unknown routes', () => {
    expect(resolveHelpKey('/nope')).toBe('help.default')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm vitest run src/lib/onboarding/helpContent.test.ts`
Expected: FAIL — cannot find module `./helpContent`.

- [ ] **Step 3: Write minimal implementation**

```ts
const HELP_KEY_BY_ROUTE: Record<string, string> = {
  home: 'help.home',
  calendar: 'help.calendar',
  tasks: 'help.tasks',
  planning: 'help.planning',
  reflections: 'help.reflections',
  tools: 'help.tools',
  settings: 'help.settings',
  profile: 'help.profile',
}

const DEFAULT_HELP_KEY = 'help.default'

export function resolveHelpKey(pathname: string): string {
  const segment = (pathname || '').split('/').filter(Boolean)[0]
  if (!segment) return DEFAULT_HELP_KEY
  return HELP_KEY_BY_ROUTE[segment] ?? DEFAULT_HELP_KEY
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm vitest run src/lib/onboarding/helpContent.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Gate + Commit (phase)**

```bash
COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm typecheck
COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm test
COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm build
git add web/src/lib/onboarding/onboardingFlag.ts web/src/lib/onboarding/onboardingFlag.test.ts \
        web/src/lib/onboarding/checklistProgress.ts web/src/lib/onboarding/checklistProgress.test.ts \
        web/src/lib/onboarding/helpContent.ts web/src/lib/onboarding/helpContent.test.ts \
        docs/superpowers/plans/2026-06-17-onboarding-cores.md
git commit -m "feat(onboarding): pure cores — flags, checklist progress, help routing"
```
Expected: typecheck/test/build all green; commit created.

---

## Self-Review

- **Spec coverage (Phase 1):** flag get/set → Task 1; checklist step-completion detection → Task 2; route → help-content resolution → Task 3. All three cores named in the spec's architecture are covered. Context + UI are explicitly out of this plan's scope (deferred to later phase plans).
- **Placeholder scan:** none — every step has complete code and an exact command with expected result.
- **Type consistency:** `OnboardingFlags`, `ChecklistCounts`/`ChecklistStepId`/`ChecklistProgress`, and `resolveHelpKey(pathname): string` are each defined once and used consistently. Help keys are returned as `help.<route>` (consumed later under the `onboarding` i18n namespace).
