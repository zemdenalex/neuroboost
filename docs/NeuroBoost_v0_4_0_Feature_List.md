<!-- паспорт: тип=документ | статус=архив | строк=339 | ~токенов=3035 | обновлён=по git -->

# NeuroBoost v0.4.0 Feature List

> ## ⚠️ HISTORICAL DOCUMENT — DO NOT READ STATUS FROM THIS FILE
>
> This is the **original v0.4.0 planning document** (Dec 2025 – Mar 2026). It is kept for the
> feature inventory and the original prioritisation, **not** as a status record.
>
> **Its per-row statuses are wrong across the board.** Verified against the codebase on
> 2026-07-19: it marks the entire Events API, Tasks API, WeekGrid, EventEditor, TaskSidebar,
> admin panel and feedback system as 📋 *planned* — all of them shipped in v0.4.2 → v0.4.9.
> The "17 of 117 done" summary predates six released versions. Recurring events (rrule) are
> listed as planned but are built and tested (`api-go/internal/events/recurrence.go`).
>
> 👉 **For real status, read [`ROADMAP.md`](ROADMAP.md) and [`../PROGRESS.md`](../PROGRESS.md).**
>
> ---
>
> Priority: 🔴 Critical | 🟠 High | 🟡 Medium | 🟢 Low | ⚪ Future
> Status: ✅ Done | 🔧 In Progress | 📋 Planned | 💭 Idea
> Last Updated: March 13, 2026 (statuses frozen — superseded by ROADMAP.md)

---

## Phase 0: Foundation (Current)

### Infrastructure
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| Server setup (Ubuntu 22.04) | 🔴 | ✅ | 62.76.228.106 |
| Docker + Docker Compose | 🔴 | ✅ | Running |
| Nginx + SSL | 🔴 | ✅ | Certbot done |
| PostgreSQL container | 🔴 | ✅ | Healthy |
| Database migrations (golang-migrate) | 🔴 | ✅ | 000001, 000002 |
| Health check endpoint | 🔴 | ✅ | /api/health with DB check |
| fail2ban + SSH hardening | 🔴 | ✅ | 4 IPs banned |
| CI: Build & test | 🟠 | ✅ | GitHub Actions (.github/workflows/ci.yml) |
| CI: Auto-deploy on merge | 🟠 | 📋 | |
| CI: Database backups | 🟠 | 📋 | Daily cron |
| Error logging | 🟠 | 📋 | Structured JSON logs |

### Authentication
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| User table with auth fields | 🔴 | ✅ | Migration 000002 |
| Telegram Login Widget | 🔴 | 🔧 | Backend ready, need frontend |
| Email/Password registration | 🔴 | ✅ | POST /api/auth/register |
| Email/Password login | 🔴 | ✅ | POST /api/auth/login |
| JWT token generation | 🔴 | ✅ | 30-day tokens |
| JWT middleware (Go) | 🔴 | ✅ | Protects API routes |
| GET /api/auth/me | 🔴 | ✅ | Returns current user |
| Login page (frontend) | 🔴 | 📋 | Next up! |
| Session refresh banner | 🟡 | 📋 | "Session expires in X days" |
| Password reset (email) | 🟠 | 📋 | |
| Email verification | 🟡 | 📋 | v0.4.1.x |
| Google OAuth | 🟢 | 💭 | Future auth provider |

---

## Phase 1: Admin & Feedback (NEW - High Priority!)

### Admin Panel
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| Admin dashboard page | 🔴 | 📋 | Overview stats |
| Admin authentication | 🔴 | 📋 | Role-based (is_admin flag) |
| User management | 🟠 | 📋 | View/search users |
| Health status view | 🟠 | 📋 | Services, DB, API |
| Feedback/bug reports view | 🔴 | 📋 | Triage incoming reports |
| IP ban management | 🟡 | 📋 | View/add/remove bans |
| System logs viewer | 🟡 | 📋 | Recent errors |
| Basic analytics | 🟡 | 📋 | Active users, events created |

### Feedback System
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| Feedback button (all pages) | 🔴 | 📋 | Fixed position FAB |
| Bug report form | 🔴 | 📋 | Type, description, steps |
| Feature suggestion form | 🔴 | 📋 | Title, description, priority |
| Screenshot attachment | 🟡 | 📋 | Optional |
| Auto-capture context | 🟠 | 📋 | Page, browser, user ID |
| POST /api/feedback | 🔴 | 📋 | Store in DB |
| GET /api/admin/feedback | 🔴 | 📋 | List all feedback |
| PATCH /api/admin/feedback/:id | 🟠 | 📋 | Update status |
| Feedback table (DB) | 🔴 | 📋 | Migration needed |
| GitHub issue auto-create | 🟢 | 💭 | Optional integration |
| Telegram notification | 🟠 | 📋 | Notify on new feedback |

---

## Phase 2: Core API

### Events
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| GET /api/events (list) | 🔴 | 📋 | Date range filter |
| POST /api/events (create) | 🔴 | 📋 | |
| PATCH /api/events/:id | 🔴 | 📋 | |
| DELETE /api/events/:id | 🔴 | 📋 | |
| PATCH /api/events/:id/move | 🔴 | 📋 | Drag & drop |
| PATCH /api/events/:id/resize | 🔴 | 📋 | |
| Recurring events (rrule) | 🟠 | 📋 | iCal format |
| Event exceptions | 🟠 | 📋 | Skip/modify occurrence |
| Multi-day events | 🟠 | 📋 | |
| All-day events | 🔴 | 📋 | |
| Calendar layers | 🟡 | 📋 | Work/Personal/Health |

### Tasks
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| GET /api/tasks (list) | 🔴 | 📋 | With filters |
| POST /api/tasks | 🔴 | 📋 | |
| PATCH /api/tasks/:id | 🔴 | 📋 | |
| DELETE /api/tasks/:id | 🔴 | 📋 | |
| POST /api/tasks/:id/schedule | 🔴 | 📋 | Convert to event |
| PATCH /api/tasks/bulk | 🟠 | 📋 | Bulk status update |
| Task priorities (0-5) | 🔴 | 📋 | |
| Task categories | 🟠 | 📋 | EMERGENCY→BUFFER |
| Task contexts (@home, @work) | 🟡 | 📋 | v0.4.x feature |
| Task energy levels | 🟡 | 📋 | 1-5 scale |
| Task dependencies | 🟢 | 📋 | |
| Subtasks (parent/child) | 🟡 | 📋 | |
| Time windows | 🟢 | 📋 | earliest/latest/ideal |
| Task aging/bumps | 🟢 | 📋 | Postpone tracking |

### Reflections & Stats
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| POST /api/reflections | 🟠 | 📋 | Focus/Goal/Mood |
| GET /api/reflections | 🟠 | 📋 | |
| GET /api/stats/week | 🟠 | 📋 | Weekly adherence |
| GET /api/stats/adherence | 🟡 | 📋 | Plan vs actual |
| Work hours tracking | 🟢 | 📋 | WorkHoursLog |

---

## Phase 3: Frontend

### Calendar (Port from v0.3.x)
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| WeekGrid component | 🔴 | 📋 | Main calendar view |
| Drag to create events | 🔴 | 📋 | |
| Drag to move events | 🔴 | 📋 | |
| Resize events (bottom handle) | 🔴 | 📋 | |
| Current time indicator | 🔴 | 📋 | Red line |
| All-day section | 🔴 | 📋 | Top of grid |
| 15-minute snap grid | 🔴 | 📋 | |
| Ghost preview on drag | 🟠 | 📋 | |
| MonthView component | 🟡 | 📋 | Overview |
| Week navigation | 🔴 | 📋 | Prev/Next/Today |
| Keyboard shortcuts | 🟡 | 📋 | |

### Task Management
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| TaskSidebar component | 🔴 | 📋 | Priority groups |
| Drag task to calendar | 🔴 | 📋 | Schedule task |
| Task list view | 🟠 | 📋 | Separate page |
| Quick task completion | 🔴 | 📋 | Checkbox toggle |
| Task filtering | 🟠 | 📋 | By status/priority |
| DeadlineTasks timeline | 🟡 | 📋 | Below calendar |

### Event Editor
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| EventEditor modal | 🔴 | 📋 | |
| Title, time inputs | 🔴 | 📋 | |
| All-day toggle | 🔴 | 📋 | |
| Recurrence settings | 🟡 | 📋 | |
| Color picker | 🟡 | 📋 | |
| Reflection sliders | 🟠 | 📋 | Focus/Goal/Mood |

### Layout & Navigation
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| Replace MUI → Lucide icons | 🔴 | ✅ | Done in skeleton |
| Clean URLs (not hash) | 🔴 | ✅ | React Router |
| HorizontalHeader | 🔴 | ✅ | Skeleton exists |
| VerticalSidebar | 🟠 | ✅ | Skeleton exists |
| Mobile responsive | 🟠 | 📋 | |
| Dark theme (default) | 🔴 | ✅ | Zinc palette |
| Light theme toggle | 🟢 | 💭 | |

---

## Phase 4: Telegram Bot

### Assistant Bot (@NeuroBoost_assistant_bot)
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| /start command | 🔴 | 📋 | Welcome + keyboard |
| /help command | 🔴 | 📋 | |
| /tasks command | 🔴 | 📋 | List by priority |
| /newtask wizard | 🟠 | 📋 | Step-by-step |
| /today command | 🔴 | 📋 | Today's schedule |
| /week command | 🟡 | 📋 | Week overview |
| /note command | 🟠 | 📋 | Quick capture |
| /stats command | 🟡 | 📋 | Weekly adherence |
| /settings command | 🟡 | 📋 | |
| /feedback command | 🟠 | 📋 | Report bug/suggestion |
| Persistent reply keyboard | 🔴 | 📋 | |
| Inline keyboards | 🟠 | 📋 | Task actions |
| MiniApp button | 🟠 | 📋 | Open web UI |

### Notification Bot (@NeuroBoost_notifications_bot)
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| Event reminders | 🔴 | 📋 | 30/10/5 min before |
| Daily planning nudge | 🟠 | 📋 | 21:00 |
| Weekly planning nudge | 🟠 | 📋 | Sunday 18:00 |
| Task deadline alerts | 🟠 | 📋 | |
| Quiet hours | 🟠 | 📋 | 22:00-08:00 default |
| Rate limiting | 🟠 | 📋 | ~1 msg/min |
| Snooze options | 🟡 | 📋 | |
| New feedback notification | 🟠 | 📋 | Alert admin |

---

## Phase 5: Polish & Gamification

### Gamification (Basic)
| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| XP system | 🟡 | 📋 | Points for tasks |
| Levels | 🟡 | 📋 | Based on XP |
| Streaks | 🟡 | 📋 | Daily activity |
| Basic badges | 🟡 | 📋 | First task, first week |
| Profile stats display | 🟡 | 📋 | |
| Leaderboard | ⚪ | 💭 | Optional social |

---

## Infrastructure Decisions

### Internationalization (i18n)
| Approach | Priority | Status | Notes |
|----------|----------|--------|-------|
| URL-based: /ru/, /en/ | 🟡 | 💭 | Simple, SEO-friendly |
| Cookie/localStorage | 🟡 | 💭 | Cleaner URLs |
| User preference (DB) | 🟡 | 💭 | Requires auth |
| Browser Accept-Language | 🟡 | 💭 | Auto-detect |

**Recommended approach:**
1. Default: Browser language detection
2. Store preference in localStorage (guest)
3. Store preference in user DB (logged in)
4. URL override: ?lang=ru or /ru/ prefix
5. No m. subdomain initially

### Mobile Strategy
| Approach | Priority | Status | Notes |
|----------|----------|--------|-------|
| Responsive web (CSS) | 🔴 | 📋 | Must have |
| PWA (installable) | 🟡 | 💭 | Add to homescreen |
| Telegram MiniApp | 🟠 | 📋 | Built-in to TG |
| Native apps | ⚪ | 💭 | v1.0+ |

---

## Implementation Order (Recommended)

### Sprint 1: Auth Complete (Current)
1. ✅ Auth backend (done)
2. 📋 Login page frontend
3. 📋 Auth context integration
4. 📋 Protected routes

### Sprint 2: Feedback & Admin Foundation
1. 📋 Feedback table migration
2. 📋 Feedback API endpoints
3. 📋 Feedback button component
4. 📋 Admin page (basic)
5. 📋 Admin auth (is_admin flag)

### Sprint 3: Events CRUD
1. 📋 Events API endpoints
2. 📋 WeekGrid implementation
3. 📋 EventEditor modal
4. 📋 Drag & drop

### Sprint 4: Tasks CRUD
1. 📋 Tasks API endpoints
2. 📋 TaskSidebar implementation
3. 📋 TaskEditor modal
4. 📋 Task → Event scheduling

### Sprint 5: Bot & Notifications
1. 📋 Assistant bot commands
2. 📋 Notification bot reminders
3. 📋 Telegram Login Widget

---

## Quick Stats

| Category | Total | 🔴 Critical | 🟠 High | 🟡 Medium | ✅ Done |
|----------|-------|-------------|---------|-----------|---------|
| Infrastructure | 11 | 6 | 4 | 0 | 7 |
| Auth | 12 | 8 | 2 | 2 | 6 |
| Admin Panel | 8 | 2 | 3 | 3 | 0 |
| Feedback | 11 | 4 | 4 | 2 | 0 |
| Events API | 11 | 6 | 4 | 1 | 0 |
| Tasks API | 14 | 5 | 3 | 4 | 0 |
| Frontend | 25 | 14 | 6 | 4 | 4 |
| Bot | 19 | 4 | 10 | 4 | 0 |
| Gamification | 6 | 0 | 0 | 5 | 0 |
| **TOTAL** | **117** | **49** | **36** | **25** | **17** |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| v0.4.0.0 | Dec 24, 2025 | Initial skeleton |
| v0.4.0.1 | Dec 25, 2025 | Server deployment |
| v0.4.0.2 | Jan 3, 2026 | Auth system foundation |
| v0.4.0.3 | Mar 13, 2026 | Claude Code setup (skills, rules, subagent, permissions) |

---

## Claude Code Setup (March 2026)

| Component | Files | Purpose |
|-----------|-------|---------|
| CLAUDE.md | `CLAUDE.md` | Project context, rules, gotchas (~80 lines) |
| Skills | `.claude/skills/deploy/` | `/deploy` — production deployment workflow |
| Skills | `.claude/skills/add-endpoint/` | `/add-endpoint` — new API endpoint workflow |
| Skills | `.claude/skills/add-page/` | `/add-page` — new frontend page workflow |
| Skills | `.claude/skills/add-migration/` | `/add-migration` — database migration workflow |
| Subagent | `.claude/agents/nb-code-reviewer.md` | Security + quality code review |
| Rules | `.claude/rules/go-backend.md` | Go conventions (loads for `api-go/**`) |
| Rules | `.claude/rules/react-frontend.md` | React conventions (loads for `web/**`) |
| Hooks | Inherited from root `E:\Projects\.claude\` | Block .env edits, auto-format on save |
| Permissions | `.claude/settings.local.json` | 15 pre-approved commands |

---

*Last updated: March 13, 2026*
