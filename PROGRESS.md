# PROGRESS

> 🔴 **Снимок заморожен на 2026-07-19 и НЕ знает о переприоритизации 27.07** (P1 задачи ·
> P2 уведомления · P3 общие события). Всё ниже про MD1/MD2 как «проблему номер один» —
> устарело; они в backlog. Числа тоже протухли, пересчитывать по `CLAUDE.md` §Счётчики.
> Актуальный статус — `docs/ROADMAP.md`, что чему верить — `docs/DOCS-MAP.md`.

> **`docs/ROADMAP.md` is the status source of truth.** This file is a short, verified snapshot.
> Last verified against the filesystem and a full test run: **2026-07-19** (branch `develop`).
>
> *Historical note: this file previously described the December 2025 v0.4.0 skeleton and claimed
> "all handlers return 501" and "18 migration files" long after both stopped being true. If a
> line here disagrees with the code, trust the code and fix this file.*

## Where the work is

| Branch | State |
|--------|-------|
| `main` | v0.4.9, tagged 2026-04-24 — what production runs |
| `develop` | v0.4.10 candidate, ~71 commits ahead, **unreleased** |

## Verified state (2026-07-19)

**Build & test — all green:**

| Gate | Result |
|------|--------|
| `go build ./...` | pass |
| `go test ./...` | pass — auth, events, export, tasks |
| `pnpm typecheck` | pass |
| `pnpm test --run` | pass — 190 tests, 26 files |
| `pnpm build` | pass |

**Backend:** 46 routes registered in `api-go/cmd/api/main.go`; 9 migrations.
Implemented: auth, events (incl. RRULE recurrence + exceptions), tasks, reflections, planning,
export/import, feedback, admin, status.
Still 501 stubs: **needs, opportunities, patterns** — those three only.

**Frontend pages:** Home, Login, Calendar, Tasks, Planning, Reflections, Tools, Settings,
Profile, Admin.

**Shipped feature groups:** calendar week/day grid with drag-create/move/resize, task sidebar
and task→event scheduling, event editor with reflections, Pomodoro focus timer, onboarding and
contextual-help system (3 hint styles), Kanban, Eisenhower, time-blocking, export/import,
en+ru bilingual UI, Telegram bot.

## Known broken

- **MD1 / MD2 — multi-day event move & resize** (High). Drag state machine's cursor-to-time
  mapping breaks across day columns; resize collapses the event.
- **MV1** — mobile calendar views (3-day, agenda, mini-month) not built; only swipe-day nav.
- **PE1** — `preventDefault` passive-listener console warnings on touch handlers (cosmetic).

See `docs/ROADMAP.md` for the full bug registry and what's planned next.
