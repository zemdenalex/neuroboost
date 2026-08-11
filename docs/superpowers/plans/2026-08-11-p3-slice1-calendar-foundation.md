# P3 срез 1 — фундамент календарей Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ввести календарь как контейнер доступа и перевести на него все запросы событий и задач, **не меняя ничего видимого пользователю**.

**Architecture:** Появляются таблицы `calendar` и `calendar_member`. Каждому существующему пользователю заводится личный календарь, все его события и задачи в него переезжают. Право доступа перестаёт давать колонка `user_id` и начинает давать членство в календаре. Единственный санкционированный способ ограничить выборку — helper `calendars.CalendarIDsFor`, и это охраняется тестом, читающим исходники handlers.

**Tech Stack:** Go 1.22 (chi, pgx/v5) в `api-go/`, PostgreSQL 16, golang-migrate, React 18 + TS + Vite в `web/`, Playwright для e2e.

## Global Constraints

- Спека: `docs/superpowers/specs/2026-08-11-p3-shared-calendars-design.md`. Расхождение с ней — дефект плана, а не свобода исполнителя.
- Следующий номер миграции — **000012**. Существующие миграции не править никогда, только добавлять новые.
- Параметризованные запросы pgx, никогда склейка строк для SQL.
- Никакого `any` в TypeScript.
- Полная проверка до «готово»: `cd api-go && go build ./... && go vet ./... && go test ./...` · то же в `bot/` (отдельный модуль) · `cd web && corepack pnpm typecheck && corepack pnpm test --run && corepack pnpm build`.
- Правка UI → e2e против **задеплоенного** staging: push → дождаться CI → `corepack pnpm exec playwright test`. Базовая линия на 11.08 — **24 passed, 3 skipped**.
- `git add` конкретных файлов, никогда `-A`. Никогда со-автор Claude/Anthropic.
- 🔴 Мерж в `main` = релиз прода. Без явного «да» Дениса не делать.
- 🔴 Внешне поведение среза 1 не меняется. Если после него список событий выглядит иначе — это дефект, а не фича.

## File Structure

| Файл | Ответственность |
|---|---|
| `api-go/migrations/000012_calendars.up.sql` (создать) | Таблицы, колонки, бэкфилл, индексы, `NOT NULL` |
| `api-go/migrations/000012_calendars.down.sql` (создать) | Откат, не теряющий смещения напоминаний |
| `api-go/internal/calendars/access.go` (создать) | Чистое правило доступа: какие членства дают чтение |
| `api-go/internal/calendars/access_test.go` (создать) | Тесты правила — без базы |
| `api-go/internal/calendars/store.go` (создать) | `InitDB`, `CalendarIDsFor` — тонкий слой над pgx |
| `api-go/internal/calendars/scoping_test.go` (создать) | Архитектурный тест: handlers не ограничивают выборку по `user_id` |
| `api-go/internal/events/handlers.go` (править) | 9 функций: `listEvents`, `getEvent`, `createEvent`, `updateEvent`, `deleteEvent`, `moveEvent`, `resizeEvent`, `ResizeHandler`, `addException` |
| `api-go/internal/tasks/handlers.go` (править) | 7 функций: `listTasks`, `getTask`, `createTask`, `updateTask`, `deleteTask`, `scheduleTask`, `logTaskTime` |
| `api-go/cmd/api/main.go` (править) | `calendars.InitDB(db)` рядом с прочими |

Разделение `access.go` / `store.go` не косметическое: в проекте **нет харнесса для тестов с базой** — все Go-тесты чисто функциональные. Правило доступа обязано быть тестируемым, поэтому оно живёт отдельно от запроса.

---

### Task 1: Миграция 000012

**Files:**
- Create: `api-go/migrations/000012_calendars.up.sql`
- Create: `api-go/migrations/000012_calendars.down.sql`

**Interfaces:**
- Consumes: существующие таблицы `"user"`, `event`, `task`.
- Produces: таблицы `calendar`, `calendar_member`; колонки `event.calendar_id`, `task.calendar_id` (обе `NOT NULL` после бэкфилла).

- [ ] **Step 1: Написать up-миграцию**

```sql
-- api-go/migrations/000012_calendars.up.sql
--
-- Календарь как контейнер доступа. До этой миграции user_id отвечал сразу за
-- две вещи: кто владелец строки и кому её видно. Здесь они расходятся:
-- доступ даёт членство в календаре, user_id остаётся авторством.

CREATE TABLE IF NOT EXISTS calendar (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    color      TEXT,
    kind       TEXT NOT NULL CHECK (kind IN ('personal','shared')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS calendar_member (
    calendar_id UUID NOT NULL REFERENCES calendar(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES "user"(id)   ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('owner','editor','viewer')),
    -- Зарезервировано под режим «занят/свободен». В срезе 1 всегда 'full'.
    visibility  TEXT NOT NULL DEFAULT 'full'   CHECK (visibility IN ('full','busy')),
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('invited','active')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (calendar_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_calendar_member_user ON calendar_member (user_id, status);

ALTER TABLE event ADD COLUMN IF NOT EXISTS calendar_id UUID REFERENCES calendar(id);
ALTER TABLE task  ADD COLUMN IF NOT EXISTS calendar_id UUID REFERENCES calendar(id);

-- Личный календарь каждому существующему пользователю.
INSERT INTO calendar (owner_id, name, kind)
SELECT u.id, 'Мой календарь', 'personal'
FROM "user" u
WHERE NOT EXISTS (
    SELECT 1 FROM calendar c WHERE c.owner_id = u.id AND c.kind = 'personal'
);

INSERT INTO calendar_member (calendar_id, user_id, role, status)
SELECT c.id, c.owner_id, 'owner', 'active'
FROM calendar c
WHERE c.kind = 'personal'
ON CONFLICT DO NOTHING;

-- Всё существующее переезжает в личный календарь автора.
UPDATE event e
SET calendar_id = c.id
FROM calendar c
WHERE c.owner_id = e.user_id AND c.kind = 'personal' AND e.calendar_id IS NULL;

UPDATE task t
SET calendar_id = c.id
FROM calendar c
WHERE c.owner_id = t.user_id AND c.kind = 'personal' AND t.calendar_id IS NULL;

ALTER TABLE event ALTER COLUMN calendar_id SET NOT NULL;
ALTER TABLE task  ALTER COLUMN calendar_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_event_calendar_time ON event (calendar_id, starts_at);
CREATE INDEX IF NOT EXISTS idx_task_calendar       ON task  (calendar_id);
```

- [ ] **Step 2: Написать down-миграцию**

```sql
-- api-go/migrations/000012_calendars.down.sql
DROP INDEX IF EXISTS idx_task_calendar;
DROP INDEX IF EXISTS idx_event_calendar_time;

ALTER TABLE task  ALTER COLUMN calendar_id DROP NOT NULL;
ALTER TABLE event ALTER COLUMN calendar_id DROP NOT NULL;

ALTER TABLE task  DROP COLUMN IF EXISTS calendar_id;
ALTER TABLE event DROP COLUMN IF EXISTS calendar_id;

DROP INDEX IF EXISTS idx_calendar_member_user;
DROP TABLE IF EXISTS calendar_member;
DROP TABLE IF EXISTS calendar;
```

- [ ] **Step 3: Применить на staging и проверить, что ничего не осиротело**

```bash
ssh root@62.76.228.106 "docker exec neuroboost-dev-db psql -U neuroboost -d neuroboost_dev -t -A \
  -c \"select 'events without calendar: '||count(*) from event where calendar_id is null\" \
  -c \"select 'tasks without calendar: '||count(*)  from task  where calendar_id is null\" \
  -c \"select 'users without personal calendar: '||count(*) from \\\"user\\\" u where not exists (select 1 from calendar c where c.owner_id=u.id and c.kind='personal')\""
```

Ожидается: **три нуля**. Миграции применяются автоматически на старте API, поэтому проверка идёт после деплоя.

- [ ] **Step 4: Commit**

```bash
git add api-go/migrations/000012_calendars.up.sql api-go/migrations/000012_calendars.down.sql
git commit -m "feat(db): calendar as the access container, with backfill"
```

---

### Task 2: Правило доступа как чистая функция

**Files:**
- Create: `api-go/internal/calendars/access.go`
- Test: `api-go/internal/calendars/access_test.go`

**Interfaces:**
- Produces: `calendars.Membership{CalendarID, Status string}`, `calendars.AccessibleIDs([]Membership) []string`, константы `calendars.StatusInvited`, `calendars.StatusActive`, `calendars.RoleOwner`, `calendars.RoleEditor`, `calendars.RoleViewer`.

- [ ] **Step 1: Написать падающий тест**

```go
// api-go/internal/calendars/access_test.go
package calendars

import "testing"

func TestAccessibleIDsKeepsOnlyActiveMemberships(t *testing.T) {
	// 🔴 Приглашённый не видит ничего до принятия. Если это правило ошибётся,
	// человек увидит чужой календарь, которого ещё не принимал.
	got := AccessibleIDs([]Membership{
		{CalendarID: "a", Status: StatusActive},
		{CalendarID: "b", Status: StatusInvited},
		{CalendarID: "c", Status: StatusActive},
	})
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("got %v, want [a c]", got)
	}
}

func TestAccessibleIDsOnEmptyInputIsEmptyNotNil(t *testing.T) {
	// Пустой срез уходит в `calendar_id = ANY($1)`. nil там означал бы
	// «нет условия», то есть выдачу чужих строк.
	got := AccessibleIDs(nil)
	if got == nil {
		t.Fatal("AccessibleIDs(nil) returned nil; must be an empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestAccessibleIDsIgnoresUnknownStatus(t *testing.T) {
	// Неизвестный статус трактуется как «нет доступа», а не как «есть».
	got := AccessibleIDs([]Membership{{CalendarID: "a", Status: "revoked"}})
	if len(got) != 0 {
		t.Fatalf("got %v, want empty for an unrecognised status", got)
	}
}
```

- [ ] **Step 2: Прогнать тест и убедиться, что он падает**

Run: `cd api-go && go test ./internal/calendars/`
Expected: FAIL — пакет `calendars` не существует.

- [ ] **Step 3: Написать минимальную реализацию**

```go
// api-go/internal/calendars/access.go
//
// Package calendars owns the answer to one question: which calendars may a
// user read? Everything about events and tasks scopes itself through this.
package calendars

// Статусы членства.
const (
	StatusInvited = "invited"
	StatusActive  = "active"
)

// Роли участника.
const (
	RoleOwner  = "owner"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// Membership is one calendar_member row, reduced to what access depends on.
type Membership struct {
	CalendarID string
	Status     string
}

// AccessibleIDs returns the calendars a user may read.
//
// Pure on purpose: the project has no database test harness, and this is the
// rule that must never be wrong — an error here shows one person another
// person's calendar. Keeping it separate from the query makes it testable.
//
// Returns an empty slice, never nil: the result goes into `calendar_id = ANY($1)`,
// where nil would read as "no condition" instead of "nothing matches".
func AccessibleIDs(memberships []Membership) []string {
	ids := make([]string, 0, len(memberships))
	for _, m := range memberships {
		if m.Status == StatusActive {
			ids = append(ids, m.CalendarID)
		}
	}
	return ids
}
```

- [ ] **Step 4: Прогнать тесты**

Run: `cd api-go && go test ./internal/calendars/ -v`
Expected: PASS, 3 теста.

- [ ] **Step 5: Commit**

```bash
git add api-go/internal/calendars/access.go api-go/internal/calendars/access_test.go
git commit -m "feat(calendars): the access rule, as a pure testable function"
```

---

### Task 3: Тонкий слой над базой

**Files:**
- Create: `api-go/internal/calendars/store.go`
- Modify: `api-go/cmd/api/main.go`

**Interfaces:**
- Consumes: `calendars.AccessibleIDs` из Task 2, `database.DB` (поле `Pool *pgxpool.Pool`).
- Produces: `calendars.InitDB(*database.DB)`, `calendars.CalendarIDsFor(ctx, userID string) ([]string, error)`, `calendars.PersonalIDFor(ctx, userID string) (string, error)`.

- [ ] **Step 1: Написать store.go**

```go
// api-go/internal/calendars/store.go
package calendars

import (
	"context"

	"neuroboost/api-go/internal/database"
)

var db *database.DB

// InitDB sets the database connection for the calendars package.
func InitDB(d *database.DB) { db = d }

// CalendarIDsFor returns every calendar the user may read.
//
// 🔴 This is the ONLY sanctioned way to scope a query for events or tasks. A
// handler that filters by user_id instead is a bug, and scoping_test.go fails
// the build for it.
//
// Filtering happens in Go rather than SQL so the rule lives in AccessibleIDs,
// where it is tested. The row count here is a handful per user.
func CalendarIDsFor(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT calendar_id::text, status FROM calendar_member WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberships := []Membership{}
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.CalendarID, &m.Status); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return AccessibleIDs(memberships), nil
}

// PersonalIDFor returns the user's own calendar, which is where anything
// created without an explicit calendar goes.
func PersonalIDFor(ctx context.Context, userID string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`SELECT id::text FROM calendar WHERE owner_id = $1 AND kind = 'personal' LIMIT 1`,
		userID).Scan(&id)
	return id, err
}
```

- [ ] **Step 2: Зарегистрировать пакет в main.go**

Найти в `api-go/cmd/api/main.go` строки вида `events.InitDB(db)` и добавить рядом:

```go
	calendars.InitDB(db)
```

Импорт — рядом с остальными: `"neuroboost/api-go/internal/calendars"`.

- [ ] **Step 3: Собрать и прогнать всё**

Run: `cd api-go && go build ./... && go vet ./... && go test ./...`
Expected: PASS, ничего не сломано.

- [ ] **Step 4: Commit**

```bash
git add api-go/internal/calendars/store.go api-go/cmd/api/main.go
git commit -m "feat(calendars): CalendarIDsFor, the single scoping entry point"
```

---

### Task 4: Архитектурный тест — красный до Task 5 и 6

**Files:**
- Create: `api-go/internal/calendars/scoping_test.go`

**Interfaces:**
- Consumes: исходники `internal/events/handlers.go`, `internal/tasks/handlers.go` как текст.
- Produces: ничего для кода — гарантию для людей.

- [ ] **Step 1: Написать тест**

```go
// api-go/internal/calendars/scoping_test.go
package calendars

import (
	"bytes"
	"os"
	"testing"
)

// 🔴 Одно забытое место = чужие данные в чужом браузере, и откатить это нельзя:
// показанное однажды показано. Поэтому запрет проверяется механически, а не
// вниманием ревьюера.
//
// Ищется буквально `user_id = $` — то есть ограничение ВЫБОРКИ по автору.
// INSERT, который пишет user_id как колонку авторства, под шаблон не попадает
// и остаётся разрешённым.
func TestHandlersDoNotScopeQueriesByUserID(t *testing.T) {
	forbidden := []byte("user_id = $")

	for _, path := range []string{
		"../events/handlers.go",
		"../tasks/handlers.go",
	} {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read %s: %v", path, err)
		}
		if bytes.Contains(src, forbidden) {
			t.Errorf(
				"%s scopes a query by %q. Access comes from calendar membership: "+
					"use calendars.CalendarIDsFor and `calendar_id = ANY($n)`.",
				path, forbidden)
		}
	}
}
```

- [ ] **Step 2: Прогнать и убедиться, что он ПАДАЕТ**

Run: `cd api-go && go test ./internal/calendars/ -run TestHandlersDoNotScope -v`
Expected: **FAIL** — оба файла пока ограничивают выборку по `user_id` (16 строк в events, 13 в tasks). Это и есть красная фаза: тест описывает работу Task 5 и Task 6.

- [ ] **Step 3: Commit красного теста**

```bash
git add api-go/internal/calendars/scoping_test.go
git commit -m "test(calendars): forbid scoping event and task queries by user_id

Fails until the handlers are migrated. Committed red on purpose: it is the
definition of done for the next two tasks."
```

---

### Task 5: Перевести events на календари

**Files:**
- Modify: `api-go/internal/events/handlers.go` — функции `listEvents`, `getEvent`, `createEvent`, `updateEvent`, `deleteEvent`, `moveEvent`, `resizeEvent`, `ResizeHandler`, `addException`

**Interfaces:**
- Consumes: `calendars.CalendarIDsFor`, `calendars.PersonalIDFor` из Task 3.
- Produces: поведение снаружи без изменений; `event.calendar_id` заполняется при создании.

- [ ] **Step 1: Перевести чтение — listEvents**

Заменить в SQL `WHERE user_id = $1` на `WHERE calendar_id = ANY($1)`, а в вызове передать список вместо userID:

```go
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to resolve calendars")
		return
	}
	// Пустой список — это законное «ничего не видно», а не ошибка:
	// ANY('{}') не вернёт ни одной строки.
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, title, description, starts_at, ends_at, all_day, rrule,
		       COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		       task_id, COALESCE(is_work_event, true), created_at, updated_at,
		       COALESCE(reminder_offsets, '{}')
		FROM event
		WHERE calendar_id = ANY($1)
		  AND (
		    (starts_at < $3 AND ends_at > $2)
		    OR (rrule IS NOT NULL AND rrule != '' AND starts_at < $3)
		  )
	`, calIDs, from, to)
```

⚠️ Порядок плейсхолдеров: `$1` теперь список календарей. Проверить нумерацию **каждого** переписанного запроса — молчаливая перестановка аргументов даёт пустую выдачу, а не ошибку.

- [ ] **Step 2: Перевести getEvent и остальные читающие**

Тот же приём: `WHERE id = $1 AND user_id = $2` → `WHERE id = $1 AND calendar_id = ANY($2)`.

- [ ] **Step 3: Перевести createEvent**

Создание кладёт событие в личный календарь автора:

```go
	calID, err := calendars.PersonalIDFor(ctx, userID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to resolve personal calendar")
		return
	}
```

и в INSERT добавляется колонка `calendar_id` со значением `calID`. `user_id` в INSERT **остаётся** — это авторство.

- [ ] **Step 4: Перевести пишущие — updateEvent, deleteEvent, moveEvent, resizeEvent, ResizeHandler, addException**

Везде `AND user_id = $N` в `WHERE` заменяется на `AND calendar_id = ANY($N)`.

- [ ] **Step 5: Прогнать архитектурный тест — events больше не должен фигурировать**

Run: `cd api-go && go test ./internal/calendars/ -run TestHandlersDoNotScope -v`
Expected: FAIL, но **только** про `../tasks/handlers.go`. Если ещё жалуется на events — какое-то место пропущено.

- [ ] **Step 6: Прогнать весь бэкенд**

Run: `cd api-go && go build ./... && go vet ./... && go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add api-go/internal/events/handlers.go
git commit -m "refactor(events): scope by calendar membership instead of user_id"
```

---

### Task 6: Перевести tasks на календари

**Files:**
- Modify: `api-go/internal/tasks/handlers.go` — функции `listTasks`, `getTask`, `createTask`, `updateTask`, `deleteTask`, `scheduleTask`, `logTaskTime`

**Interfaces:**
- Consumes: `calendars.CalendarIDsFor`, `calendars.PersonalIDFor`.
- Produces: `task.calendar_id` заполняется при создании; архитектурный тест зеленеет.

- [ ] **Step 1: Перевести все семь функций тем же приёмом, что в Task 5**

Чтение и мутации — `calendar_id = ANY($N)`. Создание — `PersonalIDFor` + колонка `calendar_id` в INSERT, `user_id` остаётся авторством.

- [ ] **Step 2: Прогнать архитектурный тест — теперь он должен пройти**

Run: `cd api-go && go test ./internal/calendars/ -v`
Expected: **PASS**, включая `TestHandlersDoNotScopeQueriesByUserID`. Это зелёная фаза Task 4.

- [ ] **Step 3: Прогнать оба Go-модуля**

Run: `cd api-go && go build ./... && go vet ./... && go test ./...`
Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: PASS в обоих. `bot/` не трогали, но он ходит в те же ручки.

- [ ] **Step 4: Commit**

```bash
git add api-go/internal/tasks/handlers.go
git commit -m "refactor(tasks): scope by calendar membership instead of user_id"
```

---

### Task 7: Отдать calendar_id наружу

**Files:**
- Modify: `api-go/internal/events/types.go` — структура ответа события
- Modify: `web/src/types/index.ts` — тип `NbEvent`
- Modify: `web/src/api/tasks.ts` — интерфейс `Task`

**Interfaces:**
- Consumes: колонку `calendar_id`, заполненную в Task 1.
- Produces: поле `calendar_id` в JSON события и задачи; `calendarId` в `NbEvent`.

- [ ] **Step 1: Добавить поле в Go-структуру события и в SELECT**

В `types.go` — поле `CalendarID string \`json:"calendar_id"\``; в `SELECT` списка и одиночного события добавить `calendar_id::text` и просканировать в него.

- [ ] **Step 2: Добавить поле во фронтовые типы**

```ts
// web/src/types/index.ts, в interface NbEvent
  /** Календарь, которому принадлежит событие. Доступ определяется членством в нём. */
  calendarId?: string;
```

```ts
// web/src/api/tasks.ts, в interface Task
  calendar_id?: string
```

⚠️ Конвертация snake_case → camelCase в этом проекте **поэндпоинтная и её легко забыть** — ровно так родился дефект T1. Добавить `calendarId: api.calendar_id` в `toNbEvent`.

- [ ] **Step 3: Проверить фронт целиком**

Run: `cd web && corepack pnpm typecheck && corepack pnpm test --run && corepack pnpm build`
Expected: PASS, тестов не меньше 348.

- [ ] **Step 4: Commit**

```bash
git add api-go/internal/events/types.go api-go/internal/events/handlers.go web/src/types/index.ts web/src/api/tasks.ts
git commit -m "feat(api): expose calendar_id on events and tasks"
```

---

### Task 8: Живая проверка на staging

**Files:** ничего не создаётся — это шлюз.

**Interfaces:**
- Consumes: всё предыдущее.
- Produces: доказательство, что срез 1 снаружи ничего не изменил.

- [ ] **Step 1: Запушить и дождаться CI**

```bash
git push origin develop
gh run list --branch develop --limit 1
```

Дождаться `completed success` **на своём коммите** — не на предыдущем.

- [ ] **Step 2: Убедиться, что миграция применилась и ничего не осиротело**

Команда из Task 1, Step 3. Ожидается три нуля. Дополнительно:

```bash
ssh root@62.76.228.106 "docker exec neuroboost-dev-db psql -U neuroboost -d neuroboost_dev -t -A \
  -c \"select version, dirty from schema_migrations\""
```

Ожидается `12|f`. `dirty = t` означает застрявшую миграцию — чинить немедленно.

- [ ] **Step 3: Прогнать e2e против задеплоенного staging**

```bash
cd web
E2E_TG_BOT_TOKEN="$(ssh root@62.76.228.106 'grep -m1 ^TELEGRAM_BOT_TOKEN= /root/neuroboost-dev/.env | cut -d= -f2-')" \
  E2E_TG_ID=495598685 corepack pnpm exec playwright test
```

Expected: **24 passed, 3 skipped** — ровно базовая линия. Срез 1 не меняет поведение, поэтому любое расхождение здесь есть регрессия, а не новая фича.

- [ ] **Step 4: Проверить руками то, чего e2e не видит**

Открыть `https://dev.neuroboost.website/calendar` и убедиться, что события на месте. 🔴 Самый вероятный дефект этого среза — **пустой календарь** из-за перепутанной нумерации плейсхолдеров: SQL при этом валиден, ошибки нет, просто ноль строк.

- [ ] **Step 5: Отчитаться Денису и НЕ мержить**

Мерж в `main` — релиз прода, только по его явному «да».

---

## Self-Review

**Покрытие спеки срезом 1.** §3 (разделение владения и доступа) — Tasks 2–6. §4.1 (таблицы) — Task 1. §4.3 (колонки, индексы) — Task 1. §7 (миграция и бэкфилл) — Task 1. §9, строка про архитектурный тест — Task 4. §10, шаг 1 — весь этот план.

**Сознательно не в этом плане** (получают свои планы): §4.2 и §6 — персональные напоминания и переписывание сканера P2 (срез 5); §5 роли и удаление календаря (срез 2); §8 ручки календарей и §8.1 приглашения (срезы 2–3); §5.1 `409 CALENDAR_NOT_EMPTY` (срез 2, вместе с `DELETE /api/calendars/:id`).

**Плейсхолдеры:** нет. Все шаги содержат либо код, либо точную команду с ожидаемым результатом.

**Согласованность имён:** `AccessibleIDs`, `Membership`, `StatusActive`, `StatusInvited`, `CalendarIDsFor`, `PersonalIDFor`, `InitDB` — объявлены в Task 2 и 3, используются в Task 5 и 6 в этом же написании. Колонка везде `calendar_id`, поле фронта `calendarId`.

**Известный риск, не закрываемый планом:** `bot/` ходит в те же ручки и в срезе 1 не правится. Его сборка и тесты прогоняются (Task 6, Step 3), но его поведение проверяется только вручную — тапом по боту, который может сделать только Денис.
