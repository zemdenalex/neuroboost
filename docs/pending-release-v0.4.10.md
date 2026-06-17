# Pending release — v0.4.10 (develop → main)

> **Status:** unreleased. `develop` is **70 commits ahead of `main`** (`53ec973`, v0.4.9, 2026-04-24) through `1f111d6` (2026-06-17).
> This is the candidate for the next production promotion. Short summaries below, grouped by area; newest themes first.
> Verify on staging (dev.neuroboost.website) before promoting `develop` → `main`.

---

## 1. Onboarding & contextual help (new feature, Phases 1–6)

First-run guidance and on-demand help for new/neurodivergent users who found the app "too hard." Designed brainstorm-first, built behind localStorage flags, fully en+ru.

- **Design spec** — first-run guidance + contextual help design (`037155a`); 3-style hints reveal spec (`04fd13f`).
- **Pure cores (TDD)** — onboarding flags, checklist progress, help routing (`dd639ad`); hint style/content/bubble-placement (`a852102`).
- **First-run welcome card** — one-screen explanation of the capture→schedule→do→reflect loop (`bda47e8`).
- **Guided checklist** — dismissible, driven by live task/event/reflection counts; auto-dismisses when done (`f1c40ed`).
- **One-tap Schedule** — schedule a task without drag-and-drop, removing a mobile dead-end (`83afed0`).
- **Contextual "?" help panel** — per-page help on every route (`3588def`); also exposed in the vertical-sidebar layout (`3bdec4a`).
- **Guided empty states** — Planning unscheduled list (`f32c3a4`) and Dashboard (`e60c401`).
- **"Show hints" reveal — 3 selectable styles** in Settings: all-at-once bubbles (`240af09`), stepped spotlight walkthrough (`2bbad81`), persistent markers (`f5763ef`).

## 2. Pomodoro focus timer (new feature — rebuilt)

A cross-page focus timer rebuilt on a global, timestamp-based engine.

- **Spec + plan** (`2ef64a7`), Vitest + jsdom test setup (`560b137`).
- **Pure engine (TDD)** — timer types/defaults (`b2e38aa`), phase-transition machine (`46b50d5`), persistence + remaining/stale computation (`ea796d3`), focus-event builder with completion/undo (`2d39a6c`).
- **Time logging** — `task.actual_minutes` column (`274a0b4`), `LogTimeRequest` + API (`2fd303d`, `8696e41`, `f8f0c6a`), client `logTaskTime` (`b6c70b7`).
- **Global context + UI** — single-fire transition context (`e9de4aa`), beep/notification helpers (`ea6843f`), floating cross-page widget with pill/card/bar styles (`c49961b`), page rebuilt on the context with cycle pips (`3d1ee60`), card polish + pure settings updater (`c138524`).
- **Toasts** — global completion/undo toast queue (`547cda5`), Retry on a failed save with partial-event rollback (`aaac50d`), i18n keys (`114833d`).
- **Fix** — "Today" focus stats computed by local day, not UTC (`955b5f4`).

## 3. Calendar & events (fixes + correctness)

- **Recurring events keep local time across DST** — was advancing in UTC and drifting ±1h after transitions (`04b5211`).
- **Calendar no longer empty on Sundays** — week-range off-by-one fetched the wrong week (`4308a52`).
- **Overlapping events** laid side-by-side in lanes (`49750ae`); multi-day event ending at midnight renders its last day full-height (`67a75aa`); cross-midnight end-date drift in positive-offset zones (`6685407`).
- **Recurrence correctness** — MONTHLY skips monthless days instead of drifting (`8c81930`); UNTIL bound inclusive of the whole final day (`39069d0`); long-running events no longer vanish from far-future windows (`db9223f`).
- **List endpoint** rejects inverted date ranges instead of silently returning empty (`7d1a168`); time-range validated on update via a shared helper (`213222c`).
- All-day drag-create ghost labels localized (`5d2889b`); timezone-util math locked by tests (`c676f19`).

## 4. Tasks

- **Due-date picker no longer shifts by the UTC offset on edit** (`018c544`).
- **Due dates labelled by calendar day**, not elapsed 24h (`fe94add`).
- Status/priority/category validated on create + update (`34f617b`); zero-length events prevented when scheduling (`e773e7f`).
- Duplicate task creation from double-submit prevented (`cef6b8b`); save/delete failures surfaced + inline delete hardened (`9e331a4`).
- Priority colors unified across Tasks, sidebar, calendar, Kanban, Eisenhower (`07e30a1`, `0498bd1`).

## 5. Backend / API hardening

- **Import validates each row** (event time-range, task status/priority/category) instead of silently persisting invalid data; reports skipped counts (`8da7b7b`).
- `actual_minutes` preserved across backup export/import (`5e42e43`); export/import surface HTTP errors instead of masking them (`8a82d1a`).
- API client foundation covered by tests (auth, errors, envelope, 401) (`475f5e6`).
- *(Focused security audit — user-scoping, admin checks, JWT — reviewed clean; no code change.)*

## 6. i18n & profile

- Date formatting follows the UI language via a centralized `dateLocale` (`e043939`); sidebar due-date labels localized (`da89feb`).
- Profile timezone accepts any valid IANA zone (was a 6-zone allowlist) (`d4e8da5`); hardcoded English name/date fallbacks localized (`e6e8634`).

## 7. Planning

- `startOfWeek` normalized to midnight; duplicate `getMondayOf` dropped (`61e16d6`).
- `fetchPlan` dependency list corrected; sole eslint-disable removed (`1f111d6`).

## 8. Infrastructure & cleanup

- **CI/deploy hardened** — git fetch/pull wrapped in retry to ride out transient DNS on the deploy host (the failure that blocked the last dev deploy); Go cache path fixed (`api-go/go.sum`); deprecated Node-20 actions bumped (`6cfa3b9`).
- Dead code removed — EisenhowerMatrix component (`271dfcd`), `lib/api` stub / `lib/time` / `utils/priority` (`618a1dc`); Kanban due dates use shared `lib/dueDate` (`f31c065`).

---

## Notes for the promotion

- **Scope:** this is a large release (70 commits, ~2 months of work) — two new features (onboarding/help, Pomodoro timer) plus a long run of correctness/i18n/infra fixes.
- **Untested-in-browser:** the onboarding/hints UI and the Pomodoro widget were built with unit tests + typecheck/build only — there is no React-component test harness — so a manual pass on staging is recommended before promoting.
- **Deploy prerequisite:** confirm the dev deploy recovered after the CI fix (`6cfa3b9`) — watch for the Telegram "DEV deployed" notification. If it still fails with "Could not resolve host", that's the dev server's DNS (server-side fix needed).
- **Promotion is fast-forward-safe** if `main` is an ancestor of `develop` (it is, as of this writing).
