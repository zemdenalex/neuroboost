# NeuroBoost

> Calendar-first productivity app for neurodivergent users

**Status:** v0.4.9 released · v0.4.10 in development  
**Live:** [neuroboost.website](https://neuroboost.website) · **Staging:** [dev.neuroboost.website](https://dev.neuroboost.website)

---

## What is NeuroBoost?

NeuroBoost is a "pushy personal assistant" that helps you stay focused on what matters. Unlike passive to-do lists, it actively nudges you toward your goals through respectful reminders, intelligent task suggestions, and honest reflections.

**For:** Neurodivergent folks, students, freelancers — anyone who knows their goals but struggles in the moment.

**Philosophy:** Calendar-first, task-smart. The calendar is truth; tasks exist to support scheduling and reflection.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.22 + chi router |
| Database | PostgreSQL 16 |
| Frontend | React 18 + TypeScript + Vite + Tailwind |
| Bot | gotgbot (Telegram) |
| Deploy | Docker + Nginx + Let's Encrypt |

---

## Quick Start (Development)

```bash
# Clone
git clone https://github.com/zemdenalex/neuroboost.git
cd neuroboost

# Copy environment
cp .env.example .env
# Edit .env with your values

# Start services
docker compose up -d

# Check health
curl http://localhost:8080/api/health
```

---

## Project Structure

```
neuroboost/
├── api-go/          # Go backend API
├── web/             # React frontend
├── docs/            # Documentation
├── scripts/         # Deployment scripts
└── docker-compose.yml
```

---

## Documentation

- [DOCS-MAP](docs/DOCS-MAP.md) — **read this first**: which document is alive, which is archive,
  and what each one contains that the code does not. Any document here older than a week is
  worth checking against it before you trust a line in it
- [Roadmap](docs/ROADMAP.md) — **status source of truth**: versions, bug registry, what's next
- [Codebase Map](docs/CODEBASE_MAP.md) — architecture and module guide. Its diagrams and
  conventions hold; ⚠️ its **inventory** is from January 2026 and its counts are wrong
- [CLAUDE.md](CLAUDE.md) — development conventions, commands, gotchas
- [Progress](PROGRESS.md) — ⚠️ frozen 2026-07-19, superseded by the Roadmap. Kept for history

🔴 **Never copy a number out of a document — recompute it.** Every counter in this repository has
gone stale at least twice. The commands are in [CLAUDE.md](CLAUDE.md) under §Счётчики.

---

## Current Version

`main` runs **v0.4.9**. `develop` holds the unreleased v0.4.10 candidate and is a long way
ahead — recompute with `git rev-list --count main..develop` rather than trusting any figure
written down here or elsewhere.

🔴 **Merging `develop` into `main` IS the production release** (`ci.yml`, job `deploy`, gated on
`refs/heads/main`). The tag is applied afterwards and triggers nothing. Do not merge without
Denis's explicit yes.

⚠️ `/api/health` reports a hardcoded version string and cannot tell you which build is deployed
— check behaviour, not the version field.

### Built
- ✅ Infrastructure — Docker, Nginx, SSL, CI/CD with per-branch auto-deploy
- ✅ Auth — email/password + Telegram, JWT, admin roles
- ✅ Calendar — week/day grid, drag to create/move/resize, recurring events (RRULE) with
  exceptions, all-day and multi-day events, timezone-aware
- ✅ Tasks — CRUD, priorities/contexts/energy, drag-to-schedule onto the calendar
- ✅ Reflections, Planning, Kanban, Eisenhower, time-blocking
- ✅ Pomodoro focus timer with cross-page widget and time logging
- ✅ Onboarding & contextual help — first-run guidance, per-page help, 3 hint styles
- ✅ Export / import, feedback system, admin panel
- ✅ Bilingual UI (en + ru), Telegram bot

### Known Broken

- ❌ Mobile calendar views (3-day, agenda, mini-month) — `MobileCalendar/` does not exist;
  only swipe-day navigation is built, despite v0.4.5 being counted as closed
- ❌ Assistant bot does not authenticate — `AuthToken` is read but never assigned, so `/today`
  and task commands reach the API with an empty token

Fixed and no longer listed here: multi-day move/resize (MD1 and MD2, 10–11.08, proven by real
mouse drags in `web/e2e/`), the mobile day view opening on the week's Monday instead of today,
and the drag-commit repaint flicker.

### Next

🔴 **Updated 2026-08-12.** Nothing in this section is a suggestion to start — it is the current
state. Read [Roadmap](docs/ROADMAP.md) before acting on any of it.

- **P1** quick task capture — ✅ built, not released
- **P2** Telegram notifications — 🟡 works on staging; production needs `SERVICE_TOKEN` in the
  prod bot's `.env` plus `up -d --force-recreate` (a plain `restart` does not reread it)
- **P3** shared calendars — 🟡 spec written, slices 1 and 2 built and on staging: access is
  now decided by calendar membership rather than by a `user_id` column, and calendars can be
  created, renamed and deleted. Invitations (slice 3) are **blocked** on a precondition only
  Denis can clear — merging his two duplicate user rows in the production database

Deferred to backlog: Eisenhower decision helper, Kanban rework.

Status → [Roadmap](docs/ROADMAP.md) · which document to trust → [DOCS-MAP](docs/DOCS-MAP.md) ·
what P3 slice 2 left behind → `docs/superpowers/plans/2026-08-11-p3-slice2-inherited-debt.md`

---

## Contributing

This is currently a solo project, but issues and suggestions are welcome!

1. Check the [Roadmap](docs/ROADMAP.md) for planned work
2. Open an issue to discuss before starting
3. Follow the conventions in [CLAUDE.md](CLAUDE.md)

Work happens on `develop` (auto-deploys to staging), then merges to `main` for release.

---

## License

Private project. All rights reserved.

---

*Built with ❤️ for the neurodivergent community*