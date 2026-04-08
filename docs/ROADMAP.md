# NeuroBoost Roadmap

> Single source of truth for versioning. Updated: 2026-04-08.

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

## Current Sprint: v0.4.4 – v0.4.9

**Priority:** Calendar UX > Mobile > Telegram > Features > Polish

| Version | Theme | Key Deliverables |
|---------|-------|-----------------|
| **v0.4.4** | Calendar UX Fix | Fix click/select/resize/move model, task-to-calendar drag, performance, i18n day names |
| **v0.4.5** | Mobile Polish | 4 mobile calendar views (day/3-day/month/agenda), event editor responsive, fix nav overlaps |
| **v0.4.6** | Telegram Bot + MiniApp | Assistant bot commands, notification bot, MiniApp integration, Telegram WebApp auth |
| **v0.4.7** | New Pages & Tools | Home dashboard, planning page, reflections page, pomodoro timer, kanban board |
| **v0.4.8** | Settings & Cleanup | Global UI scale, feature toggles, export/import, bug cleanup |
| **v0.4.9** | Final Test & Polish | Full regression, performance audit, mobile responsiveness pass |

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
