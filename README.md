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

- [Roadmap](docs/ROADMAP.md) — **status source of truth**: versions, bug registry, what's next
- [Progress](PROGRESS.md) — verified snapshot of what's built
- [Codebase Map](docs/CODEBASE_MAP.md) — architecture and module guide
- [CLAUDE.md](CLAUDE.md) — development conventions, commands, gotchas

---

## Current Version

`main` runs **v0.4.9** (tagged 2026-04-24). `develop` holds the unreleased **v0.4.10** candidate.

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
- ❌ Multi-day event move/resize (MD1/MD2) — drag mapping breaks across day columns
- ❌ Mobile calendar views (3-day, agenda, mini-month) — only swipe-day nav exists

### Next
- 📋 Fix multi-day event move/resize
- 📋 Eisenhower decision helper, Kanban rework — see [Roadmap](docs/ROADMAP.md)

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