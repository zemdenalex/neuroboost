---
last_mapped: 2026-01-21T00:00:00Z
inventory_rechecked: 2026-08-14
---

# Codebase Map

> Диаграммы и конвенции — Cartographer, 2026-01-21. Инвентарь — пересчитан вручную 2026-08-14.
>
> ⚠ `total_files` и `total_tokens` убраны из шапки намеренно: их никто не пересчитывал семь
> месяцев, а число, выглядящее измеренным и не являющееся им, хуже отсутствующего.
>
> **Инвентарь пересобран 2026-08-14** — состав пакетов, число миграций, карточка `auth/` и
> список заглушек приведены к факту. Диаграммы, потоки данных и «рецепты» ниже верны и были
> верны всё это время.
>
> 🔴 **Раньше здесь стояло предупреждение «инвентарь устарел, миграций 10».** Его сняли не
> потому, что оно было лишним, а потому, что **оно само протухло**: на момент удаления
> миграций было 12. Приписка о протухании протухает ровно так же, как текст, к которому она
> приписана. Если инвентарь снова разойдётся с кодом — **пересчитать и переписать**, а не
> дописывать вторую приписку. Команды пересчёта — в `CLAUDE.md` §Счётчики.

## System Overview

```mermaid
graph TB
    subgraph Client["Frontend (React)"]
        Web[Web App<br/>React 18 + Vite]
        Router[React Router v6]
        AuthCtx[AuthContext]
        API_Client[API Client]
    end

    subgraph Server["Backend (Go)"]
        Chi[Chi Router]
        JWT[JWT Middleware]
        Handlers[API Handlers]
        DB_Pool[pgx Pool]
    end

    subgraph Data["Data Layer"]
        PG[(PostgreSQL 16)]
        Migrate[golang-migrate]
    end

    subgraph External["External Services"]
        TG[Telegram Auth]
    end

    Web --> Router
    Router --> AuthCtx
    AuthCtx --> API_Client
    API_Client -->|HTTP/JSON| Chi
    Chi --> JWT
    JWT --> Handlers
    Handlers --> DB_Pool
    DB_Pool --> PG
    Migrate --> PG
    Handlers -->|Verify| TG
```

## Directory Structure

```
neuroboost/
├── api-go/                      # Go backend API (module `neuroboost/api-go`)
│   ├── cmd/
│   │   ├── api/main.go         # Entry point; every route is registered here (lines 84-216)
│   │   └── healthcheck/        # Docker health check binary
│   ├── internal/               # 17 packages
│   │   ├── admin/              # Operator health + logs, admin-gated
│   │   ├── auth/               # JWT + Telegram + email auth
│   │   ├── calendars/          # Calendars, membership, access scoping (P3)
│   │   ├── config/             # Environment config loader
│   │   ├── database/           # pgx connection pool
│   │   ├── events/             # Calendar events CRUD + recurrence
│   │   ├── export/             # Export / import
│   │   ├── feedback/           # User feedback; POST is PUBLIC (no JWT)
│   │   ├── logger/             # Structured logging
│   │   ├── middleware/         # JWT validation, request logging, body limits
│   │   ├── planning/           # Week overview  [IMPLEMENTED — not a stub]
│   │   ├── reflections/        # Reflections    [IMPLEMENTED — not a stub]
│   │   ├── reminders/          # Reminder scanner, worker, /api/svc endpoints
│   │   ├── status/             # Health endpoint
│   │   ├── tasks/              # Task management CRUD
│   │   ├── usersettings/       # Leaf package: needed by events, tasks, reminders
│   │   └── util/               # Response envelope helpers
│   └── migrations/             # 12 PostgreSQL migrations
├── bot/                         # Telegram bot — a SEPARATE Go module
│   ├── cmd/main.go             # Update loop; `go build ./...` from api-go never enters here
│   └── internal/               # api, auth, config, format, handlers, keyboards,
│                               # logsafe, notifier, parse, state
├── web/                         # React frontend
│   ├── e2e/                    # 10 Playwright specs, run by CI after deploy-dev
│   ├── eslint.config.mjs       # Added 2026-08-13; `pnpm lint` is a CI gate
│   ├── src/
│   │   ├── api/                # HTTP client + API modules
│   │   ├── components/         # Calendar (WeekGrid, EventEditor), Layout, TaskSidebar, …
│   │   ├── contexts/           # AuthContext, ThemeContext
│   │   ├── lib/                # Pure rules, unit-tested: calendars/, quickTask/, tools/, …
│   │   ├── pages/              # Route components
│   │   ├── hooks/              # Custom React hooks
│   │   └── types/              # TypeScript definitions
│   └── nginx.conf              # SPA routing config
├── graph/                       # Memory graph (memory-graph skill)
├── docs/                        # Documentation
├── scripts/                     # Deployment scripts
├── _legacy/                     # Pre-Go implementation, kept for reference
├── docker-compose.yml
└── .github/workflows/ci.yml     # backend · frontend · docker · deploy · deploy-dev · e2e
```


## Module Guide

### Backend: `api-go/internal/auth/`

**Purpose**: Authentication and user management (JWT + Telegram + email/password)

**Entry point**: `handlers.go`

**Key files**:
| File | Purpose |
|------|---------|
| handlers.go | HTTP handlers (TelegramLogin, Register, Login, Logout, Me, UpdateMe) |
| types.go | User, request/response types |
| validation.go | Locale, timezone and credential validation |
| validation_test.go | Its tests |

⚠ Rechecked 2026-08-14. The three files this card used to list — `telegram.go`, `jwt.go`,
`db.go` — **no longer exist**; HMAC verification and token issuing live in `handlers.go`, and
the only `Claims` struct in the codebase is in `middleware/jwt.go`.

**Exports**: `Handler`, `NewHandler(db, cfg)`, `TelegramLogin`, `Register`, `Login`, `Me`, `UpdateMe`

**Dependencies**: `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`, `jackc/pgx/v5`

**Dependents**: `cmd/api/main.go`

**Gotchas**:
- Telegram auth expires after 24 hours from `auth_date`
- JWT tokens have 30-day expiration
- Timezone validation limited to 6 hardcoded values

---

### Backend: `api-go/internal/events/`

**Purpose**: Calendar event CRUD with move/resize/recurrence support

**Entry point**: `handlers.go`

**Key files**:
| File | Purpose | Tokens |
|------|---------|--------|
| handlers.go | Full CRUD + move/resize/exceptions | 5277 |
| types.go | Event, CreateRequest, UpdateRequest | 834 |

**Exports**: `InitDB(db)`, `ListHandler`, `CreateHandler`, `GetHandler`, `UpdateHandler`, `DeleteHandler`, `MoveHandler`, `ResizeHandler`, `AddExceptionHandler`

**Dependencies**: `go-chi/chi/v5`, `jackc/pgx/v5`, `internal/middleware`, `internal/util`

**Patterns**:
- Package-level DB variable (call `InitDB()` before use)
- Dynamic SQL query building for partial updates
- RFC3339 date parsing

**Gotchas**:
- Default timezone: "Europe/Moscow"
- Tags/contexts default to `[]`, never `null`
- `is_work_event` defaults to `true`
- Recurring events use iCalendar RRULE format

---

### Backend: `api-go/internal/tasks/`

**Purpose**: Task management with scheduling to calendar

**Entry point**: `handlers.go`

**Key files**:
| File | Purpose | Tokens |
|------|---------|--------|
| handlers.go | CRUD + schedule-to-event | 4315 |
| types.go | Task, TaskStatus, TaskCategory | 824 |

**Exports**: `InitDB(db)`, `ListHandler`, `CreateHandler`, `GetHandler`, `UpdateHandler`, `DeleteHandler`, `ScheduleHandler`

**Patterns**:
- Status changes trigger side effects (DONE sets `completed_at`)
- `ScheduleHandler` creates event AND updates task status to SCHEDULED

**Gotchas**:
- Priority 1 = Emergency (highest), Priority 5 = If Possible (lowest), Priority 0 = Buffer
- Contexts are GTD-style (`@home`, `@work`, `@computer`)
- Energy scale: 1-5

---

### Frontend: `web/src/api/client.ts`

**Purpose**: Core HTTP client with token management

**Key exports**:
- Token: `getStoredToken()`, `setStoredToken()`, `clearStoredToken()`, `isTokenExpired()`
- HTTP: `api.get()`, `api.post()`, `api.patch()`, `api.delete()`
- Inline APIs: Events, Tasks, Reflections (duplicated from separate modules)

**Patterns**:
- Automatic 401 redirect to `/login` with token clearing
- Response unwrapping (handles `{data: T}` and bare `T`)
- Token stored in localStorage (`nb_token`, `nb_token_expiry`)

**Gotchas**:
- API base URL from `VITE_API_URL` (defaults to `/api`)
- 204 responses return `{}`
- **Contains duplicate implementations** - Events/tasks APIs exist both here and in separate modules

---

### Frontend: `web/src/contexts/AuthContext.tsx`

**Purpose**: Global authentication state and session management

**Exports**: `AuthProvider`, `useAuthContext()`, `useRequireAuth()`, `useRequireAdmin()`

**State**:
- `user`, `loading`, `isAuthenticated`
- Settings synced to localStorage for non-React consumers

**Patterns**:
- Async session restoration on mount (checks token, fetches user)
- Custom DOM events for cross-component updates (`neuroboost-layout-change`)

**Gotchas**:
- Token expiry stored as Unix seconds, compared with `Date.now()` (milliseconds)
- No automatic token refresh - user logged out on expiry

---

### Frontend: `web/src/components/Calendar/WeekGrid/`

**Purpose**: Interactive weekly calendar with drag-and-drop

**Key files** (17 total):
| File | Purpose | Tokens |
|------|---------|--------|
| WeekGrid.tsx | Main orchestrator | 1897 |
| DayColumn.tsx | Single day rendering | 1688 |
| AllDaySection.tsx | All-day events | 1699 |
| GhostPreview.tsx | Drag feedback | 1776 |
| useWeekGridDrag.ts | Drag state management | 1246 |
| weekgrid.utils.ts | Position/time conversions | 1347 |
| useKeyboardNav.ts | Keyboard shortcuts | 826 |

**Props**:
```typescript
{
  events: NbEvent[]
  timezone: string
  onCreate: (data) => void
  onMoveOrResize: (data) => void
  onSelect: (event) => void
  onTaskDrop?: (task, startTime) => void
}
```

**Patterns**:
- Multi-day events split into segments with visual continuity
- Timezone-aware rendering (UTC storage, local display)
- Responsive: 1 day (mobile), 3 days (tablet), 7 days (desktop)
- Task drop from sidebar via `dataTransfer` JSON

**Gotchas**:
- Week starts Monday (ISO standard)
- `HOUR_PX = 44`, `MIN_SLOT_MIN = 15` (snap interval)
- Keyboard: Arrow keys navigate, Shift+Arrow moves events

---

### Frontend: `web/src/components/Calendar/EventEditor/`

**Purpose**: Modal form for event creation/editing with reflections

**Key files** (10 total):
| File | Purpose | Tokens |
|------|---------|--------|
| EventEditor.tsx | Main form orchestrator | 1089 |
| useEditorForm.ts | 20+ state fields, validation | 1992 |
| editor.utils.ts | UTC/local conversions | 1321 |
| DateTimeFields.tsx | Date/time inputs | 712 |
| ReflectionFields.tsx | Post-event metrics | 744 |

**Patterns**:
- Flexible time input parser (handles "1050", "10:50", "9:30")
- Cross-midnight event detection
- Conditional UI (advanced toggle, reflection only for editing)

**Gotchas**:
- All UTC/local conversions in `editor.utils.ts`
- Enter to save, Ctrl+Enter in description textarea

---

### Frontend: `web/src/router.tsx`

**Purpose**: Application routing configuration

**Routes**:
- Public: `/login`, `/home`
- Protected: `/calendar`, `/tasks`, `/planning`, `/reflections`, `/tools`, `/settings`, `/profile`, `/admin`
- Default: `/` redirects to `/home`

**Patterns**:
- `ProtectedRoute` wrapper checks `isAuthenticated`
- `PublicRoute` redirects authenticated users to `/home`

---

## Data Flow

### Authentication Flow

```mermaid
sequenceDiagram
    participant User
    participant Login Page
    participant AuthContext
    participant API Client
    participant Go API
    participant PostgreSQL

    User->>Login Page: Enter credentials
    Login Page->>AuthContext: loginWithEmail(email, pwd)
    AuthContext->>API Client: POST /api/auth/login
    API Client->>Go API: HTTP request
    Go API->>PostgreSQL: Verify user
    PostgreSQL-->>Go API: User record
    Go API->>Go API: bcrypt.Compare()
    Go API->>Go API: Generate JWT (30d)
    Go API-->>API Client: {token, user}
    API Client->>API Client: Store in localStorage
    API Client-->>AuthContext: Success
    AuthContext->>AuthContext: Set user state
    AuthContext-->>Login Page: Redirect to /calendar
```

### Task Scheduling Flow

```mermaid
sequenceDiagram
    participant User
    participant TaskSidebar
    participant WeekGrid
    participant Calendar Page
    participant API
    participant DB

    User->>TaskSidebar: Drag task
    TaskSidebar->>TaskSidebar: setData('application/json', task)
    User->>WeekGrid: Drop on time slot
    WeekGrid->>WeekGrid: Calculate drop time
    WeekGrid->>Calendar Page: onTaskDrop(task, time)
    Calendar Page->>API: scheduleTask(id, startTime, duration)
    API->>DB: INSERT event, UPDATE task.status='SCHEDULED'
    DB-->>API: Success
    API-->>Calendar Page: ScheduledEvent
    Calendar Page->>Calendar Page: Reload events + tasks
    Calendar Page-->>User: Show event on calendar
```

### Event CRUD Flow

```mermaid
sequenceDiagram
    participant User
    participant WeekGrid
    participant EventEditor
    participant API
    participant DB

    User->>WeekGrid: Click to create / select event
    WeekGrid->>EventEditor: Open modal (range or draft)
    User->>EventEditor: Fill form
    EventEditor->>EventEditor: Validate (real-time)
    User->>EventEditor: Save
    EventEditor->>API: createEvent() or updateEvent()
    API->>DB: INSERT/UPDATE
    DB-->>API: Event record
    API-->>EventEditor: Success
    EventEditor->>EventEditor: Close modal
    EventEditor-->>WeekGrid: Trigger reload
```

## Conventions

### Backend
- **Handler pattern**: Each package exports HTTP handlers
- **Package-level DB**: Events/tasks use `InitDB(db)` instead of constructor injection
- **Dynamic SQL**: All updates build query based on provided fields
- **Response envelope**: `{data, error, meta}` via `util.RespondJSON/RespondError`
- **Pointer fields**: Optional fields are pointers for nullability

### Frontend
- **Context API**: Global state via AuthContext, ThemeContext
- **Type conversion**: API (snake_case) ↔ Frontend (camelCase) via `toNbEvent()`
- **Loading states**: Spinner during async operations
- **Modal pattern**: Fixed overlay, click-outside close, Escape key

### Naming
- **Go files**: lowercase with underscores (`handlers.go`, `db.go`)
- **React components**: PascalCase (`EventEditor.tsx`)
- **React hooks**: camelCase with `use` prefix (`useEditorForm.ts`)
- **TypeScript types**: PascalCase (`NbEvent`, `Task`)

## Gotchas

### Critical

⚠ All four entries that stood here were rechecked on 2026-08-14 and **every one had been fixed**
— duplicate `toNbEvent()` (one definition now, `types/index.ts:96`), dual `Claims` structs (one,
`middleware/jwt.go:18`), legacy auth files (deleted), and the token-expiry mismatch
(`client.ts:47` multiplies by 1000). A stale warning is worse than none: it spends the reader's
attention on a defect that is not there, and teaches them to distrust the rest of the list.

The live ones live in `CLAUDE.md` §Gotchas, which is injected into every session and is
therefore the only list worth keeping current. Do not copy them here — one list, one place.

### Non-obvious
- Default timezone throughout: "Europe/Moscow"
- Priority inversion: Lower number = higher priority (1=Emergency, 5=If Possible)
- Task scheduling creates event but doesn't delete task
- All arrays default to `[]`, never `null`
- Week starts Monday (ISO standard)

### Stubs (Return 501)

**None.** Rechecked 2026-08-14: `grep -rn "501" api-go/internal/` finds only the `Write501`
helper itself.

`needs`, `opportunities` and `patterns` were the last three, and they were deleted on 2026-08-14
— ten routes answering 501 to an authenticated caller, with no writer anywhere in the backend
and a one-line stub for every frontend piece. Their tables remain (migrations are never edited
in this project, and `node_kind` still references them); git holds the packages.

`planning` and `reflections` were listed here as stubs and **were never stubs** — both are fully
implemented.

## Navigation Guide

**To add a new API endpoint**:
1. Add route in `api-go/cmd/api/main.go` — inside the JWT group (`r.Group` after the public
   one), or the public group above it if the route needs no auth. Deliberately no line number:
   the last one printed here was 70 lines out of date.
2. Create handler in `api-go/internal/{module}/handlers.go`
3. Add types in `api-go/internal/{module}/types.go`
4. Add API function in `web/src/api/{module}.ts`

**To add a new page**:
1. Create page component in `web/src/pages/{Page}/{Page}.tsx`
2. Add route in `web/src/router.tsx`
3. Add navigation item in `web/src/components/Layout/Header/HorizontalHeader.tsx` (line ~30)

**To add a new migration**:
1. Create `api-go/migrations/NNNNNN_name.up.sql` and `.down.sql`
2. Migrations run automatically on API container startup

**To modify auth**:
1. Backend: `api-go/internal/auth/handlers.go`
2. Middleware: `api-go/internal/middleware/jwt.go`
3. Frontend: `web/src/contexts/AuthContext.tsx`
4. API client: `web/src/api/auth.ts`

**To add a calendar feature**:
1. WeekGrid: `web/src/components/Calendar/WeekGrid/`
2. EventEditor: `web/src/components/Calendar/EventEditor/`
3. Calendar page: `web/src/pages/Calendar/Calendar.tsx`

**To add a task feature**:
1. Backend: `api-go/internal/tasks/handlers.go`
2. Frontend API: `web/src/api/tasks.ts`
3. TaskSidebar: `web/src/components/TaskSidebar/`
4. Tasks page: `web/src/pages/Tasks/Tasks.tsx`
