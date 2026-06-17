# Phase 5 "Show Hints" — Pure Cores (5a) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the three pure, TDD'd cores the "show hints" feature needs — style preference, route→anchor content, and viewport-clamped bubble placement — with no UI yet.

**Architecture:** Three independent pure modules in `web/src/lib/onboarding/`, each mirroring an existing sibling (`onboardingFlag.ts`, `helpContent.ts`). No React, no DOM dependency beyond plain rect math. UI shells (`HintsLayer`, `HintsBubbles`, Settings control, "Show hints" button) come in later iterations and consume these cores.

**Tech Stack:** TypeScript (strict), vitest (jsdom), localStorage. Tests are pure functions — repo has no `@testing-library/react`.

**Source of truth:** `docs/superpowers/specs/2026-06-17-show-hints-reveal-design.md`.

---

### Task 1: `hintStyle.ts` — style preference (localStorage v1)

**Files:**
- Create: `web/src/lib/onboarding/hintStyle.ts`
- Test: `web/src/lib/onboarding/hintStyle.test.ts`

- [ ] **Step 1: Write the failing test** (`hintStyle.test.ts`)

```ts
import { describe, it, expect } from 'vitest'
import { parseHintStyle, getHintStyle, setHintStyle } from './hintStyle'

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

describe('parseHintStyle', () => {
  it('accepts the three valid styles', () => {
    expect(parseHintStyle('bubbles')).toBe('bubbles')
    expect(parseHintStyle('walkthrough')).toBe('walkthrough')
    expect(parseHintStyle('markers')).toBe('markers')
  })
  it('defaults to bubbles for null/unknown/garbage', () => {
    expect(parseHintStyle(null)).toBe('bubbles')
    expect(parseHintStyle('')).toBe('bubbles')
    expect(parseHintStyle('BUBBLES')).toBe('bubbles')
    expect(parseHintStyle('tour')).toBe('bubbles')
  })
})

describe('getHintStyle / setHintStyle', () => {
  it('defaults to bubbles when storage is empty', () => {
    expect(getHintStyle(fakeStorage())).toBe('bubbles')
  })
  it('round-trips a written style', () => {
    const s = fakeStorage()
    setHintStyle('walkthrough', s)
    expect(getHintStyle(s)).toBe('walkthrough')
  })
  it('never throws when storage access throws', () => {
    const throwing = {
      getItem: () => { throw new Error('blocked') },
      setItem: () => { throw new Error('quota') },
      removeItem: () => {}, clear: () => {}, key: () => null, length: 0,
    } as unknown as Storage
    expect(getHintStyle(throwing)).toBe('bubbles')
    expect(() => setHintStyle('markers', throwing)).not.toThrow()
  })
})
```

- [ ] **Step 2: Run to verify it fails**

Run: `corepack pnpm test -- hintStyle` — Expected: FAIL ("Failed to resolve import './hintStyle'").

- [ ] **Step 3: Implement** (`hintStyle.ts`)

```ts
export type HintStyle = 'bubbles' | 'walkthrough' | 'markers'

const HINT_STYLE_KEY = 'neuroboost-hints-style'
const VALID: readonly HintStyle[] = ['bubbles', 'walkthrough', 'markers']
const DEFAULT_HINT_STYLE: HintStyle = 'bubbles'

export function parseHintStyle(raw: string | null): HintStyle {
  return VALID.includes(raw as HintStyle) ? (raw as HintStyle) : DEFAULT_HINT_STYLE
}

export function getHintStyle(storage: Storage = localStorage): HintStyle {
  try {
    return parseHintStyle(storage.getItem(HINT_STYLE_KEY))
  } catch {
    return DEFAULT_HINT_STYLE
  }
}

export function setHintStyle(style: HintStyle, storage: Storage = localStorage): void {
  try {
    storage.setItem(HINT_STYLE_KEY, style)
  } catch {
    // preference is non-critical; ignore quota/availability errors
  }
}
```

- [ ] **Step 4: Run to verify it passes** — Run: `corepack pnpm test -- hintStyle` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/onboarding/hintStyle.ts web/src/lib/onboarding/hintStyle.test.ts
git commit -m "feat(hints): hintStyle core (localStorage style preference)"
```

---

### Task 2: `hintsContent.ts` — route → ordered anchors

**Files:**
- Create: `web/src/lib/onboarding/hintsContent.ts`
- Test: `web/src/lib/onboarding/hintsContent.test.ts`

- [ ] **Step 1: Write the failing test** (`hintsContent.test.ts`)

```ts
import { describe, it, expect } from 'vitest'
import { hintsForRoute } from './hintsContent'

describe('hintsForRoute', () => {
  it('returns ordered anchors for a workflow route, with derived i18n keys', () => {
    expect(hintsForRoute('/calendar')).toEqual([
      { anchor: 'calendar.newEvent', titleKey: 'hints.calendar.newEvent.title', bodyKey: 'hints.calendar.newEvent.body' },
      { anchor: 'calendar.grid', titleKey: 'hints.calendar.grid.title', bodyKey: 'hints.calendar.grid.body' },
      { anchor: 'calendar.taskSidebar', titleKey: 'hints.calendar.taskSidebar.title', bodyKey: 'hints.calendar.taskSidebar.body' },
    ])
  })
  it('ignores sub-paths and resolves by the first segment', () => {
    expect(hintsForRoute('/tasks/123').map(h => h.anchor)).toEqual(['tasks.new', 'tasks.schedule', 'tasks.complete'])
  })
  it('returns [] for the root and for routes without hints', () => {
    expect(hintsForRoute('/')).toEqual([])
    expect(hintsForRoute('')).toEqual([])
    expect(hintsForRoute('/settings')).toEqual([])
    expect(hintsForRoute('/nope')).toEqual([])
  })
  it('derives every anchor key under the hints namespace with .title/.body', () => {
    for (const path of ['/home', '/calendar', '/tasks', '/planning']) {
      for (const h of hintsForRoute(path)) {
        expect(h.titleKey).toBe(`hints.${h.anchor}.title`)
        expect(h.bodyKey).toBe(`hints.${h.anchor}.body`)
      }
    }
  })
})
```

- [ ] **Step 2: Run to verify it fails** — Run: `corepack pnpm test -- hintsContent` — Expected: FAIL (import unresolved).

- [ ] **Step 3: Implement** (`hintsContent.ts`)

```ts
export interface HintAnchor {
  anchor: string
  titleKey: string
  bodyKey: string
}

// `anchor` is the value of a `data-hint="<anchor>"` attribute placed on the real
// element. Order = reveal order. Extend by adding routes/anchors + matching i18n
// keys; no other code changes needed.
const ANCHORS_BY_ROUTE: Record<string, string[]> = {
  home: ['home.quickAdd', 'home.schedule', 'home.tasks'],
  calendar: ['calendar.newEvent', 'calendar.grid', 'calendar.taskSidebar'],
  tasks: ['tasks.new', 'tasks.schedule', 'tasks.complete'],
  planning: ['planning.unscheduled', 'planning.day'],
}

export function hintsForRoute(pathname: string): HintAnchor[] {
  const segment = (pathname || '').split('/').filter(Boolean)[0]
  const anchors = (segment && ANCHORS_BY_ROUTE[segment]) || []
  return anchors.map((anchor) => ({
    anchor,
    titleKey: `hints.${anchor}.title`,
    bodyKey: `hints.${anchor}.body`,
  }))
}
```

- [ ] **Step 4: Run to verify it passes** — Run: `corepack pnpm test -- hintsContent` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/onboarding/hintsContent.ts web/src/lib/onboarding/hintsContent.test.ts
git commit -m "feat(hints): hintsContent core (route to anchor list)"
```

---

### Task 3: `bubblePlacement.ts` — viewport-clamped placement

**Files:**
- Create: `web/src/lib/onboarding/bubblePlacement.ts`
- Test: `web/src/lib/onboarding/bubblePlacement.test.ts`

Placement rule: prefer `bottom`, then `top`, then `right`, then `left` (first side with enough room for `bubble + gap`); if none fit, fall back to `bottom`. Then clamp the bubble fully inside the viewport so it is never cut off; an oversized bubble pins to `0`.

- [ ] **Step 1: Write the failing test** (`bubblePlacement.test.ts`)

```ts
import { describe, it, expect } from 'vitest'
import { placeBubble } from './bubblePlacement'

const viewport = { width: 1000, height: 800 }
const bubble = { width: 200, height: 100 }

describe('placeBubble', () => {
  it('places below a top-area anchor, horizontally centered', () => {
    const r = placeBubble({ top: 100, left: 400, width: 40, height: 40 }, bubble, viewport)
    expect(r.placement).toBe('bottom')
    expect(r.top).toBe(148)            // 100 + 40 + 8
    expect(r.left).toBe(320)           // (400 + 20) - 100
  })

  it('flips above when there is no room below', () => {
    const r = placeBubble({ top: 740, left: 400, width: 40, height: 40 }, bubble, viewport)
    expect(r.placement).toBe('top')
    expect(r.top).toBe(632)            // 740 - 100 - 8
  })

  it('uses the right side when neither below nor above fit', () => {
    // tall anchor: roomAbove=100 (<108) and roomBelow=80 (<108), so it flips to the side
    const r = placeBubble({ top: 100, left: 100, width: 40, height: 620 }, bubble, viewport)
    expect(r.placement).toBe('right')
    expect(r.left).toBe(148)           // 100 + 40 + 8
  })

  it('clamps a centered bubble to the left viewport edge', () => {
    const r = placeBubble({ top: 100, left: 10, width: 40, height: 40 }, bubble, viewport)
    expect(r.left).toBe(0)             // centered would be -70 → clamped
  })

  it('clamps a centered bubble to the right viewport edge', () => {
    const r = placeBubble({ top: 100, left: 960, width: 40, height: 40 }, bubble, viewport)
    expect(r.left).toBe(800)           // viewport.width - bubble.width
  })

  it('pins an oversized bubble to 0,0 instead of going negative', () => {
    const big = { width: 1200, height: 900 }
    const r = placeBubble({ top: 100, left: 400, width: 40, height: 40 }, big, viewport)
    expect(r.left).toBe(0)
    expect(r.top).toBe(0)
  })
})
```

- [ ] **Step 2: Run to verify it fails** — Run: `corepack pnpm test -- bubblePlacement` — Expected: FAIL (import unresolved).

- [ ] **Step 3: Implement** (`bubblePlacement.ts`)

```ts
export interface Rect { top: number; left: number; width: number; height: number }
export interface Size { width: number; height: number }
export interface Viewport { width: number; height: number }
export type Placement = 'top' | 'bottom' | 'left' | 'right'
export interface Placed { top: number; left: number; placement: Placement }

function clamp(value: number, max: number): number {
  // pins to 0 when max < 0 (bubble larger than viewport)
  return Math.max(0, Math.min(value, max))
}

export function placeBubble(anchor: Rect, bubble: Size, viewport: Viewport, gap = 8): Placed {
  const roomBelow = viewport.height - (anchor.top + anchor.height)
  const roomAbove = anchor.top
  const roomRight = viewport.width - (anchor.left + anchor.width)
  const roomLeft = anchor.left
  const needV = bubble.height + gap
  const needH = bubble.width + gap

  let placement: Placement
  if (roomBelow >= needV) placement = 'bottom'
  else if (roomAbove >= needV) placement = 'top'
  else if (roomRight >= needH) placement = 'right'
  else if (roomLeft >= needH) placement = 'left'
  else placement = 'bottom'

  const centerX = anchor.left + anchor.width / 2 - bubble.width / 2
  const centerY = anchor.top + anchor.height / 2 - bubble.height / 2

  let top: number
  let left: number
  switch (placement) {
    case 'bottom': top = anchor.top + anchor.height + gap; left = centerX; break
    case 'top': top = anchor.top - bubble.height - gap; left = centerX; break
    case 'right': left = anchor.left + anchor.width + gap; top = centerY; break
    case 'left': left = anchor.left - bubble.width - gap; top = centerY; break
  }

  return {
    placement,
    left: clamp(left, viewport.width - bubble.width),
    top: clamp(top, viewport.height - bubble.height),
  }
}
```

- [ ] **Step 4: Run to verify it passes** — Run: `corepack pnpm test -- bubblePlacement` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/onboarding/bubblePlacement.ts web/src/lib/onboarding/bubblePlacement.test.ts
git commit -m "feat(hints): bubblePlacement core (viewport-clamped positioning)"
```

---

### Task 4: Full gate + merge

- [ ] **Step 1:** `corepack pnpm typecheck` — Expected: clean.
- [ ] **Step 2:** `corepack pnpm test` — Expected: all suites pass (existing 158 + the new core tests).
- [ ] **Step 3:** `corepack pnpm build` — Expected: built OK.
- [ ] **Step 4:** Add the plan doc, fast-forward merge the branch to `develop`, remove the worktree.

---

## What this plan does NOT cover (later iterations)

- `walkthroughStep.ts` core → ships with **5b** (walkthrough).
- UI shells (`HintsLayer`, `HintsBubbles`, `HintsWalkthrough`, `HintsMarkers`), the `OnboardingContext` extension, the "Show hints" button in the "?" panel, the Settings radio control, `data-hint` attributes on real elements, and all `hints.*` / `settings.hints.*` i18n copy → ship with **5a-UI / 5b / 5c**.
