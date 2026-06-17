# Design — First-Run Guidance & Contextual Help

> Status: approved (brainstorming, 2026-06-17). Next: writing-plans → incremental implementation.
> Target branch: `develop` (staging auto-deploy). Each implementation step is a separate shippable PR.

## Problem

A brand-new user signs up and lands on `/calendar` — a blank week grid with an empty task sidebar, eight nav items, and zero guidance. The app's core loop (**Task → schedule on Calendar → complete → Reflect**) is never explained. Recon of `origin/develop` confirmed there is no onboarding, tour, welcome, sample data, or contextual help anywhere in `web/src`. Testers (including a neurodivergent first-timer) report the app feels "too hard."

## Goals

1. A first-time user understands the core loop **and** achieves a real first win (a real task created, scheduled, and completed/reflected) without external help.
2. On every page, a user can always ask "what is this / what do I do here?" and get an answer in-context.
3. New-user empty states guide the next action instead of showing a blank screen.

Non-goals (deferred to a future spec): a broad visual/UX overhaul beyond the above; server-synced onboarding state; gamification.

## Decisions (locked with the user)

- **First-run mechanism:** welcome card + dismissible **guided checklist** (not a spotlight tour, not a carousel). Self-paced, non-blocking, visible progress — chosen for ADHD-friendliness.
- **Contextual help:** header **"?" panel** explaining the current page (primary), **plus** a "show hints on this page" reveal of inline bubbles (secondary, staged after the panel).
- **Step detection:** data-driven (from counts), not event-spying — self-heals across sessions and out-of-order actions.
- **Flag storage:** `localStorage` for v1 (`neuroboost-*` idiom). Server `UserSettings.onboarding_seen` is a later upgrade.
- **Empty-state fixes** (Calendar / Planning / Dashboard) are **in scope** for this spec.
- **i18n:** new `onboarding` namespace, mirrored **en + ru**. No hard-coded copy.

## Architecture

The feature is almost entirely presentational React, and this codebase has **no `@testing-library/react`** (all web tests are pure functions on jsdom). Therefore every unit is split into a **pure, TDD-able core** plus a **thin presentational shell**. State is held in a small `OnboardingContext` (the codebase uses React Context only — no Zustand/Redux).

### Pure cores (each gets vitest red→green tests)

- **`web/src/lib/onboarding/onboardingFlag.ts`**
  - `getOnboardingState()` / `setWelcomeSeen()` / `setChecklistDismissed()` over `localStorage` keys `neuroboost-onboarding-welcome-seen` and `neuroboost-onboarding-checklist-dismissed`.
  - Pure given an injected storage object (so tests pass a fake `Storage`); tolerant of malformed/missing values (returns defaults, never throws).

- **`web/src/lib/onboarding/checklistProgress.ts`** — the heart of Pillar 1.
  - `computeChecklistProgress({ taskCount, eventCount, completedTaskCount, reflectionCount })` → `{ steps: { createTask, scheduleTask, completeAndReflect }: boolean, currentStep: 'createTask' | 'scheduleTask' | 'completeAndReflect' | 'done', completedCount: number }`.
  - Rules: `createTask` done when `taskCount ≥ 1`; `scheduleTask` done when `eventCount ≥ 1`; `completeAndReflect` done when `completedTaskCount ≥ 1` OR `reflectionCount ≥ 1`. `currentStep` = first not-done step in order, else `'done'`.

- **`web/src/lib/onboarding/helpContent.ts`** — Pillar 2 routing.
  - `resolveHelpKey(pathname)` → the i18n key for the page's help block (e.g. `/tasks` → `help.tasks`), with a default fallback key for unknown paths. Pure string→string.

### Shells (thin; self-review + typecheck/build, no unit tests)

- `OnboardingProvider` / `useOnboarding` — wraps the flag + progress cores, exposes state to the tree, fetches the counts the progress core needs.
- `WelcomeCard` — one-screen explanation of the loop; "Show me" (start checklist) / "Skip" (sets welcome-seen + dismissed).
- `GuidedChecklist` — corner card driven by `computeChecklistProgress`; each step shows a CTA that navigates to the relevant page; fully dismissible.
- `HelpPanel` — header "?" button → slide-in panel rendering `resolveHelpKey(pathname)` copy.
- `HintsLayer` — "show hints" reveal: labelled bubbles anchored to key elements on the current page; off by default.

## Data flow

`OnboardingProvider` reads the flag (localStorage) on mount. If welcome not seen → render `WelcomeCard`. Once the checklist is active, the provider supplies the counts (task/event/completed/reflection — sourced from the existing data the relevant pages already load, surfaced via a lightweight counts query) to `computeChecklistProgress`; `GuidedChecklist` renders from the result. When all steps are done or the user dismisses, set the dismissed flag and stop rendering. `HelpPanel`/`HintsLayer` are independent of the flag — always available via the header "?".

## Mobile / drag dead-end

Checklist step 2 ("put it on your calendar") must be completable without drag-and-drop (touch-hostile). Detection is by `eventCount`, so any creation path satisfies it. **Implementation step 3 ensures a tap/click "Schedule" path exists** for a task (add one if the current UI is drag-only). The checklist CTA opens the calendar/scheduling affordance rather than instructing a drag.

## Error handling

- Cores never throw on bad input: `onboardingFlag` returns defaults on malformed/missing localStorage and swallows `setItem` quota errors (help/checklist are non-critical). `computeChecklistProgress` treats missing counts as 0. `resolveHelpKey` falls back to a generic key.
- If the counts query fails, the checklist degrades to showing all steps incomplete with working CTAs (it never blocks the app).

## i18n

Add `web/src/i18n/locales/en/onboarding.json` and `…/ru/onboarding.json`, and register `onboarding` in the namespace list in `web/src/i18n/index.ts`. Keys: `welcome.*`, `checklist.*` (3 steps + title + progress), `help.<page>` blocks, `hints.*` labels. Every key present in both locales.

## Testing strategy

- **TDD (vitest, red→green):** `onboardingFlag`, `checklistProgress`, `resolveHelpKey`. These hold all branching logic.
- **Shells:** typecheck (`tsc --noEmit`) + `vite build` green + self-review; consistent with how this codebase has shipped UI.
- **Gate per PR:** `corepack pnpm typecheck && corepack pnpm test && corepack pnpm build`.

## Implementation sequencing (one shippable PR per loop iteration)

1. **Cores + context:** `onboardingFlag.ts`, `checklistProgress.ts` (+ tests), `OnboardingProvider`/`useOnboarding`. No visible UI.
2. **First-run UI:** `WelcomeCard` + `GuidedChecklist` wired to the cores; `onboarding.json` (en+ru) for those strings.
3. **Tap-friendly scheduling:** ensure a non-drag "Schedule" path for a task (removes the step-2 dead-end).
4. **Help panel:** `helpContent.ts` (+ tests) + `HelpPanel` ("?") + per-page help copy (en+ru).
5. **Hints reveal:** `HintsLayer` "show hints on this page" (C).
6. **Empty states:** Calendar / Planning / Dashboard adopt the Reflections empty-state pattern (icon + heading + hint + CTA).

Each step lands on `develop` via its own worktree, TDD where a pure core exists, self-reviewed, then fast-forward merged.
