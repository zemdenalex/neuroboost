# Phase 5 — "Show hints" reveal (element-level contextual hints)

> Design spec. Extends the approved onboarding/help design
> (`docs/superpowers/specs/2026-06-17-onboarding-and-contextual-help-design.md`, option C, sequencing step 5).
> Phases 1–4 + 6 are shipped on `develop`. This spec covers Phase 5 only.

## Goal

An on-demand, **element-level** hints layer that complements the already-shipped **page-level** "?" help panel
(`components/Help/HelpButton.tsx`). Where the "?" panel explains *what a page is for*, hints point at *specific
controls on that page* ("click here to add an event", "drag a task here to schedule it").

Hints are **off by default** and revealed on demand. Three reveal **styles** are user-selectable in Settings;
the default is all-at-once bubbles.

## The three styles (user-selectable)

| Style | Behaviour | Trigger |
|-------|-----------|---------|
| **`bubbles`** (default) | Clicking "Show hints on this page" pops a labelled bubble on every key element at once. One "Got it" / `✕` / `Escape` dismisses all. Non-blocking, scannable, calm. | "Show hints" button in the "?" panel |
| **`walkthrough`** | Clicking "Show hints" starts a guided tour: one element at a time with **Next / Back**, the rest dimmed by a backdrop "spotlight". Scrolls each anchor into view. `Escape` or finishing closes it. | "Show hints" button in the "?" panel |
| **`markers`** | Small persistent `ⓘ` dots sit on key elements at all times; hovering (desktop) or tapping (touch) one reveals that single bubble. No reveal "moment" — always available. | none (dots are always rendered) |

The style is a single preference. Switching style in Settings changes how *all* pages reveal hints.

## Architecture

### Pure cores (TDD — `web/src/lib/onboarding/`, red→green vitest)

These hold every branch of logic. No React, no DOM beyond plain rect math.

- **`hintStyle.ts`**
  - `type HintStyle = 'bubbles' | 'walkthrough' | 'markers'`
  - `parseHintStyle(raw: string | null): HintStyle` — returns the matching style or `'bubbles'` for null/unknown/garbage.
  - `getHintStyle(storage = localStorage): HintStyle` — reads key `neuroboost-hints-style`, parses, never throws.
  - `setHintStyle(style: HintStyle, storage = localStorage): void` — writes the key, swallows quota/availability errors.
  - Mirrors the existing `onboardingFlag.ts` pattern (localStorage v1, consistent with the approved onboarding flag decision).

- **`hintsContent.ts`**
  - `interface HintAnchor { anchor: string; titleKey: string; bodyKey: string }`
  - `hintsForRoute(pathname: string): HintAnchor[]` — pure route→ordered-list. Returns `[]` for routes with no hints.
    `anchor` is the value of a `data-hint="<anchor>"` attribute placed on the real element; `titleKey`/`bodyKey`
    are `onboarding` i18n keys (`hints.<page>.<anchor>.title` / `.body`).
  - Sibling of the existing `helpContent.ts`; same routing idiom.

- **`bubblePlacement.ts`** (the trickiest logic → most tests)
  - `interface Rect { top: number; left: number; width: number; height: number }`
  - `interface Placed { top: number; left: number; placement: 'top' | 'bottom' | 'left' | 'right' }`
  - `placeBubble(anchor: Rect, bubble: { width: number; height: number }, viewport: { width: number; height: number }, gap = 8): Placed`
  - Picks a side with room (prefers `bottom`, then `top`, then `right`, then `left`), then **clamps** the bubble fully
    inside the viewport so it is never cut off. Deterministic, no DOM. Used by all three styles.

- **`walkthroughStep.ts`**
  - `interface StepState { index: number; isFirst: boolean; isLast: boolean }`
  - `clampStep(index: number, total: number): StepState` and `nextStep(current: number, total: number, dir: 1 | -1): StepState`.
    Pure index math; `total === 0` → `index 0, isFirst && isLast`.

### Thin shells (typecheck + `vite build` + self-review; no unit tests — repo has no RTL)

- **`contexts/OnboardingContext.tsx`** (extend) — add to the existing provider (already mounted in the stable `Layout`):
  - `hintStyle: HintStyle` (read once on mount via `getHintStyle`; updated by a `neuroboost-hints-style-change`
    CustomEvent + cross-tab `storage` listener, mirroring how `Layout` listens for `header_variant`).
  - `hintsActive: boolean` — ephemeral; resets to `false` on `useLocation().pathname` change and on reload.
  - `showHints(): void` / `hideHints(): void`.

- **`components/Hints/HintsLayer.tsx`** — dispatcher. Reads `hintStyle`, `hintsActive`, `useLocation()`; resolves
  `hintsForRoute(pathname)`; renders the matching renderer (or nothing). Mounted once in `Layout` beside `OnboardingOverlay`.

- **`components/Hints/HintsBubbles.tsx`** — for each resolved anchor present in the DOM, measure its rect, call
  `placeBubble`, render a fixed-position bubble (`titleKey`/`bodyKey`). A single "Got it" control + `Escape` calls `hideHints()`.
  Re-measures on `resize`/`scroll` (throttled). Skips anchors whose element is absent.

- **`components/Hints/HintsWalkthrough.tsx`** — full-screen dim backdrop with a cut-out/highlight ring around the current
  anchor; a bubble with `Step N of M`, Back, Next (driven by `walkthroughStep`). Scrolls the current anchor into view,
  re-measures with `placeBubble`. `Escape` / finishing calls `hideHints()`. If an anchor is missing, it is skipped.

- **`components/Hints/HintsMarkers.tsx`** — for each resolved anchor in the DOM, render a small `ⓘ` button pinned near
  the anchor (via `placeBubble` with a tiny "bubble"). Hover (desktop) / tap (touch) reveals that anchor's single bubble;
  blur/second-tap/`Escape` hides it. Always rendered while `hintStyle === 'markers'` (independent of `hintsActive`).

- **`components/Help/HelpButton.tsx`** (extend) — add a **"Show hints on this page"** button inside the open panel
  that calls `showHints()` and closes the panel. The button is hidden when **either** `hintStyle === 'markers'` (dots are
  always visible, so there is nothing to trigger) **or** `hintsForRoute(pathname)` is empty (nothing to show).

- **`pages/Settings/Settings.tsx`** (extend) — a radio group (beside the existing layout-variant control) to choose the
  hint style. On change: `setHintStyle(style)` + dispatch `neuroboost-hints-style-change` so the live context updates
  immediately (same instant-apply pattern as `header_variant`).

### `data-hint` anchors (curated, 2–4 per workflow page)

Attributes added to existing elements; copy sourced from the shipped `help.*` blocks. Starter set (extensible):

- **`/home` (Dashboard):** `home.quickAdd` (add event/task action), `home.schedule` (Today's Schedule card), `home.tasks` (Task Summary card).
- **`/calendar`:** `calendar.newEvent` (New / quick-create), `calendar.grid` (the week grid — click a slot to create), `calendar.taskSidebar` (drag a task to schedule).
- **`/tasks`:** `tasks.new` (New Task button), `tasks.schedule` (per-row Schedule button), `tasks.complete` (the done circle).
- **`/planning`:** `planning.unscheduled` (unscheduled list — source of draggable tasks), `planning.day` (a day column — drop target).

Other routes resolve to `[]` (no hints) for v1 and can be filled in later without code changes — only `hintsContent.ts` + i18n grow.

## Data flow

`OnboardingProvider` reads `hintStyle` once on mount and keeps it live via the change/`storage` listeners. `hintsActive`
starts `false` and is forced back to `false` whenever the route changes (hints are per-page). The "?" panel's "Show hints"
button calls `showHints()`. `HintsLayer` renders the active style's component, which resolves anchors from
`hintsForRoute(pathname)`, measures the live DOM, places bubbles with `placeBubble`, and renders. `markers` style renders
its dots whenever selected, regardless of `hintsActive`.

## Error handling

- Cores never throw: `parseHintStyle` defaults to `'bubbles'`; `hintsForRoute` returns `[]` for unknown routes;
  `placeBubble`/`clampStep` are total functions over their numeric inputs.
- A `data-hint` anchor with no matching DOM element is **skipped** (filtered out after `querySelector`), so hints degrade
  gracefully when an element is hidden (e.g. inside a closed mobile drawer) or removed.
- localStorage failures fall back to the default style; hints are non-critical and never block the app.

## Mobile

Bubbles and markers are positioned with `placeBubble` and clamped to the viewport. Anchors not currently in the DOM
(inside closed drawers) are skipped — v1 targets the always-visible main-content elements. Walkthrough scrolls each
anchor into view before measuring. No drawer-opening orchestration in v1 (YAGNI; revisit if testers need drawer hints).

## i18n

Extend `web/src/i18n/locales/{en,ru}/onboarding.json` under a new `hints` block. Every key in **both** locales:

- `hints.showHints` ("Show hints on this page"), `hints.gotIt` ("Got it"), `hints.next`, `hints.back`,
  `hints.step` (`"Step {{n}} of {{total}}"`), `hints.reveal` (markers aria-label, e.g. "Show this hint").
- `hints.<page>.<anchor>.title` / `.body` for every anchor in the starter set above.
- Settings: `settings.hints.title`, `settings.hints.note`, and the three style names
  `settings.hints.bubbles` / `.walkthrough` / `.markers` (in the `settings` namespace, matching `layout.*`).

## Testing strategy

- **TDD (vitest, red→green):** `hintStyle` (parse/default/round-trip), `hintsContent` (route mapping + a coverage guard
  that every starter anchor has both i18n keys present in en+ru), `bubblePlacement` (side selection + viewport clamping
  on all four edges + oversized-bubble fallback), `walkthroughStep` (clamp, first/last, empty list, direction).
- **Shells:** `tsc --noEmit` + `vite build` green + self-review. Independent reviewer subagent for the walkthrough
  (most stateful: focus/backdrop/scroll) and the placement integration.
- **Gate per PR:** `corepack pnpm typecheck && corepack pnpm test && corepack pnpm build`.

## Implementation sequencing (one shippable PR per loop iteration)

1. **5a — cores + bubbles (default) end-to-end.** `hintStyle.ts`, `hintsContent.ts`, `bubblePlacement.ts` (+ tests);
   `data-hint` attributes on the starter elements; `HintsLayer` + `HintsBubbles`; the "Show hints" button in the "?" panel;
   the Settings radio control; `hints.*` + `settings.hints.*` i18n (en+ru). After 5a the default style works fully.
2. **5b — walkthrough.** `walkthroughStep.ts` (+ tests) + `HintsWalkthrough`. Selectable in Settings.
3. **5c — markers.** `HintsMarkers` (persistent `ⓘ` dots, hover/tap reveal). Selectable in Settings.

Each lands on `develop` via its own worktree, TDD where a pure core exists, self-reviewed, then fast-forward merged.

## Out of scope (v1, YAGNI)

- Server-side storage of the style (localStorage v1, consistent with the onboarding flag; promote later if cross-device
  sync is wanted).
- Per-anchor "seen" tracking / auto-show on first visit (hints are purely on-demand here; first-run guidance is the
  already-shipped welcome card + checklist).
- Opening drawers/menus to reveal hidden anchors on mobile.
- Hints on secondary routes (`/tools/*`, `/reflections`, `/settings`, `/profile`, `/admin`) — added later via
  `hintsContent.ts` + i18n only.
