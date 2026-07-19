# NeuroBoost

Calendar-first productivity app for neurodivergent users. Go backend + React/TypeScript frontend + PostgreSQL. "The calendar is truth; tasks exist to support scheduling and reflection."

**Live:** https://neuroboost.website · **Staging:** https://dev.neuroboost.website
**Released:** v0.4.9 (tagged 2026-04-24 on `main`) · **Unreleased:** v0.4.10 on `develop`

> ⚠️ **Read before anything else** (verified by a full audit on 2026-07-19):
> - **Active development is on `develop`**, ~71 commits ahead of `main` (147 files, +6173/−962),
>   including *all* tests and migration `000009`. A fresh checkout lands on `main` and is missing
>   them. **Check out `develop` before reading code.**
> - **`docs/ROADMAP.md` is the status source of truth.** `docs/NeuroBoost_v0_4_0_Feature_List.md`
>   is a historical planning artifact from Dec 2025 — its per-row statuses are wrong across the
>   board (it marks shipped features as 📋 planned). Never read status from it.
> - **Event recurrence (RRULE) is BUILT** — `api-go/internal/events/recurrence.go`
>   (`parseRRule` / `expandRecurrence` / `buildRRule`, exceptions via the `event_exception`
>   table). **Reuse it, do not rewrite it.** Tasks have no recurrence — that is the real gap.
> - **Counts drift. Count the filesystem, never trust a doc line** — including this one.
> - This project has **no `graph/`**, deliberately. Durable notes live here and in `.remember/`
>   (`loop-state.md` holds the running backlog from prior development loops).

## Tech Stack

| Layer | Stack |
|-------|-------|
| Backend | Go 1.22, chi router, pgx/v5, JWT auth |
| Frontend | React 18, TypeScript, Vite 5.4, Tailwind CSS 3.4, pnpm 10.12.0 |
| Database | PostgreSQL 16, golang-migrate (**9** migrations on `develop`, 8 on `main`) |
| Testing | Go: 4 packages with tests · Frontend: Vitest + jsdom, **190 tests / 26 files** |
| Infra | Docker Compose, Nginx, GitHub Actions CI, auto-deploy per branch |

## Commands

```bash
# Full stack
docker compose up -d
docker compose logs -f api

# Backend  (both MUST pass before backend work is done)
cd api-go && go build ./...
cd api-go && go test ./...

# Frontend (all three MUST pass before frontend work is done)
cd web && pnpm typecheck
cd web && pnpm test --run
cd web && pnpm build
cd web && pnpm dev          # port 5173

# Migrations auto-run on API start. Manual:
migrate -path api-go/migrations -database "$DATABASE_URL" up
```

**pnpm not found?** This repo pins pnpm via the `packageManager` field. Run `corepack enable`
once — it activates the correct version (10.12.0) automatically. Do not `npm i -g pnpm`.

## Architecture

- **Backend entry:** `api-go/cmd/api/main.go` — **46** routes
- **Handler pattern:** `internal/{module}/handlers.go` + `types.go`. Some use `NewHandler(db, cfg)`,
  others `InitDB(db)` + exported functions
- **Frontend entry:** `web/src/App.tsx` → `web/src/router.tsx` (React Router v6)
- **API client:** `web/src/api/client.ts` with `api.get/post/patch/delete`
- **Auth state:** `web/src/contexts/AuthContext.tsx`
- **Real 501 stubs — only these three:** `needs`, `opportunities`, `patterns`.
  `planning` and `reflections` are **fully implemented** (zero 501s in either).

See @docs/ROADMAP.md for current status and what's next.
See @docs/CODEBASE_MAP.md for architecture detail (⚠️ last mapped 2026-01-21 — predates
v0.4.4→v0.4.10; treat its module inventory as indicative, not current).

## Gotchas

1. **`E:` is a Windows junction to `C:\E_Drive`.** Vite resolves real paths while Vitest globs
   yield the junction path — every test file then fails to load. Handled by the Vitest-scoped
   `resolve.preserveSymlinks` in `web/vite.config.ts`. **Do not enable that flag for builds** —
   it breaks Rollup's resolution of pnpm's symlinked `node_modules`.
2. **Token expiry mismatch:** stored as Unix seconds, compared with `Date.now()` (milliseconds)
3. **Two parallel event API stacks** (this replaces the old "duplicate `toNbEvent()` in 3 places"
   note — that was fixed; there is now exactly one, `web/src/types/index.ts:88`).
   Raw snake_case `web/src/api/events.ts` vs camelCase `web/src/api/index.ts`. **Both export a
   `moveEvent` that hits a different endpoint** — `PATCH /events/:id/move` vs generic
   `PATCH /events/:id`. The calendar never calls the dedicated `/move` and `/resize` endpoints.
   Check which stack you're importing from.
4. **Priority inversion:** 1=Emergency, 5=If Possible (lower number = higher priority), 0=Buffer
5. **Default timezone:** "Europe/Moscow" hardcoded in places — will need i18n
6. **Legacy auth files:** `internal/auth/db.go` and `jwt.go` are unused by current handlers
7. **Arrays default to `[]`:** never null from API
8. **Week starts Monday** (ISO standard)
9. **Health endpoint version is hardcoded** (`internal/status/handlers.go:44` → `"0.4.0"`) —
   it cannot tell you which build is deployed

## Known Broken

Verified by code audit 2026-07-19 — see `docs/ROADMAP.md` for detail and fix plans.

- **MD1 / MD2 — multi-day event move & resize** (High). Root cause is a **coordinate system**
  problem, not a broken state machine: resize states store time as minutes-within-one-day plus a
  single `dayUtc0` (`weekgrid.types.ts:80-94`), which cannot represent a range crossing midnight.
  `utcToLocalMinutes` (`timezone.utils.ts:38-43`) does `localMs % DAY_MS`, discarding the date.
  Resize also ignores the cursor's X/day entirely (`useWeekGridDrag.ts:106`) and `min/max`-swaps
  endpoints, so it silently moves the endpoint you weren't holding.
  ⚠️ Resize has **no movement threshold** — a plain *click* on a multi-day event's resize zone
  commits the collapse.
- **Mutating any recurring-event instance returns 500** (High). `expandRecurrence` gives
  instances synthetic IDs `"<parentUUID>:2026-07-21"` (`events/recurrence.go:152`); no handler
  strips the suffix, so the UUID cast fails. Recurring events are effectively display-only. The
  `/exceptions` endpoint is built but the frontend never calls it.
- **`getTasks` returns snake_case cast to a camelCase type** (Medium). No conversion happens
  (`web/src/api/index.ts:161-168`), so `task.estimatedMinutes` is always `undefined` and
  scheduling a task always defaults to 60 minutes.

## Workflow

1. **Explore** — read relevant files, understand context
2. **Plan** — use plan mode for multi-file changes
3. **Implement** — follow existing patterns (see `.claude/rules/`)
4. **Verify** — frontend: `pnpm typecheck && pnpm test --run && pnpm build`; backend:
   `go build ./... && go test ./...`
5. **Commit** — descriptive message, never co-author Claude/Anthropic

**Branching:** implement on `develop` (auto-deploys to staging) → manual test → PR to `main` →
tag on `main` → production deploys.

## Rules

### ALWAYS
- Read this file + relevant rules before working
- Run the full verify step above before marking work done
- Use parameterized queries with pgx — never string concatenation for SQL
- Use existing API client (`api.get/post/patch/delete`) for frontend API calls
- Use Lucide React for icons, Tailwind for styling
- Surface assumptions before implementing

### DO NOT
- Edit `.env` files — use `.env.example` as reference
- Use `any` in TypeScript
- Add new dependencies without discussing first
- Skip verification steps
- Delete or modify existing migrations — create new ones
- Hardcode secrets or credentials
