# NeuroBoost Roadmap

> **Single source of truth for status and versioning.** Updated: 2026-07-19 (post-audit).
> Other docs lag; this one is maintained. `docs/NeuroBoost_v0_4_0_Feature_List.md` is a
> historical planning artifact — do not read status from it.

## Versioning Convention

`vMAJOR.MINOR.PATCH` — where:
- **MAJOR** (0.x) = pre-production, (1.x) = production-ready
- **MINOR** = feature group / era
- **PATCH** = incremental releases within a group

Tags are created on `main` after merging tested `develop` code.

---

## Version History (Released)

| Tag | Date | Theme |
|-----|------|-------|
| `v0.3.0` | 2025-09-01 | Production deployment (Prisma, old stack) |
| `v0.4.0-skeleton` | 2025-12-24 | Go+React rewrite skeleton |
| `v0.4.0.2` | 2025-12-25 | Auth foundation (email/password, JWT, Telegram backend) |
| `v0.4.0.3` | 2026-03-13 | Claude Code setup (skills, rules, agents, CI) |
| `v0.4.1.0` | — | CI/CD, health checks, env vars |
| `v0.4.1.1` | — | Admin patch |
| `v0.4.2.0` | — | Profile page, calendar view, task creation, events |
| `v0.4.3.0-beta` | — | Calendar page with old+new features, bilingual support |

---

## Current Sprint: v0.4.4 – v0.4.9 (DONE)

**Priority:** Calendar UX > Mobile > Telegram > Features > Polish

| Version | Theme | Key Deliverables | Status |
|---------|-------|-----------------|--------|
| **v0.4.4** | Calendar UX Fix | Fix click/select/resize/move model, task-to-calendar drag, performance, i18n day names | ✅ Tagged |
| **v0.4.5** | Mobile Polish | 4 mobile calendar views (day/3-day/month/agenda), event editor responsive, fix nav overlaps | ✅ Tagged (partial — see M*) |
| **v0.4.6** | Telegram Bot + MiniApp | Assistant bot commands, notification bot, MiniApp integration, Telegram WebApp auth | ✅ Tagged (bot blocked by proxy) |
| **v0.4.7** | New Pages & Tools | Home dashboard, planning page, reflections page, pomodoro timer, kanban board | ✅ Tagged |
| **v0.4.8** | Settings & Cleanup | Global UI scale, feature toggles, export/import, bug cleanup | ✅ Tagged |
| **v0.4.9** | Polish | Reflections migration, calendar null guard, Planning two-pane + TZ fix, Settings hybrid auto-save, mobile Telegram login, backup script | ✅ Tagged 2026-04-24 |

## v0.4.10 — Onboarding, Pomodoro & Hardening (BUILT, UNRELEASED)

**Status as of 2026-07-19:** complete on `develop`, ~71 commits ahead of `main`. Code is green
(Go build+test, `pnpm typecheck` + 190 tests + build all pass). **Not yet promoted to `main`.**
See `docs/pending-release-v0.4.10.md` for the full commit-level breakdown.

| Delivered | Notes |
|-----------|-------|
| Onboarding & contextual help | First-run welcome, guided checklist, per-page "?" panel, guided empty states, 3 selectable hint styles. en+ru. |
| Pomodoro focus timer | Rebuilt on a global timestamp-based engine; cross-page widget, cycle pips, time logging to `task.actual_minutes`, toasts with undo/retry. |
| Calendar & recurrence fixes | DST drift, Sunday week-range off-by-one, overlap lanes, MONTHLY skips, UNTIL bound, far-future windows. |
| Task fixes | Due-date UTC-offset shift, validation on create/update, double-submit guard, unified priority colours. |
| Backend hardening | Import row validation, inverted date-range rejection, export/import error surfacing, API client test coverage. |
| i18n & profile | Centralized `dateLocale`, any valid IANA timezone, localized fallbacks. |

**Promotion checklist:**
1. Push `develop` → auto-deploys to dev.neuroboost.website
2. **Manual browser pass** — onboarding/hints UI and Pomodoro widget have unit tests only,
   never been exercised in a browser (no React component-test harness)
3. PR `develop` → `main`. ⚠️ Not a fast-forward — `main` carries docs commits `develop` lacks,
   so expect a normal merge (possible trivial ROADMAP conflict).
4. Tag `v0.4.10` on `main` → production deploys

## Next Sprint: v0.4.11 — Multi-day Events

**Chosen 2026-07-19.** Fixing an actually-broken feature before adding new ones.
Root cause established by code audit — this is a **contained coordinate-system fix, not a
state-machine redesign**. The discriminated union in `weekgrid.types.ts:47-94` is sound; keep it.

### Root cause

Three of the four drag states store time as *minutes-within-one-day* + a single `dayUtc0`,
which cannot represent a range crossing midnight. `utcToLocalMinutes`
(`timezone.utils.ts:38-43`) does `localMs % DAY_MS`, discarding the date.

| ID | Symptom | Cause |
|----|---------|-------|
| MD2 | Resize collapses a multi-day event to one day | `startResizeEnd` (`useWeekGridDrag.ts:48-52`) drops the date; `handleResizeComplete` (`dragHandlers.ts:88-99`) rebuilds *both* endpoints on one `dayUtc0` with a `min/max` swap — moving the endpoint the user wasn't holding |
| MD1 | Cursor-to-time mapping breaks across columns | `onMove` (`useWeekGridDrag.ts:106`) updates only `curMin` from Y and **discards the `targetDayUtc0` computed from X** |
| — | ⚠️ A plain **click** on a resize zone collapses the event | Resize has no movement/threshold guard — `onUp` (`useWeekGridDrag.ts:111-117`) only skips commit for `move`+`pending` |
| — | Multi-day move shows a meaningless 15px ghost | `startMove` computes `durMin` mod 24h (can go negative); `MultiDayTimedGhost` only renders for `kind === 'create'` (`GhostPreview.tsx:84`) |
| — | Single-day move jumps ~1h on grab | `offsetMin` is overwritten by the first mousemove; there is no `grabOffset` field |
| — | All-day multi-day move re-anchors to the grabbed cell | `AllDaySection.tsx:88` passes the *cell's* day; the all-day branch (`dragHandlers.ts:63-68`) skips the delta correction the timed branch does |
| — | First all-day drag of a session is inert | `handleAllDayDragStart` (`WeekGrid.tsx:143-150`) never sets `dragMeta.current`, so `onMove` early-returns |

**Note:** the multi-day *timed move commit* is already correct and delta-based
(`dragHandlers.ts:69-78`) — copy that pattern. Keyboard move is correct too (`useKeyboardNav.ts:52-59`).

### Fix plan (~150–200 lines across 4 files)

1. **Write `dragHandlers.test.ts` first** — `handleDragComplete` is pure and has **zero** tests
   today, which is exactly where these bugs live. The scenarios above become the regression cases.
2. Resize states carry absolute `startMs`/`endMs` (as `DragMove.originalStartMs/EndMs` already
   does); `onMove` computes `cursorAbsMs = targetDayUtc0 + curMin*60000` for **all** kinds.
3. `handleResizeComplete`: **clamp, don't min/max-swap** — resize-end → `newEnd = max(cursorAbs,
   start + 15min)`, held endpoint untouched. Fixes MD1 + MD2 together.
4. Add `grabOffsetMs` at mousedown; move commit = `cursorAbs − grabOffset`. Unifies single/multi-day
   move and removes the `daySpan > 1` special case.
5. All-day move: reuse the timed branch's delta pattern; set `dragMeta` in `handleAllDayDragStart`.
6. Add a movement threshold before any resize commits.
7. Generalize `MultiDayTimedGhost` to render any absolute range (largest UI chunk).

**Schedule alongside:** the recurring-instance 500 (below). Drag work will make it more visible.

## Backlog (post-v0.4.11)

| Feature | Notes |
|---------|-------|
| Eisenhower Decision Helper | Inbox of untriaged tasks + triage wizard (3 questions) + overload advisor + healthy-life nudges. User wants it to "make users ask questions about each problem". Big — decompose/brainstorm first. |
| Kanban rework | 5/10 → 8/10 UX. Design/UX improvements + missing features (TBD brainstorm). |
| Mobile calendar views (MV1) | 3-day, agenda, mini-month. Current swipe-day-nav is the only mobile calendar mode. |
| Profile editable fields | Beyond name — timezone, gamification stats real data. |
| Planning v2 polish | Day cells didn't stretch vertically even with `auto-rows-fr` — parent height chain needs investigation. Maybe redesign to hourly grid. |

**Workflow per version:**
1. Implement on `develop`
2. Auto-deploy to `dev.neuroboost.website`
3. Manual testing + user approval
4. PR `develop` → `main`, merge
5. Tag on `main` (e.g. `git tag v0.4.4`)
6. Push tag, production deploys

---

## Future Roadmap

| Version | Era | Theme | Key Features |
|---------|-----|-------|-------------|
| **v0.5.x** | Task Intelligence | Smart scheduling, context-awareness | Smart scheduling algorithm, energy patterns, routines, dependencies, critical path |
| **v0.6.x** | Real-time & Sync | Live updates, external integrations | WebSockets, Google Calendar sync, CalDAV |
| **v0.7.x** | Gamification | Motivation system | XP, levels, streaks, badges, achievements |
| **v0.8.x** | Social & Sharing | Multi-user, collaboration | Multi-user support, shared calendars, leaderboard |
| **v0.9.x** | Platform | Native apps, PWA | PWA installation, React Native mobile apps |
| **v1.0** | Production Ready | Stable, polished, scalable | Full test coverage, performance optimized, documentation complete |

---

## Out of Scope (until labeled version)

| Feature | Earliest Version |
|---------|-----------------|
| WebSockets / real-time | v0.6.0 |
| Google Calendar sync | v0.6.0 |
| Gamification (XP, badges) | v0.7.0 |
| Multi-user / collaboration | v0.8.0 |
| Native mobile apps | v0.9.0 |
| Google OAuth | v0.5.0+ |
| Email verification | v0.5.0+ |
| Task dependencies | v0.5.0 |
| Subtasks | v0.5.0 |
| i18n beyond EN/RU | v0.6.0+ |
| Offline / service workers | v1.0 |

---

## Bug Registry (v0.4.3 testing, 2026-04-08)

| ID | Bug | Fix Version | Severity |
|----|-----|-------------|----------|
| C1 | Click event triggers resize instead of select | v0.4.4 | Critical |
| C2 | Can't drag to move events | v0.4.4 | High |
| C3 | Multi-day events collapse when resizing | v0.4.4 | High |
| C4 | Task sidebar opens by default | v0.4.4 | Low |
| C5 | Add task from sidebar uses browser prompt() | v0.4.4 | Medium |
| C6 | Task drag deletes task, creates plain event | v0.4.4 | High |
| C7 | Performance degrades >20 events | v0.4.4 | Medium |
| L1 | Calendar day names stay Russian in EN mode | v0.4.4 | Medium |
| L2 | Task sidebar doesn't translate to RU | v0.4.4 | Medium |
| M1 | Hamburger icon overlaps logo | v0.4.5 | Medium |
| M2 | FAB overlaps feedback button | v0.4.5 | Medium |
| M3 | Mobile calendar: 1 day only, no nav, cuts at 4pm | v0.4.5 | High |
| M4 | Event editor overflows on mobile | v0.4.5 | High |
| S1 | UI scale only applies to Settings page | v0.4.8 | Medium |
| S2 | UI scale applies before saving (confusing default) | v0.4.8 | Medium |
| S3 | Export/Import non-functional | v0.4.8 | Low |
| S4 | Feature toggles don't affect anything | v0.4.8 | Low |

### Known Limitations (deferred to post-v0.4.9)

| ID | Issue | Priority | Notes |
|----|-------|----------|-------|
| MD1 | Multi-day event move/resize broken | **High** | Drag state machine needs redesign — cursor-to-time mapping breaks across day columns. Needs dedicated sprint. |
| MD2 | Multi-day event resize collapses event | **High** | Same root cause as MD1. |
| MV1 | Mobile calendar views (3-day, agenda, mini-month) | Medium | Spec'd in v0.4.5 but deferred — only swipe day nav implemented. |
| PE1 | preventDefault passive listener warnings | Low | Touch handlers need `{ passive: false }` option. Cosmetic console noise. |
| S4 | Feature toggles don't affect anything | v0.4.8 | Low |

---

## Newly Found Bugs (code audit 2026-07-19)

Not in the old bug registry — found by reading code, all verified with file:line evidence.

| ID | Bug | Severity | Detail |
|----|-----|----------|--------|
| **R1** | **Mutating any recurring-event instance returns 500** | **High** | `expandRecurrence` assigns synthetic IDs `"<parentUUID>:2026-07-21"` (`events/recurrence.go:152`). No handler strips the suffix (`handlers.go:187,259,291`), and `event.id` is a Postgres `UUID` — so the cast fails and returns 500 (not `ErrNoRows`). Drag, Shift+Arrow, edit-save or delete on any recurring instance fails and the calendar snaps back. **Recurring events are effectively display-only.** `AddExceptionHandler` exists but the frontend never calls it — decide the instance-mutation story (exception + detached event) during v0.4.11. |
| **T1** | `getTasks` casts snake_case JSON to a camelCase type | Medium | Go emits `estimated_minutes`/`due_date` (`tasks/types.go:37,39`); `getTasks` (`web/src/api/index.ts:161-168`) casts with **no conversion**. So `task.estimatedMinutes` is always `undefined` → `scheduleTask` always defaults to 60 min (`api/index.ts:208`). Same for `dueDate` reads off that array. |
| **A1** | Two same-named `moveEvent` functions hit different endpoints | Medium | `api/events.ts` → `PATCH /events/:id/move`; `api/index.ts:135-137` → generic `PATCH /events/:id`. The calendar never calls the dedicated `/move`+`/resize` endpoints, so any move-specific backend logic is silently bypassed. |
| **A2** | Two `NbEvent` interfaces | Low | `types/index.ts:66` (full) vs `weekgrid.types.ts:4` (subset, missing `recurringEventId`/`reflections`). Structural typing hides it until WeekGrid needs `recurringEventId` — which R1's fix will require. |
| **D1** | Dead module `web/src/hooks/useEvents.ts` | Low | Zero consumers. Delete with the §Dead Code batch. |

### Dead code — verified safe to delete
`api-go/internal/auth/telegram.go` `VerifyTelegramAuth` (zero callers; live path is the private
`(h *Handler) verifyTelegramAuth` at `handlers.go:310`), `api-go/internal/auth/db.go` (all 4
exported funcs unused; uses `database/sql` while live code is pgx), `api-go/internal/auth/jwt.go`
(unused; live JWT is `handlers.go:355 generateJWT`) — plus the then-orphaned `auth/types.go:28
Claims` — and `web/src/hooks/useEvents.ts`. One safe cleanup commit.

*Already deleted on develop in `618a1dc`: `web/src/utils/priority.ts`, `web/src/lib/time.ts`.
Do not confuse `utils/priority.ts` (gone) with `lib/priority.ts` (live, used by
`weekgrid.constants.ts:17`).*

## Engineering Health (audit 2026-07-19)

### Fixed in this pass
- **CI now runs the frontend tests.** It previously ran `typecheck` + `build` only, so all
  190 tests gated nothing. CI also installed `pnpm@9` against a `pnpm@10.12.0` declaration —
  now pinned to match `packageManager`.
- **`E:` junction test breakage.** `E:` is a Windows junction to `C:\E_Drive`; Vite resolved
  real paths while Vitest globs yielded junction paths, so all 26 test files failed to load.
  Fixed via a **Vitest-scoped** `resolve.preserveSymlinks` in `web/vite.config.ts` — enabling
  it for builds breaks Rollup's resolution of pnpm's symlinked `node_modules`.

### Open
| Item | Priority | Notes |
|------|----------|-------|
| `pnpm-lock.yaml` is gitignored (`.gitignore:104`) | Medium | Frontend builds are not reproducible; CI can't use `--frozen-lockfile`, so dependency versions can drift silently between local, CI and deploy. Committing it is the standard fix. |
| Health endpoint version hardcoded | Low | `api-go/internal/status/handlers.go:44` always reports `"0.4.0"` — it cannot tell you which build is live. Wire to a build-time ldflag or the tag. |
| No React component-test harness | Medium | Only pure logic is tested. UI features (onboarding, Pomodoro widget) ship on unit tests + manual passes alone. |
| Drag layer has zero tests | **High** | No tests for `dragHandlers.ts` or `useWeekGridDrag.ts` — exactly where MD1/MD2 live. `handleDragComplete` is pure and trivially testable. Fix first in v0.4.11. |
| `docs/CODEBASE_MAP.md` stale | Medium | Last mapped 2026-01-21 — predates v0.4.4→v0.4.10. Re-run the cartographer skill. |
| Stale remote branches | Low | `v0.3.3-archive`, `main-v0.3.x-backup`, `feature/auth-system`, `chore/docs-and-workflow` — confirm each is merged/obsolete before pruning. |
