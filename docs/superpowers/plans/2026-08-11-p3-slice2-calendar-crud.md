<!-- паспорт: тип=план | статус=действует | строк=1170 | ~токенов=10242 | обновлён=по git -->

# P3 срез 2 — CRUD календарей и список в UI

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** пользователь может завести общий календарь, переименовать его и удалить — и видит
список своих календарей в интерфейсе.

**Architecture:** новый слой хранения и HTTP-ручки внутри существующего пакета
`api-go/internal/calendars`, который со среза 1 уже владеет вопросом «кто что видит». Наружу —
четыре ручки `/api/calendars` в том же стиле, что остальные модули (`util.RespondJSON` /
`util.RespondError`, `middleware.UserIDFromContext`). На фронте — отдельный API-модуль и
самодостаточная секция в настройках.

**Tech Stack:** Go 1.22 (chi, pgx/v5), PostgreSQL 16, React 18 + TypeScript + Vite, Playwright.

---

## Границы среза — прочитать до первой задачи

Срез равен **пункту §10.2 спеки** дословно: *«`GET/POST/PATCH/DELETE /api/calendars` + список
календарей в UI, пока без приглашений»*.

🔴 **Что в срез НЕ входит и почему** — это не забывчивость, а осознанная граница:

| Не входит | Почему |
|---|---|
| Приглашения (`/invites`) | Спека §10.3 — следующий срез. Кроме того, у него есть **предусловие**: слияние раздвоенных личностей в проде (спека §10.0), которое решает Денис и только он |
| Проверка ролей на путях записи событий/задач | До приглашений у каждого календаря **ровно один участник — owner**. Роль, проверяемая сегодня, не имеет ни одного сценария, кроме вручную вставленной в базу строки. Тот же довод, по которому это отложил срез 1 (`2026-08-11-p3-slice2-inherited-debt.md` §1) |
| Создание события/задачи в общем календаре, признак `is_shared` | Спека §10.4. **Следствие:** общие календари в этом срезе физически пусты — `POST /api/events` по-прежнему безусловно кладёт в личный через `PersonalIDFor` |
| «Перенести содержимое одним действием» (спека §5.1) | Ручки для этого в §8 спеки нет, а переносить нечего: см. строку выше. Появляется вместе со срезом 4 |
| Персональные напоминания | Спека §10.5 |

⚠️ **Честная оценка результата:** после среза в списке у Дениса будет одна строка
(«Мой календарь») и кнопка «создать общий», чей календарь пока ничего не может содержать. Это
корректный, безопасный, выпускаемый срез — и он тонкий. Так и задумано порядком §10.

## Global Constraints

- **Формат на проводе — `snake_case`.** Весь API проекта такой (`json:"created_at"`,
  `json:"calendar_id"`). 🔴 TypeScript-тип нового модуля повторяет эти имена **буква в букву**,
  без слоя конвертации. Дефекты T1 (27.07) и T7 (11.08) — оба одной формы: тип обещал
  camelCase, Go отдавал snake_case, поле молча приходило `undefined`. Отсутствие слоя
  конвертации означает отсутствие места, где он разъедется.
- **Никакого `any` в TypeScript.**
- **Новых зависимостей не добавлять.** В `web/` нет `@testing-library` — значит компонент
  юнит-тестом не покрывается. Логика, которую надо проверить, выносится чистой функцией в
  `web/src/lib/calendars/` и тестируется там; UI покрывает e2e.
- **Комментарии в коде, сообщения коммитов и идентификаторы — по-английски.** Пользовательские
  строки — по-русски через существующий i18n, как в соседних секциях настроек.
- **Параметризованные запросы pgx.** Никакой склейки SQL строками.
- **Миграции не трогать.** Срез 2 обходится схемой миграции `000012`; новой миграции нет.
- 🔴 **Тесты с базой пропускаются локально.** `setupTestDB` делает `t.Skip`, когда нет
  `DATABASE_URL` — двенадцать зелёных локальных прогонов 11.08 не заметили nil pointer, который
  CI уронил с первого раза. **До push:**
  ```bash
  docker run --rm -d --name nb-test-pg -e POSTGRES_PASSWORD=x -p 5433:5432 postgres:16
  migrate -path api-go/migrations -database "postgres://postgres:x@localhost:5433/postgres?sslmode=disable" up
  DATABASE_URL="postgres://postgres:x@localhost:5433/postgres?sslmode=disable" go test ./... # из api-go
  ```
- **Охранные тесты среза 1 не мешают.** `scoping_test.go` ловит запросы к таблицам
  `event`/`task`, ограниченные по `user_id`. Запросы этого среза идут к `calendar` и
  `calendar_member`, а единственное обращение к `event`/`task` — счётчик по `calendar_id`.
  Конфликта нет; убедиться прогоном, не рассуждением.

---

## Контракт API — зафиксирован здесь, чтобы Go и TypeScript не разошлись

### Объект календаря (одна форма во всех четырёх ответах)

```json
{
  "id": "6f2c…",
  "name": "Мой календарь",
  "color": null,
  "kind": "personal",
  "role": "owner",
  "status": "active",
  "created_at": "2026-08-11T20:00:00Z"
}
```

`kind` ∈ `personal` | `shared` · `role` ∈ `owner` | `editor` | `viewer` · `status` ∈
`invited` | `active`. `color` — `null` или строка. `role` и `status` приходят из **моей** строки
`calendar_member`, а не из календаря.

### Ручки

| Метод | Путь | Тело | Успех | Отказы |
|---|---|---|---|---|
| `GET` | `/api/calendars` | — | `200`, массив | `401 NOT_AUTHENTICATED` |
| `POST` | `/api/calendars` | `{"name": "…", "color": "#7c3aed"}` | `201`, объект | `400 INVALID_NAME`, `400 INVALID_COLOR`, `400 INVALID_REQUEST`, `401` |
| `PATCH` | `/api/calendars/{id}` | `{"name": "…"}` и/или `{"color": "…"}` | `200`, объект | `400 INVALID_NAME`, `400 INVALID_COLOR`, `400 INVALID_REQUEST`, `401`, `403 NOT_CALENDAR_OWNER`, `404 CALENDAR_NOT_FOUND` |
| `DELETE` | `/api/calendars/{id}` | — | `204` | `401`, `403 NOT_CALENDAR_OWNER`, `404 CALENDAR_NOT_FOUND`, `409 CALENDAR_NOT_EMPTY`, `409 CALENDAR_IS_PERSONAL` |

🔴 **`error.code` — открытое множество, а не закрытое.** Таблица перечисляет коды, которые
ручки отдают сегодня; клиент обязан внятно отработать код, которого в ней нет. Первая
редакция таблицы пропустила `INVALID_COLOR` и `INVALID_REQUEST` — оба отдаются кодом с первого
дня. Исполнитель следующего среза читает этот файл как истину, поэтому `switch` без
ветки по умолчанию здесь превращается в дефект на фронте.

⚠️ **`color` валидируется в handler'е, не в store:** `^#[0-9a-fA-F]{6}$`. Отсутствующий цвет
(поля нет в JSON) — валиден и значит «нет цвета» при создании, «не менять» при правке. Пустая
строка **не** принимается: обнулить цвет в v1 намеренно нельзя.

🔴 **Не участник получает `404 CALENDAR_NOT_FOUND`, а не `403`.** `403` подтверждает, что
календарь с таким id существует — это утечка существования чужого объекта. `403` отдаётся
только тому, кто состоит в календаре, но не `owner`.

### Тело `409 CALENDAR_NOT_EMPTY`

Спека §5.1 требует «счётчик того, что внутри». Форма — ровно такая, потому что общий
`util.RespondError` полей под счётчики не имеет и ответ собирается вручную:

```json
{
  "error": {
    "code": "CALENDAR_NOT_EMPTY",
    "message": "Календарь не пуст",
    "events": 12,
    "tasks": 3
  }
}
```

⚠️ Проверить формой ответа `util.RespondError` **перед** реализацией (`internal/util/response.go`):
конверт `{"error": {...}}` обязан совпасть с тем, что отдают остальные ручки, иначе фронтовый
разбор ошибок этот случай не увидит. Если конверт другой — повторить его, а не изобрести.

### Удаление личного календаря

🔴 Спека §5.1 разбирает только общий. Личный удалять **нельзя вовсе**: `event.calendar_id
REFERENCES calendar(id)` объявлен **без `ON DELETE`**, поэтому удаление непустого личного
даст нарушение внешнего ключа и `500`, а удаление пустого молча снимет у человека личный
календарь, пока `PersonalIDFor` не создаст его заново. Отказ — `409 CALENDAR_IS_PERSONAL`,
явный и типизированный.

---

## Файлы

| Файл | Ответственность |
|---|---|
| `api-go/internal/calendars/types.go` (создать) | `Calendar`, тела запросов, типизированные ошибки |
| `api-go/internal/calendars/crud.go` (создать) | `ListFor`, `Create`, `Update`, `Delete` — только SQL и правила |
| `api-go/internal/calendars/crud_test.go` (создать) | Тесты с базой по образцу `export_test.go` |
| `api-go/internal/calendars/handlers.go` (создать) | Четыре HTTP-обработчика, разбор и коды |
| `api-go/cmd/api/main.go` (правка, блок роутов ~строка 165) | Регистрация четырёх маршрутов |
| `web/src/api/calendars.ts` (создать) | Клиент, типы `snake_case` |
| `web/src/lib/calendars/order.ts` (создать) | Чистая функция сортировки списка |
| `web/src/lib/calendars/order.test.ts` (создать) | Её тест |
| `web/src/components/Calendars/CalendarsSection.tsx` (создать) | Секция настроек целиком |
| `web/src/pages/Settings/Settings.tsx` (правка) | Одна строка — вставить секцию |
| `web/e2e/calendars-crud.spec.ts` (создать) | Путь целиком против staging |

**Почему отдельный компонент, а не код внутри `Settings.tsx`:** файл настроек уже 830+ строк и
тринадцать секций. Ещё одна секция внутри него делает файл хуже; самодостаточный компонент,
вставленный одной строкой, — нет.

---

### Task 1: Слой хранения календарей

**Files:**
- Create: `api-go/internal/calendars/types.go`
- Create: `api-go/internal/calendars/crud.go`
- Test: `api-go/internal/calendars/crud_test.go`

**Interfaces:**
- Consumes: `db` и `InitDB` из `store.go`; константы `RoleOwner`, `StatusActive` из `access.go`;
  `PersonalIDFor` из `store.go`.
- Produces:
  ```go
  func ListFor(ctx context.Context, userID string) ([]Calendar, error)
  func Create(ctx context.Context, userID, name string, color *string) (Calendar, error)
  func Update(ctx context.Context, userID, calendarID string, p UpdateFields) (Calendar, error)
  func Delete(ctx context.Context, userID, calendarID string) error
  ```
  плюс типы `Calendar`, `UpdateFields`, ошибки `ErrCalendarNotFound`, `ErrNotCalendarOwner`,
  `ErrCalendarIsPersonal`, `NotEmptyError`.

- [ ] **Step 1: Написать `types.go`**

```go
package calendars

import (
	"errors"
	"fmt"
	"time"
)

// Calendar kinds.
const (
	KindPersonal = "personal"
	KindShared   = "shared"
)

// Calendar is one calendar as the API returns it. Role and status come from
// the requesting user's own calendar_member row, not from the calendar.
//
// Field names on the wire are snake_case, matching every other module in this
// API. The TypeScript type mirrors them verbatim: no conversion layer means no
// place for a conversion layer to disagree.
type Calendar struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color"`
	Kind      string    `json:"kind"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateFields carries a PATCH. A nil field means "leave unchanged" — clearing
// a colour back to null is not expressible in v1 and no caller needs it.
type UpdateFields struct {
	Name  *string
	Color *string
}

var (
	// ErrCalendarNotFound covers both "no such calendar" and "not a member of
	// it" on purpose: distinguishing them tells a stranger that the calendar
	// exists.
	ErrCalendarNotFound = errors.New("calendar not found")

	// ErrNotCalendarOwner is only ever returned to an actual member.
	ErrNotCalendarOwner = errors.New("not the calendar owner")

	// ErrCalendarIsPersonal guards the delete path. event.calendar_id
	// references calendar(id) with no ON DELETE clause, so deleting a personal
	// calendar either violates the foreign key (non-empty) or silently strips
	// the user of their own calendar (empty).
	ErrCalendarIsPersonal = errors.New("personal calendar cannot be deleted")
)

// NotEmptyError reports what is still inside a calendar the caller tried to
// delete. The counts are the payload of the 409 the spec requires (§5.1).
type NotEmptyError struct {
	Events int
	Tasks  int
}

func (e *NotEmptyError) Error() string {
	return fmt.Sprintf("calendar not empty: %d events, %d tasks", e.Events, e.Tasks)
}
```

- [ ] **Step 2: Написать `crud.go`**

```go
package calendars

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// maxNameLen bounds the stored name. The column is TEXT; this is a product
// limit, not a storage one.
const maxNameLen = 100

// ListFor returns every calendar the user belongs to, personal one first.
//
// It calls PersonalIDFor before reading so a user who registered after
// migration 000012 ran sees their personal calendar on the very first request
// instead of an empty list — that self-healing lives in one place and this is
// the first read path that would expose its absence.
func ListFor(ctx context.Context, userID string) ([]Calendar, error) {
	if _, err := PersonalIDFor(ctx, userID); err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT c.id::text, c.name, c.color, c.kind, m.role, m.status, c.created_at
		 FROM calendar c
		 JOIN calendar_member m ON m.calendar_id = c.id
		 WHERE m.user_id = $1
		 ORDER BY (c.kind = 'personal') DESC, c.created_at`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Calendar{}
	for rows.Next() {
		var c Calendar
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.Kind, &c.Role, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// NormalizeName trims a supplied name and reports whether it is usable.
// Exported so the handler validates with the same rule the store enforces.
func NormalizeName(name string) (string, bool) {
	n := strings.TrimSpace(name)
	if n == "" || len([]rune(n)) > maxNameLen {
		return "", false
	}
	return n, true
}

// Create makes a shared calendar and makes its creator the owning member.
//
// Both inserts run in one transaction: a calendar with no membership row is
// invisible to its own creator (CalendarIDsFor reads calendar_member only),
// and nothing would ever repair it — PersonalIDFor only heals the personal one.
func Create(ctx context.Context, userID, name string, color *string) (Calendar, error) {
	n, ok := NormalizeName(name)
	if !ok {
		return Calendar{}, errors.New("invalid name")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return Calendar{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var c Calendar
	if err := tx.QueryRow(ctx,
		`INSERT INTO calendar (owner_id, name, color, kind)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, name, color, kind, created_at`,
		userID, n, color, KindShared,
	).Scan(&c.ID, &c.Name, &c.Color, &c.Kind, &c.CreatedAt); err != nil {
		return Calendar{}, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status)
		 VALUES ($1, $2, $3, $4)`,
		c.ID, userID, RoleOwner, StatusActive); err != nil {
		return Calendar{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Calendar{}, err
	}

	c.Role = RoleOwner
	c.Status = StatusActive
	return c, nil
}

// membership reads the caller's own row for a calendar. pgx.ErrNoRows here
// means "not a member", which callers translate to ErrCalendarNotFound.
func membership(ctx context.Context, userID, calendarID string) (kind, role string, err error) {
	err = db.Pool.QueryRow(ctx,
		`SELECT c.kind, m.role
		 FROM calendar c
		 JOIN calendar_member m ON m.calendar_id = c.id
		 WHERE c.id = $1 AND m.user_id = $2 AND m.status = $3`,
		calendarID, userID, StatusActive).Scan(&kind, &role)
	return kind, role, err
}

// requireOwner resolves the caller's standing in one place so update and
// delete cannot drift apart.
func requireOwner(ctx context.Context, userID, calendarID string) (kind string, err error) {
	kind, role, err := membership(ctx, userID, calendarID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrCalendarNotFound
	}
	if err != nil {
		return "", err
	}
	if role != RoleOwner {
		return "", ErrNotCalendarOwner
	}
	return kind, nil
}

// Update renames or recolours a calendar. Owner only, including the personal
// one — renaming "Мой календарь" is harmless and expected.
func Update(ctx context.Context, userID, calendarID string, p UpdateFields) (Calendar, error) {
	if _, err := requireOwner(ctx, userID, calendarID); err != nil {
		return Calendar{}, err
	}

	if p.Name != nil {
		n, ok := NormalizeName(*p.Name)
		if !ok {
			return Calendar{}, errors.New("invalid name")
		}
		p.Name = &n
	}

	var c Calendar
	if err := db.Pool.QueryRow(ctx,
		`UPDATE calendar
		 SET name  = COALESCE($2, name),
		     color = COALESCE($3, color)
		 WHERE id = $1
		 RETURNING id::text, name, color, kind, created_at`,
		calendarID, p.Name, p.Color,
	).Scan(&c.ID, &c.Name, &c.Color, &c.Kind, &c.CreatedAt); err != nil {
		return Calendar{}, err
	}

	c.Role = RoleOwner
	c.Status = StatusActive
	return c, nil
}

// Delete removes an empty shared calendar.
//
// Two refusals, both load-bearing:
//   - personal calendars are never deletable (see ErrCalendarIsPersonal);
//   - a non-empty calendar reports its contents instead of cascading. Deleting
//     it would take both members' events, including ones the deleter did not
//     create (spec §5.1).
func Delete(ctx context.Context, userID, calendarID string) error {
	kind, err := requireOwner(ctx, userID, calendarID)
	if err != nil {
		return err
	}
	if kind == KindPersonal {
		return ErrCalendarIsPersonal
	}

	var ne NotEmptyError
	if err := db.Pool.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM event WHERE calendar_id = $1),
		        (SELECT count(*) FROM task  WHERE calendar_id = $1)`,
		calendarID).Scan(&ne.Events, &ne.Tasks); err != nil {
		return err
	}
	if ne.Events > 0 || ne.Tasks > 0 {
		return &ne
	}

	// calendar_member cascades from calendar; nothing else references an empty
	// calendar at this point.
	_, err = db.Pool.Exec(ctx, `DELETE FROM calendar WHERE id = $1`, calendarID)
	return err
}
```

- [ ] **Step 3: Написать тесты с базой — сначала красными**

Образец подключения — `api-go/internal/export/export_test.go`, строки 19–53. Здесь `InitDB`
пакета `calendars` и есть та самая инициализация, поэтому отдельный вызов не нужен.

```go
package calendars

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/database"
)

// setupTestDB connects to DATABASE_URL and seeds a user, returning the user ID
// and a cleanup func. Skips when DATABASE_URL is unset, matching the export and
// tasks packages: these run in CI, not on a bare local run.
func setupTestDB(t *testing.T) (userID string, cleanup func()) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	InitDB(d)
	ctx := context.Background()

	email := fmt.Sprintf("calendars-test-%d@example.com", time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	cleanup = func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE user_id = $1`, userID)
		_, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID)
		d.Close()
	}
	return userID, cleanup
}

// TestListForPutsPersonalFirst also covers the self-healing call: the seeded
// user has no calendar until ListFor creates one.
func TestListForPutsPersonalFirst(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if _, err := Create(ctx, userID, "Общий", nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err := ListFor(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 calendars, got %d", len(list))
	}
	if list[0].Kind != KindPersonal {
		t.Errorf("personal calendar must sort first, got %q", list[0].Kind)
	}
	if list[0].Role != RoleOwner || list[0].Status != StatusActive {
		t.Errorf("role/status must come from my own membership row, got %q/%q",
			list[0].Role, list[0].Status)
	}
}

// TestCreateMakesCreatorAMember is the invariant that makes a calendar usable:
// without the membership row CalendarIDsFor never returns it and the creator
// cannot see their own calendar.
func TestCreateMakesCreatorAMember(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, userID, "  Отпуск  ", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.Name != "Отпуск" {
		t.Errorf("name must be trimmed, got %q", c.Name)
	}
	if c.Kind != KindShared {
		t.Errorf("created calendars are shared, got %q", c.Kind)
	}

	ids, err := CalendarIDsFor(ctx, userID)
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	found := false
	for _, id := range ids {
		if id == c.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("created calendar is not reachable through CalendarIDsFor")
	}
}

// TestDeleteRefusesNonEmpty is the spec §5.1 guard: deleting a calendar with
// content would take events belonging to every member, not just the deleter.
func TestDeleteRefusesNonEmpty(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, userID, "Проект", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at)
		 VALUES ($1, $2, 'x', NOW(), NOW() + interval '1 hour')`,
		userID, c.ID); err != nil {
		t.Fatalf("seed event: %v", err)
	}

	err = Delete(ctx, userID, c.ID)
	var ne *NotEmptyError
	if !errors.As(err, &ne) {
		t.Fatalf("want NotEmptyError, got %v", err)
	}
	if ne.Events != 1 || ne.Tasks != 0 {
		t.Errorf("counts must report contents, got %d events / %d tasks", ne.Events, ne.Tasks)
	}
}

// TestDeleteRefusesPersonal: event.calendar_id references calendar(id) with no
// ON DELETE, so deleting the personal calendar is either an FK violation or a
// silent loss of the user's own container.
func TestDeleteRefusesPersonal(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	personalID, err := PersonalIDFor(ctx, userID)
	if err != nil {
		t.Fatalf("personal: %v", err)
	}
	if err := Delete(ctx, userID, personalID); !errors.Is(err, ErrCalendarIsPersonal) {
		t.Fatalf("want ErrCalendarIsPersonal, got %v", err)
	}
}

// TestDeleteEmptySharedSucceeds is the positive control: without it the three
// refusals above would pass on an implementation that refuses everything.
func TestDeleteEmptySharedSucceeds(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, userID, "Временный", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := Delete(ctx, userID, c.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, err := ListFor(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, x := range list {
		if x.ID == c.ID {
			t.Fatal("calendar still listed after delete")
		}
	}
}

// TestNonMemberGetsNotFound: a stranger must not learn that the calendar
// exists — 403 would confirm it.
func TestNonMemberGetsNotFound(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Чужой", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var strangerID string
	email := fmt.Sprintf("stranger-%d@example.com", time.Now().UnixNano())
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&strangerID); err != nil {
		t.Fatalf("seed stranger: %v", err)
	}
	defer func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, strangerID) }()

	if _, err := Update(ctx, strangerID, c.ID, UpdateFields{}); !errors.Is(err, ErrCalendarNotFound) {
		t.Errorf("update by stranger: want ErrCalendarNotFound, got %v", err)
	}
	if err := Delete(ctx, strangerID, c.ID); !errors.Is(err, ErrCalendarNotFound) {
		t.Errorf("delete by stranger: want ErrCalendarNotFound, got %v", err)
	}
}
```

- [ ] **Step 4: Прогнать тесты против настоящего Postgres**

```bash
docker run --rm -d --name nb-test-pg -e POSTGRES_PASSWORD=x -p 5433:5432 postgres:16
# подождать готовности, затем накатить схему:
migrate -path api-go/migrations \
  -database "postgres://postgres:x@localhost:5433/postgres?sslmode=disable" up
cd api-go && DATABASE_URL="postgres://postgres:x@localhost:5433/postgres?sslmode=disable" \
  go test ./internal/calendars/ -v
```

Ожидание: **все шесть тестов проходят**. 🔴 Прогон без `DATABASE_URL` печатает `SKIP` и ничего
не доказывает — если в выводе `SKIP`, переменная не доехала, а не «тесты зелёные».

- [ ] **Step 5: Полная проверка бэкенда и коммит**

```bash
cd api-go && go build ./... && go vet ./... && go test ./...
cd bot    && go build ./... && go vet ./... && go test ./...
git add api-go/internal/calendars/types.go api-go/internal/calendars/crud.go \
        api-go/internal/calendars/crud_test.go
git commit -m "feat(calendars): store layer for calendar CRUD with delete guards"
```

---

### Task 2: HTTP-ручки и маршруты

**Files:**
- Create: `api-go/internal/calendars/handlers.go`
- Modify: `api-go/cmd/api/main.go` (блок роутов, рядом с `// Planning`)

**Interfaces:**
- Consumes: `ListFor`, `Create`, `Update`, `Delete`, `UpdateFields` и типизированные ошибки из
  Task 1; `middleware.UserIDFromContext`; `util.RespondJSON` / `util.RespondError`.
- Produces: `ListHandler`, `CreateHandler`, `UpdateHandler`, `DeleteHandler` —
  все `func(http.ResponseWriter, *http.Request)`.

- [ ] **Step 1: Прочитать `internal/util/response.go`**

Убедиться в точной форме конверта ошибки, которую строит `util.RespondError`, — тело
`409 CALENDAR_NOT_EMPTY` собирается вручную и **обязано** его повторить. Если конверт
отличается от `{"error":{"code","message"}}`, повторить фактический, а не описанный здесь.

- [ ] **Step 2: Написать `handlers.go`**

```go
package calendars

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

type createRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color"`
}

type updateRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// caller resolves the authenticated user, writing the 401 itself. Returns ""
// when it has already responded.
func caller(w http.ResponseWriter, r *http.Request) string {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return ""
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
	}
	return userID
}

// ListHandler handles GET /api/calendars
func ListHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	list, err := ListFor(r.Context(), userID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "LIST_ERROR", "Failed to list calendars")
		return
	}
	util.RespondJSON(w, http.StatusOK, list)
}

// CreateHandler handles POST /api/calendars
func CreateHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if _, ok := NormalizeName(req.Name); !ok {
		util.RespondError(w, http.StatusBadRequest, "INVALID_NAME", "Name must be 1-100 characters")
		return
	}
	c, err := Create(r.Context(), userID, req.Name, req.Color)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create calendar")
		return
	}
	util.RespondJSON(w, http.StatusCreated, c)
}

// UpdateHandler handles PATCH /api/calendars/{id}
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.Name != nil {
		if _, ok := NormalizeName(*req.Name); !ok {
			util.RespondError(w, http.StatusBadRequest, "INVALID_NAME", "Name must be 1-100 characters")
			return
		}
	}
	c, err := Update(r.Context(), userID, chi.URLParam(r, "id"), UpdateFields{Name: req.Name, Color: req.Color})
	if err != nil {
		respondCalendarError(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, c)
}

// DeleteHandler handles DELETE /api/calendars/{id}
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	if err := Delete(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		respondCalendarError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// respondCalendarError maps the store's typed errors onto status codes in one
// place, so update and delete cannot disagree about what a stranger is told.
func respondCalendarError(w http.ResponseWriter, err error) {
	var ne *NotEmptyError
	switch {
	case errors.Is(err, ErrCalendarNotFound):
		// Deliberately 404, not 403: a 403 would confirm to a stranger that
		// this calendar exists.
		util.RespondError(w, http.StatusNotFound, "CALENDAR_NOT_FOUND", "Календарь не найден")
	case errors.Is(err, ErrNotCalendarOwner):
		util.RespondError(w, http.StatusForbidden, "NOT_CALENDAR_OWNER", "Только владелец может изменить календарь")
	case errors.Is(err, ErrCalendarIsPersonal):
		util.RespondError(w, http.StatusConflict, "CALENDAR_IS_PERSONAL", "Личный календарь удалить нельзя")
	case errors.As(err, &ne):
		respondNotEmpty(w, ne)
	default:
		util.RespondError(w, http.StatusInternalServerError, "CALENDAR_ERROR", "Failed to update calendar")
	}
}

// respondNotEmpty writes the 409 the spec requires with the counts inline.
// util.RespondError has no field for them, so the envelope is built by hand —
// keep its shape identical to util's, or the frontend error parser will miss
// this case entirely.
func respondNotEmpty(w http.ResponseWriter, ne *NotEmptyError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    "CALENDAR_NOT_EMPTY",
			"message": "Календарь не пуст",
			"events":  ne.Events,
			"tasks":   ne.Tasks,
		},
	})
}
```

- [ ] **Step 3: Зарегистрировать маршруты в `main.go`**

В защищённом блоке роутов, рядом с существующими секциями (перед `// Export / Import`):

```go
		// Calendars
		r.Get("/api/calendars", calendars.ListHandler)
		r.Post("/api/calendars", calendars.CreateHandler)
		r.Patch("/api/calendars/{id}", calendars.UpdateHandler)
		r.Delete("/api/calendars/{id}", calendars.DeleteHandler)
```

Пакет `calendars` в `main.go` уже импортирован (`calendars.InitDB(db)` на строке 56) — новый
импорт не нужен, проверить.

- [ ] **Step 4: Проверить, что охранные тесты не покраснели**

```bash
cd api-go && go build ./... && go vet ./... && go test ./internal/calendars/
```

Ожидание: `TestNoUserIDScopedEventOrTaskQueries` и
`TestInsertsIntoEventOrTaskSetCalendarID` — зелёные. Счётчик просканированных SQL-блоков вырос
(порог `minSQLBlocksScanned = 60` — снизу, не сверху, поэтому рост безопасен).

- [ ] **Step 5: Полная проверка и коммит**

```bash
cd api-go && go build ./... && go vet ./... && go test ./...
cd bot    && go build ./... && go vet ./... && go test ./...
git add api-go/internal/calendars/handlers.go api-go/cmd/api/main.go
git commit -m "feat(api): expose calendar CRUD endpoints"
```

---

### Task 3: Фронт — клиент, сортировка, секция настроек

**Files:**
- Create: `web/src/api/calendars.ts`
- Create: `web/src/lib/calendars/order.ts`
- Test: `web/src/lib/calendars/order.test.ts`
- Create: `web/src/components/Calendars/CalendarsSection.tsx`
- Modify: `web/src/pages/Settings/Settings.tsx`

**Interfaces:**
- Consumes: `api` из `web/src/api/client.ts`; `showToast` из `web/src/components/ui/Toast`;
  иконки Lucide.
- Produces: тип `Calendar`, функции `listCalendars` / `createCalendar` / `updateCalendar` /
  `deleteCalendar`; чистая `sortCalendars`; компонент по умолчанию `CalendarsSection`.

- [ ] **Step 1: Написать падающий тест сортировки**

`web/src/lib/calendars/order.test.ts`:

```ts
import { describe, it, expect } from 'vitest'
import { sortCalendars } from './order'
import type { Calendar } from '../../api/calendars'

const make = (over: Partial<Calendar>): Calendar => ({
  id: 'x',
  name: 'x',
  color: null,
  kind: 'shared',
  role: 'owner',
  status: 'active',
  created_at: '2026-08-11T00:00:00Z',
  ...over,
})

describe('sortCalendars', () => {
  it('puts the personal calendar first regardless of creation order', () => {
    const list = [
      make({ id: 'a', kind: 'shared', created_at: '2026-01-01T00:00:00Z' }),
      make({ id: 'p', kind: 'personal', created_at: '2026-08-01T00:00:00Z' }),
    ]
    expect(sortCalendars(list).map((c) => c.id)).toEqual(['p', 'a'])
  })

  it('orders the rest oldest first', () => {
    const list = [
      make({ id: 'new', created_at: '2026-08-10T00:00:00Z' }),
      make({ id: 'old', created_at: '2026-02-10T00:00:00Z' }),
    ]
    expect(sortCalendars(list).map((c) => c.id)).toEqual(['old', 'new'])
  })

  it('does not mutate its input', () => {
    const list = [make({ id: 'a' }), make({ id: 'p', kind: 'personal' })]
    sortCalendars(list)
    expect(list.map((c) => c.id)).toEqual(['a', 'p'])
  })
})
```

- [ ] **Step 2: Прогнать — упасть**

```bash
cd web && corepack pnpm test --run src/lib/calendars
```
Ожидание: FAIL, модуль `./order` не найден.

- [ ] **Step 3: Написать `web/src/api/calendars.ts`**

```ts
import { api } from './client'

/**
 * Calendar as the API returns it.
 *
 * Field names are snake_case because that is what the API sends — every other
 * module in this backend does the same. There is deliberately no camelCase
 * mapping layer here: defects T1 (27.07) and T7 (11.08) were both a type that
 * promised camelCase over a payload that was snake_case, and the field simply
 * arrived undefined. No mapping means nothing to get wrong.
 */
export interface Calendar {
  id: string
  name: string
  color: string | null
  kind: 'personal' | 'shared'
  role: 'owner' | 'editor' | 'viewer'
  status: 'invited' | 'active'
  created_at: string
}

export interface CalendarNotEmpty {
  code: 'CALENDAR_NOT_EMPTY'
  events: number
  tasks: number
}

export function listCalendars(): Promise<Calendar[]> {
  return api.get<Calendar[]>('/calendars')
}

export function createCalendar(name: string, color?: string): Promise<Calendar> {
  return api.post<Calendar>('/calendars', { name, color: color ?? null })
}

export function updateCalendar(
  id: string,
  patch: { name?: string; color?: string },
): Promise<Calendar> {
  return api.patch<Calendar>(`/calendars/${id}`, patch)
}

export function deleteCalendar(id: string): Promise<void> {
  return api.delete(`/calendars/${id}`)
}
```

⚠️ **Шаг проверки, а не предположения:** открыть `web/src/api/client.ts` и убедиться, как
`request` сообщает об ошибке — бросает ли он объект с `code`, и как выглядит успешный `204`
(тело пустое; `response.json()` на пустом теле бросает). Если `delete` на `204` падает —
чинить в клиенте нельзя вслепую: посмотреть, как это делают существующие вызовы
`api.delete` (например, удаление события), и повторить.

- [ ] **Step 4: Написать `web/src/lib/calendars/order.ts`**

```ts
import type { Calendar } from '../../api/calendars'

/**
 * Personal calendar first, then the rest oldest first.
 *
 * The API already sorts this way; sorting again on the client keeps the list
 * stable after a local create, which appends rather than refetches.
 */
export function sortCalendars(list: Calendar[]): Calendar[] {
  return [...list].sort((a, b) => {
    if (a.kind !== b.kind) return a.kind === 'personal' ? -1 : 1
    return a.created_at.localeCompare(b.created_at)
  })
}
```

- [ ] **Step 5: Прогнать — пройти**

```bash
cd web && corepack pnpm test --run src/lib/calendars
```
Ожидание: 3 passed.

- [ ] **Step 6: Написать `web/src/components/Calendars/CalendarsSection.tsx`**

Требования к компоненту (свобода в вёрстке, границы — жёсткие):

- Обёртка повторяет соседние секции настроек: `<section className="bg-zinc-900 border
  border-zinc-800 rounded-lg p-5">`, заголовок с иконкой Lucide (`CalendarDays`).
- Загрузка: `useEffect` → `listCalendars()` → `sortCalendars`. На время — скелет или «Загрузка…».
- Строка календаря: имя, значок `Личный` для `kind === 'personal'`, кнопки «Переименовать» и
  «Удалить» — **только** при `role === 'owner' && kind === 'shared'` для удаления;
  переименование доступно владельцу и личного тоже.
- Создание: поле ввода + кнопка «Создать общий календарь» → `createCalendar` → добавить в
  состояние и пересортировать. Пустое имя — кнопка `disabled`.
- Удаление: `deleteCalendar`; при `409 CALENDAR_NOT_EMPTY` показать `showToast` с числами из
  ответа («В календаре 12 событий и 3 задачи — сначала перенесите их»); при
  `409 CALENDAR_IS_PERSONAL` — «Личный календарь удалить нельзя».
- 🔴 **`data-testid` обязательны**, e2e-спека Task 4 опирается на них:
  `calendars-section`, `calendar-row`, `calendar-name`, `calendar-create-input`,
  `calendar-create-submit`, `calendar-delete`.
- Все видимые строки — по-русски, через существующий `useTranslation`, как в соседних секциях.
  Ключи класть в тот же файл переводов, где живут ключи настроек.
- Ни одного `any`.

- [ ] **Step 7: Вставить секцию в настройки**

В `web/src/pages/Settings/Settings.tsx` — импорт и **одна** строка рендера рядом с другими
секциями (уместно после секции напоминаний):

```tsx
import { CalendarsSection } from '../../components/Calendars/CalendarsSection'
// …
        <CalendarsSection />
```

- [ ] **Step 8: Полная проверка фронта**

```bash
cd web && corepack pnpm typecheck
cd web && corepack pnpm test --run
cd web && corepack pnpm build
```

🔴 Все три. `build` проходит на коде с ошибкой типов — Vite типы стирает; настоящий гейт —
`typecheck`. Это случилось в этом проекте 11.08.

- [ ] **Step 9: Коммит**

```bash
git add web/src/api/calendars.ts web/src/lib/calendars/order.ts \
        web/src/lib/calendars/order.test.ts \
        web/src/components/Calendars/CalendarsSection.tsx \
        web/src/pages/Settings/Settings.tsx
git commit -m "feat(web): calendar list and creation in settings"
```

---

### Task 4: e2e против задеплоенного staging

**Files:**
- Create: `web/e2e/calendars-crud.spec.ts`

**Interfaces:**
- Consumes: существующие фикстуры авторизации — читать `web/e2e/auth-fixture.spec.ts` и
  `web/e2e/fixtures/` и повторить их способ логина, **не изобретая свой**.

- [ ] **Step 1: Запушить бэкенд и фронт, дождаться CI**

```bash
git push origin develop
gh run watch   # или проверить, что job deploy-dev прошёл
```

🔴 Спека гоняется против **задеплоенного** staging. Прогон до окончания деплоя проверяет
предыдущий билд и его зелёный цвет ничего не значит.

- [ ] **Step 2: Написать спеку**

Один сценарий, четыре утверждения — по API, не по пикселям, там где можно:

1. Открыть настройки → секция `calendars-section` видна, в списке **есть** строка с личным
   календарём.
2. Создать общий календарь с уникальным именем (`Тест ${Date.now()}`) → строка появилась.
3. Удалить его → строка исчезла.
4. 🔴 **Отказ `409 CALENDAR_NOT_EMPTY` в e2e НЕ проверяется — и это проверено, а не
   предположено.** `CreateRequest` в `api-go/internal/events/types.go` поля `calendar_id` не
   имеет, а `createEvent` (`handlers.go:667`) безусловно берёт `PersonalIDFor`. Значит положить
   событие в общий календарь снаружи в этом срезе **невозможно**, и отказ покрывается только
   Go-тестом `TestDeleteRefusesNonEmpty` из Task 1. Ограничение назвать в отчёте вслух: молча
   пропущенная проверка хуже отсутствующей. Полный e2e-путь появится со срезом 4.

- [ ] **Step 3: Прогнать против staging**

```bash
E2E_TG_BOT_TOKEN="$(ssh root@62.76.228.106 'grep -m1 ^TELEGRAM_BOT_TOKEN= /root/neuroboost-dev/.env | cut -d= -f2-')" \
  E2E_TG_ID=495598685 corepack pnpm exec playwright test
```

Базовая линия до этого среза — **24 passed / 4 skipped**. Ожидание после: 25 passed, те же
4 skipped. Любое падение из старых 24 — регресс этого среза, а не «флак».

- [ ] **Step 4: Коммит и push**

```bash
git add web/e2e/calendars-crud.spec.ts
git commit -m "test(e2e): calendar create, list and delete against staging"
git push origin develop
```

---

## Проверка после всех задач

```bash
cd api-go && go build ./... && go vet ./... && go test ./...
cd bot    && go build ./... && go vet ./... && go test ./...
cd web && corepack pnpm typecheck && corepack pnpm test --run && corepack pnpm build
# и один раз с настоящей базой:
cd api-go && DATABASE_URL="postgres://postgres:x@localhost:5433/postgres?sslmode=disable" go test ./...
docker rm -f nb-test-pg
```

🔴 **Мерж в `main` запрещён без явного «да» Дениса** — это релиз прода.

## Что останется открытым после среза

Долг из `2026-08-11-p3-slice2-inherited-debt.md` **не уменьшается** этим срезом: роли
по-прежнему не проверяются на путях записи событий и задач, напоминания по-прежнему одни на
всех, экспорт по-прежнему массовый. Всё это становится достижимым только со срезом 3
(приглашения), у которого есть собственное предусловие — слияние личностей в проде, решение
Дениса.
