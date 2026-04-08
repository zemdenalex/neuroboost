# NeuroBoost

Calendar-first productivity app for neurodivergent users. Go backend + React/TypeScript frontend + PostgreSQL. "The calendar is truth; tasks exist to support scheduling and reflection."

**Live:** https://neuroboost.website | **Version:** v0.4.0

## Tech Stack

| Layer | Stack |
|-------|-------|
| Backend | Go 1.22, chi router, pgx/v5, JWT auth |
| Frontend | React 18, TypeScript, Vite 5.4, Tailwind CSS 3.4, pnpm 10 |
| Database | PostgreSQL 16, golang-migrate (6 migrations) |
| Infra | Docker Compose, Nginx, GitHub Actions CI |

## Commands

```bash
# Full stack
docker compose up -d
docker compose logs -f api

# Backend
cd api-go && go build -o bin/api ./cmd/api
cd api-go && go test -v ./...

# Frontend
cd web && pnpm dev          # port 5173
cd web && pnpm typecheck    # MUST pass before done
cd web && pnpm build        # MUST pass before done

# Migrations auto-run on API start. Manual:
migrate -path api-go/migrations -database "$DATABASE_URL" up
```

## Architecture

- **Backend entry:** `api-go/cmd/api/main.go` (all 36 routes, lines 54-124)
- **Handler pattern:** `internal/{module}/handlers.go` + `types.go`. Some use `NewHandler(db, cfg)`, others `InitDB(db)` + exported functions
- **Frontend entry:** `web/src/App.tsx` -> `web/src/router.tsx` (React Router v6)
- **API client:** `web/src/api/client.ts` with `api.get/post/patch/delete`
- **Auth state:** `web/src/contexts/AuthContext.tsx`
- **Stubs (return 501):** needs, opportunities, patterns, planning, reflections

See @docs/CODEBASE_MAP.md for detailed architecture, diagrams, and module analysis.
See @docs/NeuroBoost_v0_4_0_Feature_List.md for roadmap (117 features, 17 done).

## Gotchas

1. **Token expiry mismatch:** Stored as Unix seconds, compared with `Date.now()` (milliseconds)
2. **Duplicate `toNbEvent()`:** Exists in 3 places with different implementations — consolidate before extending
3. **Priority inversion:** 1=Emergency, 5=If Possible (lower number = higher priority)
4. **Default timezone:** "Europe/Moscow" hardcoded throughout — will need i18n
5. **Legacy auth files:** `internal/auth/db.go` and `jwt.go` are unused by current handlers
6. **Arrays default to `[]`:** Never null from API
7. **Week starts Monday** (ISO standard)

## Workflow

1. **Explore** — read relevant files, understand context
2. **Plan** — use plan mode for multi-file changes
3. **Implement** — follow existing patterns (see `.claude/rules/`)
4. **Verify** — `pnpm typecheck && pnpm build` (frontend), `go build ./...` (backend)
5. **Commit** — descriptive message, never co-author Claude/Anthropic

## Rules

### ALWAYS
- Read this file + relevant rules before working
- Run `pnpm typecheck && pnpm build` before marking frontend work done
- Run `go build ./...` before marking backend work done
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
