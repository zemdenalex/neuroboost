# «Задачи дня» D1 — API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** API для «задач дня»: хранение набора на день, цвет дня одной функцией на сервере,
предложение набора, правило «убрать — только до полудня».

**Architecture:** новый пакет `api-go/internal/daytasks` (как `broadcast`: свой `InitDB`, свои
хэндлеры), миграция `000022`. Уровень дня — чистая `Level(done, target)`; всё, что зависит от
дня, считается **в зоне пользователя** (`"user".timezone`). Задачи в тестах создаются и
закрываются **через настоящие двери** `tasks` (CreateHandler, UpdateHandler, MarkOccurrence) —
урок 20.09: тест, который сеет колонку в обход API, не доказывает, что дверь есть.

**Tech Stack:** Go 1.22, chi, pgx/v5, golang-migrate, тестовая Postgres (`DATABASE_URL`).

**Spec:** `docs/superpowers/specs/2026-09-22-day-tasks-design.md`

## Global Constraints

- Уровни `0…5` = ⬛🟫🟥🟧🟨🟩. **N = 3 → «по оставшимся»:** 3→5, 2→4, 1→3, 0→0. **Прочие N →
  «по доле»:** 100 % →5 · ≥ 80 % →4 · ≥ 60 % →3 · ≥ 40 % →2 · > 0 →1 · 0 →0. Больше N → 5.
- **День не взят (нет подтверждения) → уровень 0**, сколько бы ни было сделано (слова Дениса 22.09).
- Набор короче N считается против N (2 из 2 при N = 5 → уровень 2, 🟥).
- «Сделано»: разовая — `completed_at` в этот день **в зоне пользователя**; серия —
  `task_occurrence(state='done')` на этот день. Только задачи.
- Добавить — только открытую задачу (разовая не `DONE`; у серии этот день не отвечен).
- Убрать: будущий день — всегда; сегодня — до 12:00 в зоне пользователя; прошлый день — никогда.
  Иначе `409 TOO_LATE`.
- Цель N — `settings.day_tasks_target`, целое 3–7; нет или вне диапазона → 5.
- Диапазон `GET /api/day-tasks` ≤ 62 дня, иначе `400 RANGE_TOO_LARGE`.
- Ответ — конверт `{"data": …}` через `util.RespondJSON`; ошибки — `util.RespondError(w, код, CODE, msg)`.
- Массивы в ответе — `[]`, никогда `null` (gotcha 6).
- Доступ к задаче — `calendars.CalendarIDsFor` (видимые). Набор — личный: строки по `user_id`.
- Тесты БД: `DATABASE_URL`, **`-count=1`**, пустой вывод `ok` без прогона — не зачёт.
- Сабботаж — компилируется, применяется, падает на своей строке, и прочитан целиком.
- `git add` конкретных файлов; коммит без co-author.

---

## Карта файлов

| Файл | Что |
|---|---|
| `api-go/migrations/000022_day_commitment.up.sql` / `.down.sql` | две таблицы |
| `api-go/internal/daytasks/level.go` | `Level(done, target)` — чистая |
| `api-go/internal/daytasks/level_test.go` | таблица N = 3…7 |
| `api-go/internal/daytasks/rules.go` | `CanRemove(day, now, tz)`, `Target(raw settings)` — чистые |
| `api-go/internal/daytasks/rules_test.go` | полдень в New York, границы цели |
| `api-go/internal/daytasks/store.go` | `InitDB`, `List`, `Add`, `Remove`, `Confirm`, ошибки |
| `api-go/internal/daytasks/proposal.go` | `Propose` |
| `api-go/internal/daytasks/handlers.go` | пять хэндлеров |
| `api-go/internal/daytasks/db_test.go` | фикстуры + HTTP-тесты против БД |
| `api-go/cmd/api/main.go` | `daytasks.InitDB(db)`, пять роутов |

---

### Task 1: Уровень дня — чистая функция

**Files:**
- Create: `api-go/internal/daytasks/level.go`
- Test: `api-go/internal/daytasks/level_test.go`

**Interfaces:**
- Produces: `func Level(done, target int) int` (0…5).

- [ ] **Step 1: Write the failing test**

```go
package daytasks

import "testing"

// Spec §3. N=5 is Denis's table word for word; N=3 counts what is left; every
// other N goes by share. The table is the spec — a change here is a change to
// what the colours mean.
func TestLevel(t *testing.T) {
	cases := []struct{ done, target, want int }{
		// N=5: 🟩🟨🟧🟥🟫⬛
		{5, 5, 5}, {4, 5, 4}, {3, 5, 3}, {2, 5, 2}, {1, 5, 1}, {0, 5, 0},
		// N=3, by what is left: 3 🟩 · 2 🟨 · 1 🟧 · 0 ⬛
		{3, 3, 5}, {2, 3, 4}, {1, 3, 3}, {0, 3, 0},
		// N=4 by share: 100 · 75 · 50 · 25 · 0
		{4, 4, 5}, {3, 4, 3}, {2, 4, 2}, {1, 4, 1}, {0, 4, 0},
		// N=6: 100 · 83 · 66 · 50 · 33 · 16 · 0
		{6, 6, 5}, {5, 6, 4}, {4, 6, 3}, {3, 6, 2}, {2, 6, 1}, {1, 6, 1}, {0, 6, 0},
		// N=7: 100 · 85 · 71 · 57 · 42 · 28 · 14 · 0
		{7, 7, 5}, {6, 7, 4}, {5, 7, 3}, {4, 7, 2}, {3, 7, 2}, {2, 7, 1}, {1, 7, 1}, {0, 7, 0},
		// more done than promised is still a full day
		{6, 5, 5},
	}
	for _, c := range cases {
		if got := Level(c.done, c.target); got != c.want {
			t.Errorf("Level(%d, %d) = %d, want %d", c.done, c.target, got, c.want)
		}
	}
}

// A target outside 3–7 never reaches here (Target clamps it), but a zero would
// divide by zero — the function answers 0 rather than panicking the API.
func TestLevelWithNoTargetIsZero(t *testing.T) {
	if got := Level(3, 0); got != 0 {
		t.Errorf("Level(3, 0) = %d, want 0", got)
	}
}
```

- [ ] **Step 2: Run to see it fail**

Run: `cd api-go && go test ./internal/daytasks/ -run TestLevel -count=1`
Expected: FAIL — `undefined: Level`

- [ ] **Step 3: Implement**

```go
// Package daytasks is «задачи дня» (spec 2026-09-22): N tasks promised for a
// day, and the day's colour by how many of them were closed on it.
package daytasks

// Level is the colour of a day: 0 ⬛ · 1 🟫 · 2 🟥 · 3 🟧 · 4 🟨 · 5 🟩.
//
// 🔴 The server computes it and every client only draws it — the same rule as
// the fill of a day: two computations would one day paint one day twice.
//
// Denis, 22.09: N=3 counts what is left, every other N goes by share. At N=5
// both readings give his table exactly.
func Level(done, target int) int {
	if target <= 0 || done <= 0 {
		return 0
	}
	if done >= target {
		return 5
	}
	if target == 3 {
		return map[int]int{2: 4, 1: 3}[done]
	}
	share := done * 100 / target
	switch {
	case share >= 80:
		return 4
	case share >= 60:
		return 3
	case share >= 40:
		return 2
	default:
		return 1
	}
}
```

- [ ] **Step 4: Run to see it pass**

Run: `cd api-go && go test ./internal/daytasks/ -run TestLevel -count=1 -v`
Expected: PASS, both tests.

- [ ] **Step 5: Sabotage** — поменять `share >= 80` на `share >= 90`: обязан упасть
  `Level(4, 5) = 3, want 4`. Вернуть, снова зелёно.

- [ ] **Step 6: Commit**

```bash
git add api-go/internal/daytasks/level.go api-go/internal/daytasks/level_test.go
git commit -m "feat(api): day-tasks level — one function for the colour of a day"
```

---

### Task 2: Правила — полдень и цель

**Files:**
- Create: `api-go/internal/daytasks/rules.go`
- Test: `api-go/internal/daytasks/rules_test.go`

**Interfaces:**
- Produces:
  - `func CanRemove(day, now time.Time, tz string) bool` — `day` — дата (любая зона, берутся Y-M-D).
  - `func Target(settings []byte) int` — 3…7, иначе 5.
  - `func Today(now time.Time, tz string) time.Time` — полночь сегодняшнего дня в зоне `tz`.

- [ ] **Step 1: Write the failing test**

```go
package daytasks

import (
	"testing"
	"time"
)

// Spec §5, in the USER's zone: 11:59 in New York may remove today's task,
// 12:00 may not — whatever the server's clock says.
func TestCanRemoveIsNoonWhereTheUserIs(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	day := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC) // a date; zone irrelevant
	at := func(h, m int) time.Time { return time.Date(2026, 9, 22, h, m, 0, 0, ny) }

	cases := []struct {
		name string
		now  time.Time
		day  time.Time
		want bool
	}{
		{"today 11:59", at(11, 59), day, true},
		{"today 12:00", at(12, 0), day, false},
		{"tomorrow, late", at(23, 0), day.AddDate(0, 0, 1), true},
		{"yesterday, early", at(8, 0), day.AddDate(0, 0, -1), false},
		// 03:00 UTC on the 23rd is still the 22nd, 23:00, in New York: it is
		// «today after noon» there, not «tomorrow morning».
		{"server already on the 23rd", time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC), day, false},
		{"server already on the 23rd, the 23rd is tomorrow", time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC), day.AddDate(0, 0, 1), true},
	}
	for _, c := range cases {
		if got := CanRemove(c.day, c.now, "America/New_York"); got != c.want {
			t.Errorf("%s: CanRemove = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTargetIsThreeToSevenElseFive(t *testing.T) {
	cases := map[string]int{
		`{}`:                        5,
		`{"day_tasks_target":3}`:    3,
		`{"day_tasks_target":7}`:    7,
		`{"day_tasks_target":2}`:    5,
		`{"day_tasks_target":8}`:    5,
		`{"day_tasks_target":"4"}`:  5, // a string is not a number the bot wrote
		`not json`:                  5,
	}
	for raw, want := range cases {
		if got := Target([]byte(raw)); got != want {
			t.Errorf("Target(%s) = %d, want %d", raw, got, want)
		}
	}
}
```

- [ ] **Step 2: Run to see it fail**

Run: `cd api-go && go test ./internal/daytasks/ -run 'CanRemove|Target' -count=1`
Expected: FAIL — `undefined: CanRemove`

- [ ] **Step 3: Implement**

```go
package daytasks

import (
	"encoding/json"
	"time"
)

// DefaultTarget is Atrioc's five, Denis's default.
const DefaultTarget = 5

// Today is midnight of the user's current day, in the user's zone.
func Today(now time.Time, tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	l := now.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

// sameOrder compares two dates by their Y-M-D alone.
func ymd(t time.Time) int { return t.Year()*10000 + int(t.Month())*100 + t.Day() }

// CanRemove is spec §5: a future day always, today until 12:00 where the user
// is, a past day never. Without the noon line, dropping an undone task at
// 23:00 would turn the day green — the colour would stop meaning anything.
func CanRemove(day, now time.Time, tz string) bool {
	today := Today(now, tz)
	switch d, t := ymd(day), ymd(today); {
	case d > t:
		return true
	case d < t:
		return false
	}
	loc := today.Location()
	return now.In(loc).Hour() < 12
}

// Target reads settings.day_tasks_target: an integer 3–7, else the default.
// Top level, not bot.*: the web reads it too, and the server needs it for the
// level.
func Target(settings []byte) int {
	var s struct {
		Target *int `json:"day_tasks_target"`
	}
	if err := json.Unmarshal(settings, &s); err != nil || s.Target == nil {
		return DefaultTarget
	}
	if *s.Target < 3 || *s.Target > 7 {
		return DefaultTarget
	}
	return *s.Target
}
```

⚠ `json.Unmarshal` строки `"4"` в `*int` даёт ошибку → вернётся 5; это и проверяет кейс.

- [ ] **Step 4: Run to see it pass** — `go test ./internal/daytasks/ -count=1 -v`, все зелёные.

- [ ] **Step 5: Sabotage** — в `CanRemove` вместо `now.In(loc).Hour()` поставить `now.Hour()`:
  обязан упасть кейс `"server already on the 23rd"` (UTC 03:00 < 12). Вернуть.

- [ ] **Step 6: Commit**

```bash
git add api-go/internal/daytasks/rules.go api-go/internal/daytasks/rules_test.go
git commit -m "feat(api): day-tasks rules — noon where the user is, a target of 3 to 7"
```

---

### Task 3: Миграция и хранилище (List / Add / Remove / Confirm)

**Files:**
- Create: `api-go/migrations/000022_day_commitment.up.sql`, `000022_day_commitment.down.sql`
- Create: `api-go/internal/daytasks/store.go`
- Create: `api-go/internal/daytasks/db_test.go`

**Interfaces:**
- Consumes: `Level`, `Target`, `CanRemove`, `Today`.
- Produces:
  - `type Item struct { TaskID, Title string; Done bool }` (json `task_id`, `title`, `done`)
  - `type Day struct { Day string; Target int; Confirmed bool; Items []Item; Done, Level int }`
    (json `day`, `target`, `confirmed`, `items`, `done`, `level`)
  - `func List(ctx, userID string, from, to time.Time) ([]Day, error)`
  - `func Add(ctx, userID string, day time.Time, taskID string) error`
  - `func Remove(ctx, userID string, day time.Time, taskID string, now time.Time) error`
  - `func Confirm(ctx, userID string, day time.Time, taskIDs []string) error`
  - errors: `ErrTooLate`, `ErrNotOpen`, `ErrTaskNotFound`, `ErrRangeTooLarge`, `ErrInvalidRange`
  - `func userZoneAndTarget(ctx, userID string) (tz string, target int, err error)`

- [ ] **Step 1: Migration**

`000022_day_commitment.up.sql`:

```sql
-- «Задачи дня» (spec 2026-09-22 §7). A day is a DATE in the user's zone, never
-- an instant: a task has no clock (learning-a-day-is-a-date-not-an-instant).
CREATE TABLE IF NOT EXISTS day_commitment (
    user_id    UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    day        DATE NOT NULL,
    task_id    UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Soft delete: «removed before noon» stays visible to anyone asking why a
    -- day is the colour it is.
    removed_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, day, task_id)
);
CREATE INDEX IF NOT EXISTS idx_day_commitment_task ON day_commitment(task_id);

-- «The day is taken» — the proposal was confirmed. A day without a row here
-- is ⬛ however much was done (Denis, 22.09).
CREATE TABLE IF NOT EXISTS day_commitment_day (
    user_id      UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    day          DATE NOT NULL,
    confirmed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, day)
);
```

`000022_day_commitment.down.sql`:

```sql
DROP TABLE IF EXISTS day_commitment_day;
DROP TABLE IF EXISTS day_commitment;
```

Применить к локальной тестовой базе (контейнер `nb-test-db`, см. memory `test-database.md`):

```bash
docker exec -i nb-test-db psql -U neuroboost -d neuroboost_test < api-go/migrations/000022_day_commitment.up.sql
```

⚠ `IF NOT EXISTS` здесь безопасен только потому, что таблицы новые: gotcha 18 — baseline
принял чужую таблицу молча. Проверка формы — Step 2 спрашивает колонки у базы.

- [ ] **Step 2: Write the failing DB tests**

`db_test.go`:

```go
package daytasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/tasks"
)

const ny = "America/New_York"

// dayDB is a user in New York with an empty day-tasks history.
func dayDB(t *testing.T) (*database.DB, context.Context, string) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(d.Close)
	InitDB(d)
	tasks.InitDB(d)
	calendars.InitDB(d)
	ctx := context.Background()
	var id string
	email := fmt.Sprintf("daytasks-%d@example.com", time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email, timezone) VALUES ($1, $2) RETURNING id`, email, ny).Scan(&id); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, id) })
	return d, ctx, id
}

func asUser(req *http.Request, userID string) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
}

// newTask goes through the tasks API's own door, not an INSERT.
func newTask(t *testing.T, userID string, body map[string]any) string {
	t.Helper()
	b, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	tasks.CreateHandler(rec, asUser(httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(b)), userID))
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("create task: %d %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Data.ID == "" {
		t.Fatalf("create task: no id in %s", rec.Body.String())
	}
	return got.Data.ID
}

// closeTask is «✅ Готово» through PATCH, which is what sets completed_at.
func closeTask(t *testing.T, userID, taskID string) {
	t.Helper()
	req := asUser(httptest.NewRequest(http.MethodPatch, "/api/tasks/"+taskID,
		bytes.NewReader([]byte(`{"status":"DONE"}`))), userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", taskID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	tasks.UpdateHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("close task: %d %s", rec.Code, rec.Body.String())
	}
}

// 🔴 gotcha 18: ask the database for the shape, do not trust the chain that built it.
func TestDayCommitmentTablesHaveTheirShape(t *testing.T) {
	d, ctx, _ := dayDB(t)
	for table, cols := range map[string][]string{
		"day_commitment":     {"user_id", "day", "task_id", "added_at", "removed_at"},
		"day_commitment_day": {"user_id", "day", "confirmed_at"},
	} {
		for _, c := range cols {
			var n int
			if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
				WHERE table_name = $1 AND column_name = $2`, table, c).Scan(&n); err != nil || n != 1 {
				t.Errorf("%s.%s: found %d (%v) — migration 000022 not applied?", table, c, n, err)
			}
		}
	}
}

// Spec §1, §2, §3 in one day: 5 promised, 3 closed today, one closed tomorrow
// does not count, the day is confirmed → 3 of 5, 🟧.
func TestADayCountsWhatWasClosedOnIt(t *testing.T) {
	_, ctx, user := dayDB(t)
	now := time.Now()
	today := Today(now, ny)

	var ids []string
	for i := 0; i < 5; i++ {
		ids = append(ids, newTask(t, user, map[string]any{"title": fmt.Sprintf("дело %d", i), "priority": 2}))
	}
	if err := Confirm(ctx, user, today, ids); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	for _, id := range ids[:3] {
		closeTask(t, user, id)
	}

	days, err := List(ctx, user, today, today)
	if err != nil || len(days) != 1 {
		t.Fatalf("list: %v %+v", err, days)
	}
	got := days[0]
	if got.Done != 3 || got.Target != 5 || got.Level != 3 || !got.Confirmed || len(got.Items) != 5 {
		t.Errorf("day = %+v, want done 3 of 5, level 3 (🟧), confirmed, 5 items", got)
	}
}

// Denis 22.09: a day never taken is ⬛ however much was done on it.
func TestAnUnconfirmedDayIsBlackEvenIfDone(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	id := newTask(t, user, map[string]any{"title": "без подтверждения"})
	if err := Add(ctx, user, today, id); err != nil {
		t.Fatalf("add: %v", err)
	}
	closeTask(t, user, id)
	days, _ := List(ctx, user, today, today)
	if days[0].Level != 0 || days[0].Confirmed {
		t.Errorf("day = %+v, want level 0 and not confirmed", days[0])
	}
}

// Spec §2: a done task cannot be added — «add the finished one, get green».
func TestADoneTaskCannotBeAdded(t *testing.T) {
	_, ctx, user := dayDB(t)
	id := newTask(t, user, map[string]any{"title": "уже сделано"})
	closeTask(t, user, id)
	if err := Add(ctx, user, Today(time.Now(), ny), id); err != ErrNotOpen {
		t.Errorf("add done task: err = %v, want ErrNotOpen", err)
	}
}

// Spec §2: a series counts its own day, marked through the series door.
func TestASeriesDayCountsWhenThatDayIsDone(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	id := newTask(t, user, map[string]any{"title": "таблетки", "rrule": "FREQ=DAILY",
		"due_date": today.Format(time.RFC3339)})
	if err := Confirm(ctx, user, today, []string{id}); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if err := tasks.MarkOccurrence(ctx, user, id, today, tasks.StateDone); err != nil {
		t.Fatalf("mark: %v", err)
	}
	days, _ := List(ctx, user, today, today)
	if days[0].Done != 1 {
		t.Errorf("series day not counted: %+v", days[0])
	}
	// The same series is open again tomorrow: adding it there is allowed.
	if err := Add(ctx, user, today.AddDate(0, 0, 1), id); err != nil {
		t.Errorf("tomorrow's day of the series refused: %v", err)
	}
}

// Spec §5: yesterday cannot be edited; tomorrow can.
func TestRemovingFollowsTheNoonRule(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	id := newTask(t, user, map[string]any{"title": "убрать"})
	for _, d := range []time.Time{today.AddDate(0, 0, -1), today.AddDate(0, 0, 1)} {
		if err := Add(ctx, user, d, id); err != nil {
			t.Fatalf("add %s: %v", d.Format("2006-01-02"), err)
		}
	}
	if err := Remove(ctx, user, today.AddDate(0, 0, -1), id, time.Now()); err != ErrTooLate {
		t.Errorf("remove from yesterday: err = %v, want ErrTooLate", err)
	}
	if err := Remove(ctx, user, today.AddDate(0, 0, 1), id, time.Now()); err != nil {
		t.Errorf("remove from tomorrow: %v", err)
	}
	days, _ := List(ctx, user, today.AddDate(0, 0, 1), today.AddDate(0, 0, 1))
	if len(days[0].Items) != 0 {
		t.Errorf("removed task still listed: %+v", days[0].Items)
	}
}

// Every day of the range is returned, empty ones as [] (gotcha 6).
func TestListReturnsEveryDayWithEmptyArrays(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	days, err := List(ctx, user, today, today.AddDate(0, 0, 2))
	if err != nil || len(days) != 3 {
		t.Fatalf("list: %v, %d days", err, len(days))
	}
	b, _ := json.Marshal(days[1])
	if !bytes.Contains(b, []byte(`"items":[]`)) {
		t.Errorf("empty day serialises as %s, want items []", b)
	}
}

// A stranger's task cannot enter my day.
func TestAnotherUsersTaskIsNotFound(t *testing.T) {
	_, ctx, user := dayDB(t)
	_, _, other := dayDB(t)
	theirs := newTask(t, other, map[string]any{"title": "чужое"})
	if err := Add(ctx, user, Today(time.Now(), ny), theirs); err != ErrTaskNotFound {
		t.Errorf("add stranger's task: err = %v, want ErrTaskNotFound", err)
	}
}
```

Проверено 22.09 по `tasks/types.go`: `CreateTaskRequest` принимает `title`, `priority` (`*int`),
`due_date` (`*string`, ISO 8601), `rrule`; `tasks.StateDone = "done"` экспортирован
(`occurrence.go:32`); `CreateHandler` отвечает `201`. `due_date` в тестах передавать RFC3339
полуночи New York — `today.Format(time.RFC3339)`, не голой датой.

- [ ] **Step 3: Run to see them fail**

Run: `cd api-go && DATABASE_URL="<из memory test-database.md>" go test ./internal/daytasks/ -count=1`
Expected: FAIL — `undefined: InitDB` (компиляция). После миграции `TestDayCommitmentTablesHaveTheirShape`
должен быть зелёным уже на Step 4.

- [ ] **Step 4: Implement `store.go`**

```go
package daytasks

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
)

var db *database.DB

// InitDB follows the package-level-pool pattern the other packages use.
func InitDB(d *database.DB) { db = d }

var (
	ErrTooLate       = errors.New("today's tasks can only be removed before noon; past days not at all")
	ErrNotOpen       = errors.New("only an open task can be promised for a day")
	ErrTaskNotFound  = errors.New("task not found")
	ErrRangeTooLarge = errors.New("range must be at most 62 days")
	ErrInvalidRange  = errors.New("from must be a date not after to")
)

type Item struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title"`
	Done   bool   `json:"done"`
}

type Day struct {
	Day       string `json:"day"`
	Target    int    `json:"target"`
	Confirmed bool   `json:"confirmed"`
	Items     []Item `json:"items"`
	Done      int    `json:"done"`
	Level     int    `json:"level"`
}

const dateFmt = "2006-01-02"

func userZoneAndTarget(ctx context.Context, userID string) (string, int, error) {
	var tz string
	var raw []byte
	err := db.Pool.QueryRow(ctx,
		`SELECT COALESCE(timezone, 'Europe/Moscow'), COALESCE(settings, '{}') FROM "user" WHERE id = $1`,
		userID).Scan(&tz, &raw)
	return tz, Target(raw), err
}

// doneOnDay is the one definition of «closed on that day» (spec §2), shared by
// List and the «only open» check so the two cannot disagree.
//   one-off: completed_at falls on the day in the user's zone
//   series:  that day of the series is answered «done»
const doneOnDay = `
	CASE WHEN COALESCE(t.rrule, '') <> ''
	     THEN EXISTS (SELECT 1 FROM task_occurrence o
	                   WHERE o.task_id = t.id AND o.occurrence = $DAY AND o.state = 'done')
	     ELSE t.status = 'DONE' AND (t.completed_at AT TIME ZONE $TZ)::date = $DAY
	END`

// List returns every day of [from, to], empty ones included.
func List(ctx context.Context, userID string, from, to time.Time) ([]Day, error) {
	if ymd(to) < ymd(from) {
		return nil, ErrInvalidRange
	}
	if to.Sub(from) > 62*24*time.Hour {
		return nil, ErrRangeTooLarge
	}
	tz, target, err := userZoneAndTarget(ctx, userID)
	if err != nil {
		return nil, err
	}

	byDay := map[string]*Day{}
	var out []Day
	for d := from; ymd(d) <= ymd(to); d = d.AddDate(0, 0, 1) {
		out = append(out, Day{Day: d.Format(dateFmt), Target: target, Items: []Item{}})
	}
	for i := range out {
		byDay[out[i].Day] = &out[i]
	}

	confirmed, err := db.Pool.Query(ctx, `
		SELECT to_char(day, 'YYYY-MM-DD') FROM day_commitment_day
		 WHERE user_id = $1 AND day BETWEEN $2 AND $3`,
		userID, from.Format(dateFmt), to.Format(dateFmt))
	if err != nil {
		return nil, err
	}
	for confirmed.Next() {
		var d string
		if err := confirmed.Scan(&d); err != nil {
			confirmed.Close()
			return nil, err
		}
		if day := byDay[d]; day != nil {
			day.Confirmed = true
		}
	}
	confirmed.Close()

	// $DAY is the row's own day, $TZ the user's zone: substituted by hand into
	// the shared fragment, parameters stay parameters.
	q := `
		SELECT to_char(c.day, 'YYYY-MM-DD'), t.id::text, t.title,
		       ` + replaceAll(doneOnDay, "$DAY", "c.day", "$TZ", "$4") + `
		  FROM day_commitment c
		  JOIN task t ON t.id = c.task_id
		 WHERE c.user_id = $1 AND c.day BETWEEN $2 AND $3 AND c.removed_at IS NULL
		 ORDER BY c.day, c.added_at`
	rows, err := db.Pool.Query(ctx, q, userID, from.Format(dateFmt), to.Format(dateFmt), tz)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d string
		var it Item
		if err := rows.Scan(&d, &it.TaskID, &it.Title, &it.Done); err != nil {
			return nil, err
		}
		if day := byDay[d]; day != nil {
			day.Items = append(day.Items, it)
			if it.Done {
				day.Done++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Confirmed {
			out[i].Level = Level(out[i].Done, out[i].Target)
		}
	}
	return out, nil
}

// Add promises an OPEN task, visible to the caller, for a day. Adding again
// what was removed brings it back.
func Add(ctx context.Context, userID string, day time.Time, taskID string) error {
	tz, _, err := userZoneAndTarget(ctx, userID)
	if err != nil {
		return err
	}
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return err
	}
	var done bool
	err = db.Pool.QueryRow(ctx, `
		SELECT `+replaceAll(doneOnDay, "$DAY", "$3::date", "$TZ", "$4")+`
		       OR (COALESCE(t.rrule, '') = '' AND t.status = 'DONE')
		  FROM task t WHERE t.id = $1 AND t.calendar_id = ANY($2)`,
		taskID, calIDs, day.Format(dateFmt), tz).Scan(&done)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTaskNotFound
	}
	if err != nil {
		return err
	}
	if done {
		return ErrNotOpen
	}
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO day_commitment (user_id, day, task_id) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, day, task_id) DO UPDATE SET removed_at = NULL`,
		userID, day.Format(dateFmt), taskID)
	return err
}

// Remove is spec §5: refused after noon today and on any past day.
func Remove(ctx context.Context, userID string, day time.Time, taskID string, now time.Time) error {
	tz, _, err := userZoneAndTarget(ctx, userID)
	if err != nil {
		return err
	}
	if !CanRemove(day, now, tz) {
		return ErrTooLate
	}
	_, err = db.Pool.Exec(ctx, `
		UPDATE day_commitment SET removed_at = NOW()
		 WHERE user_id = $1 AND day = $2 AND task_id = $3 AND removed_at IS NULL`,
		userID, day.Format(dateFmt), taskID)
	return err
}

// Confirm takes the day with these tasks. It only ADDS: removing goes through
// Remove and its noon rule, or «confirm» would be a way around it.
func Confirm(ctx context.Context, userID string, day time.Time, taskIDs []string) error {
	for _, id := range taskIDs {
		if err := Add(ctx, userID, day, id); err != nil {
			return err
		}
	}
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO day_commitment_day (user_id, day) VALUES ($1, $2)
		ON CONFLICT (user_id, day) DO NOTHING`, userID, day.Format(dateFmt))
	return err
}

func replaceAll(s string, pairs ...string) string {
	for i := 0; i+1 < len(pairs); i += 2 {
		s = strings.ReplaceAll(s, pairs[i], pairs[i+1])
	}
	return s
}
```

Добавь `"strings"` в import. Проверено 22.09: `"user".timezone TEXT DEFAULT 'Europe/Moscow'`
(`000002`), `task.status` — строка `'DONE'` (`tasks.StatusDone`).

- [ ] **Step 5: Run to see them pass**

Run: `cd api-go && DATABASE_URL=… go test ./internal/daytasks/ -count=1 -v`
Expected: все PASS, **ни одного SKIP** (SKIP = базы нет = тесты не шли).

- [ ] **Step 6: Sabotages** (по одному, каждый — компилируется и падает на своём тесте, потом вернуть):
  1. в `doneOnDay` убрать `AT TIME ZONE $TZ` → упасть обязан `TestADayCountsWhatWasClosedOnIt`
     **в часы, когда дата UTC ≠ дате New York** (00:00–04:00 UTC). Вне этого окна он пройдёт —
     поэтому добавь к тесту вторую проверку: задача, закрытая сегодня, и `List` на **вчера**
     не должен её засчитать. Если сабботаж не краснеет ни в какой час — тест не про зону.
  2. в `List` убрать `if out[i].Confirmed` → `TestAnUnconfirmedDayIsBlackEvenIfDone`.
  3. в `Add` убрать проверку `done` → `TestADoneTaskCannotBeAdded`.
  4. в `Remove` убрать вызов `CanRemove` → `TestRemovingFollowsTheNoonRule`.

- [ ] **Step 7: Commit**

```bash
git add api-go/migrations/000022_day_commitment.up.sql api-go/migrations/000022_day_commitment.down.sql \
        api-go/internal/daytasks/store.go api-go/internal/daytasks/db_test.go
git commit -m "feat(api): day-tasks store — a day counts what was closed on it, in the user's zone"
```

---

### Task 4: Предложение набора

**Files:**
- Create: `api-go/internal/daytasks/proposal.go`
- Modify: `api-go/internal/daytasks/db_test.go` (добавить тесты)

**Interfaces:**
- Consumes: `List`, `userZoneAndTarget`, `doneOnDay`.
- Produces: `func Propose(ctx, userID string, day time.Time) ([]Item, error)` — ≤ target, без дублей,
  только открытые; ничего не пишет.

Порядок (спека §4): 1) уже в наборе дня · 2) вчерашние невыполненные из набора · 3) разовые
просроченные и со сроком ≤ день · 4) серии, у которых этот день положен (`recurrence.Occurs`) и
не отвечен · 5) остальные открытые — приоритет 1…5, затем 0, затем `created_at`.

- [ ] **Step 1: Failing test**

```go
// Spec §4: already-promised first, then yesterday's undone, then due, then
// priority; never a done task, never more than N.
func TestProposalOrder(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	yesterday := today.AddDate(0, 0, -1)

	pinned := newTask(t, user, map[string]any{"title": "заранее", "priority": 5})
	carried := newTask(t, user, map[string]any{"title": "вчерашнее", "priority": 5})
	due := newTask(t, user, map[string]any{"title": "срок сегодня", "priority": 4,
		"due_date": today.Format(time.RFC3339)})
	urgent := newTask(t, user, map[string]any{"title": "срочно", "priority": 1})
	buffer := newTask(t, user, map[string]any{"title": "буфер", "priority": 0})
	done := newTask(t, user, map[string]any{"title": "сделано", "priority": 1})
	extra := newTask(t, user, map[string]any{"title": "лишнее", "priority": 3})
	closeTask(t, user, done)

	if err := Add(ctx, user, today, pinned); err != nil {
		t.Fatal(err)
	}
	if err := Confirm(ctx, user, yesterday, []string{carried}); err != nil {
		t.Fatal(err)
	}

	got, err := Propose(ctx, user, today)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	var ids []string
	for _, it := range got {
		ids = append(ids, it.TaskID)
	}
	want := []string{pinned, carried, due, urgent, extra} // N=5: buffer is sixth, done never
	if fmt.Sprint(ids) != fmt.Sprint(want) {
		t.Errorf("proposal = %v\nwant       %v (pinned, carried, due, urgent, extra)", ids, want)
	}
	_ = buffer

	// Proposing writes nothing.
	days, _ := List(ctx, user, today, today)
	if days[0].Confirmed || len(days[0].Items) != 1 {
		t.Errorf("proposal wrote something: %+v", days[0])
	}
}
```

- [ ] **Step 2: Run to see it fail** — `undefined: Propose`.

- [ ] **Step 3: Implement**

```go
package daytasks

import (
	"context"
	"sort"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/recurrence"
)

type candidate struct {
	Item
	rank     int // 1..5, spec §4
	priority int
	created  time.Time
}

// Propose is spec §4 — what the morning offers, in order, up to N. It writes
// nothing: the day is taken only by Confirm.
func Propose(ctx context.Context, userID string, day time.Time) ([]Item, error) {
	tz, target, err := userZoneAndTarget(ctx, userID)
	if err != nil {
		return nil, err
	}
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	d := day.Format(dateFmt)
	y := day.AddDate(0, 0, -1).Format(dateFmt)

	rows, err := db.Pool.Query(ctx, `
		-- $1 calendars · $2 unused (kept NULL so the numbering below reads as the
		-- spec's order) · $3 day · $4 zone · $5 user · $6 yesterday
		SELECT t.id::text, t.title, COALESCE(t.priority, 0), t.created_at,
		       COALESCE(t.rrule, ''), t.repeat_anchor,
		       (t.due_date AT TIME ZONE $4)::date <= $3::date AS due,
		       EXISTS (SELECT 1 FROM day_commitment c WHERE c.user_id = $5 AND c.task_id = t.id
		                 AND c.day = $3::date AND c.removed_at IS NULL) AS pinned,
		       EXISTS (SELECT 1 FROM day_commitment c WHERE c.user_id = $5 AND c.task_id = t.id
		                 AND c.day = $6::date AND c.removed_at IS NULL) AS yesterday
		  FROM task t
		 WHERE t.calendar_id = ANY($1)
		   AND NOT (`+replaceAll(doneOnDay, "$DAY", "$3::date", "$TZ", "$4")+`)
		   AND NOT (COALESCE(t.rrule, '') = '' AND t.status = 'DONE')`,
		calIDs, nil, d, tz, userID, y)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cands []candidate
	for rows.Next() {
		var c candidate
		var rrule string
		var anchor *time.Time
		var due *bool
		var pinned, yesterday bool
		if err := rows.Scan(&c.TaskID, &c.Title, &c.priority, &c.created, &rrule, &anchor,
			&due, &pinned, &yesterday); err != nil {
			return nil, err
		}
		series := rrule != ""
		switch {
		case pinned:
			c.rank = 1
		case yesterday:
			c.rank = 2
		case !series && due != nil && *due:
			c.rank = 3
		case series && anchor != nil && occursOn(rrule, *anchor, day):
			c.rank = 4
		case series:
			continue // a series not due today is not today's task
		default:
			c.rank = 5
		}
		cands = append(cands, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Priority: 1 is the most urgent … 5; 0 is the buffer and goes last (gotcha 4).
	pri := func(p int) int {
		if p == 0 {
			return 99
		}
		return p
	}
	sort.SliceStable(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if pri(a.priority) != pri(b.priority) {
			return pri(a.priority) < pri(b.priority)
		}
		return a.created.Before(b.created)
	})

	out := []Item{}
	for _, c := range cands {
		if len(out) == target {
			break
		}
		out = append(out, c.Item)
	}
	return out, nil
}

func occursOn(rrule string, anchor, day time.Time) bool {
	rule, err := recurrence.Parse(rrule)
	if err != nil {
		return false
	}
	return recurrence.Occurs(rule, anchor, day)
}
```

⚠ pgx не может вывести тип параметра, который не встречается в запросе: если `$2 = nil` даст
`could not determine data type of parameter $2`, перенумеруй на `$1…$5` (day → `$2`, zone → `$3`,
user → `$4`, yesterday → `$5`) и поправь `replaceAll` на `"$2::date", "$3"`. Проверено 22.09:
`task.due_date` — `TIMESTAMPTZ` (`000001_baseline.up.sql:76`), поэтому `AT TIME ZONE` у него нужен.
⚠ Приоритет 1…5 внутри ранга 1–4 тоже сортирует — порядок внутри «вчерашних» и «со сроком»
по срочности. Тест проверяет только ранги; не добавляй ранжирование, которого спека не просит.

- [ ] **Step 4: Run to see it pass**, без SKIP.

- [ ] **Step 5: Sabotage** — поменять `rank = 2` и `rank = 3` местами → тест обязан упасть на
  порядке `carried`/`due`. Вернуть.

- [ ] **Step 6: Commit**

```bash
git add api-go/internal/daytasks/proposal.go api-go/internal/daytasks/db_test.go
git commit -m "feat(api): day-tasks proposal — promised, yesterday's undone, due, then by priority"
```

---

### Task 5: Хэндлеры и роуты

**Files:**
- Create: `api-go/internal/daytasks/handlers.go`
- Modify: `api-go/cmd/api/main.go` (рядом с `broadcast.InitDB(db)` и рядом с роутами задач)
- Modify: `api-go/internal/daytasks/db_test.go` (HTTP-тесты)

**Interfaces:**
- Consumes: `List`, `Propose`, `Add`, `Remove`, `Confirm`, ошибки.
- Produces (роуты, авторизованная группа — там же, где `/api/tasks`):
  - `GET /api/day-tasks?from=&to=` → `[]Day`
  - `GET /api/day-tasks/proposal?day=` → `[]Item`
  - `POST /api/day-tasks/confirm` `{"day","task_ids"}` → `Day`
  - `POST /api/day-tasks` `{"day","task_id"}` → `Day`
  - `DELETE /api/day-tasks/{day}/{task_id}` → `Day`
  - Коды: `INVALID_DAY` 400 · `RANGE_TOO_LARGE` 400 · `INVALID_RANGE` 400 · `NOT_OPEN` 409 ·
    `TOO_LATE` 409 · `TASK_NOT_FOUND` 404 · `NOT_AUTHENTICATED` 401.

- [ ] **Step 1: Failing HTTP tests** (через хэндлеры — дверь, которой пойдёт бот)

```go
func callDay(t *testing.T, h http.HandlerFunc, userID, method, target string, body any, params map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := asUser(httptest.NewRequest(method, target, rd), userID)
	if params != nil {
		rctx := chi.NewRouteContext()
		for k, v := range params {
			rctx.URLParams.Add(k, v)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// The bot's whole path through HTTP: propose → confirm → close → read the colour.
func TestTheDayThroughHTTP(t *testing.T) {
	_, _, user := dayDB(t)
	today := Today(time.Now(), ny).Format("2006-01-02")
	id := newTask(t, user, map[string]any{"title": "одно дело", "priority": 1})

	rec := callDay(t, ProposalHandler, user, http.MethodGet, "/api/day-tasks/proposal?day="+today, nil, nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(id)) {
		t.Fatalf("proposal: %d %s", rec.Code, rec.Body.String())
	}
	rec = callDay(t, ConfirmHandler, user, http.MethodPost, "/api/day-tasks/confirm",
		map[string]any{"day": today, "task_ids": []string{id}}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body.String())
	}
	closeTask(t, user, id)
	rec = callDay(t, ListHandler, user, http.MethodGet, "/api/day-tasks?from="+today+"&to="+today, nil, nil)
	var got struct {
		Data []Day `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != http.StatusOK || len(got.Data) != 1 || got.Data[0].Done != 1 || got.Data[0].Level != 1 {
		t.Errorf("list: %d %+v — want 1 of 5 done, level 1 (🟫)", rec.Code, got.Data)
	}
}

func TestHTTPErrorsHaveTheirCodes(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
	id := newTask(t, user, map[string]any{"title": "x"})
	if err := Add(ctx, user, today.AddDate(0, 0, -1), id); err != nil {
		t.Fatal(err)
	}
	done := newTask(t, user, map[string]any{"title": "done"})
	closeTask(t, user, done)

	cases := []struct {
		name string
		rec  *httptest.ResponseRecorder
		code int
		err  string
	}{
		{"remove from yesterday", callDay(t, RemoveHandler, user, http.MethodDelete, "/", nil,
			map[string]string{"day": yesterday, "task_id": id}), 409, "TOO_LATE"},
		{"add done", callDay(t, AddHandler, user, http.MethodPost, "/",
			map[string]any{"day": today.Format("2006-01-02"), "task_id": done}, nil), 409, "NOT_OPEN"},
		{"bad day", callDay(t, AddHandler, user, http.MethodPost, "/",
			map[string]any{"day": "22.09.2026", "task_id": id}, nil), 400, "INVALID_DAY"},
		{"huge range", callDay(t, ListHandler, user, http.MethodGet,
			"/?from=2026-01-01&to=2026-12-31", nil, nil), 400, "RANGE_TOO_LARGE"},
		{"no user", callDay(t, ListHandler, "", http.MethodGet, "/?from=2026-01-01&to=2026-01-02", nil, nil), 401, "NOT_AUTHENTICATED"},
	}
	for _, c := range cases {
		if c.rec.Code != c.code || !bytes.Contains(c.rec.Body.Bytes(), []byte(c.err)) {
			t.Errorf("%s: %d %s — want %d %s", c.name, c.rec.Code, c.rec.Body.String(), c.code, c.err)
		}
	}
}
```

- [ ] **Step 2: Run to see them fail** — `undefined: ProposalHandler`.

- [ ] **Step 3: Implement `handlers.go`**

```go
package daytasks

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

func authed(w http.ResponseWriter, r *http.Request) (string, bool) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return "", false
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return "", false
	}
	return userID, true
}

func parseDay(w http.ResponseWriter, s string) (time.Time, bool) {
	d, err := time.Parse(dateFmt, s)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_DAY", "day must be YYYY-MM-DD")
		return time.Time{}, false
	}
	return d, true
}

func respondErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTooLate):
		util.RespondError(w, http.StatusConflict, "TOO_LATE", err.Error())
	case errors.Is(err, ErrNotOpen):
		util.RespondError(w, http.StatusConflict, "NOT_OPEN", err.Error())
	case errors.Is(err, ErrTaskNotFound):
		util.RespondError(w, http.StatusNotFound, "TASK_NOT_FOUND", err.Error())
	case errors.Is(err, ErrRangeTooLarge):
		util.RespondError(w, http.StatusBadRequest, "RANGE_TOO_LARGE", err.Error())
	case errors.Is(err, ErrInvalidRange):
		util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", err.Error())
	default:
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Day tasks failed")
	}
}

// respondDay answers a write with the day as it now is — the bot redraws from
// it without a second request.
func respondDay(w http.ResponseWriter, r *http.Request, userID string, day time.Time) {
	days, err := List(r.Context(), userID, day, day)
	if err != nil {
		respondErr(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, days[0])
}

// ListHandler — GET /api/day-tasks?from=&to=
func ListHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	from, ferr := time.Parse(dateFmt, r.URL.Query().Get("from"))
	to, terr := time.Parse(dateFmt, r.URL.Query().Get("to"))
	if ferr != nil || terr != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "from and to must be YYYY-MM-DD")
		return
	}
	days, err := List(r.Context(), userID, from, to)
	if err != nil {
		respondErr(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, days)
}

// ProposalHandler — GET /api/day-tasks/proposal?day=
func ProposalHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	day, ok := parseDay(w, r.URL.Query().Get("day"))
	if !ok {
		return
	}
	items, err := Propose(r.Context(), userID, day)
	if err != nil {
		respondErr(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, items)
}

type writeRequest struct {
	Day     string   `json:"day"`
	TaskID  string   `json:"task_id"`
	TaskIDs []string `json:"task_ids"`
}

func decode(w http.ResponseWriter, r *http.Request) (writeRequest, bool) {
	var req writeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return req, false
	}
	return req, true
}

// ConfirmHandler — POST /api/day-tasks/confirm {"day","task_ids"}
func ConfirmHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	req, ok := decode(w, r)
	if !ok {
		return
	}
	day, ok := parseDay(w, req.Day)
	if !ok {
		return
	}
	if err := Confirm(r.Context(), userID, day, req.TaskIDs); err != nil {
		respondErr(w, err)
		return
	}
	respondDay(w, r, userID, day)
}

// AddHandler — POST /api/day-tasks {"day","task_id"}
func AddHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	req, ok := decode(w, r)
	if !ok {
		return
	}
	day, ok := parseDay(w, req.Day)
	if !ok {
		return
	}
	if err := Add(r.Context(), userID, day, req.TaskID); err != nil {
		respondErr(w, err)
		return
	}
	respondDay(w, r, userID, day)
}

// RemoveHandler — DELETE /api/day-tasks/{day}/{task_id}
func RemoveHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	day, ok := parseDay(w, chi.URLParam(r, "day"))
	if !ok {
		return
	}
	if err := Remove(r.Context(), userID, day, chi.URLParam(r, "task_id"), time.Now()); err != nil {
		respondErr(w, err)
		return
	}
	respondDay(w, r, userID, day)
}
```

⚠ `AddHandler` и `ConfirmHandler` для чужого `task_id`, не-UUID — pgx вернёт ошибку каста, а не
`ErrNoRows` → 500. Если тест «no user»/«bad id» это покажет — ловить в `Add`: невалидный UUID
→ `ErrTaskNotFound` (проверка `uuid` регэкспом или `pgconn` код `22P02`).

- [ ] **Step 4: Routes** — в `cmd/api/main.go`:

```go
	broadcast.InitDB(db)
	daytasks.InitDB(db)
```

и в той же авторизованной группе, что `r.Get("/api/tasks", t.ListHandler)`:

```go
		// «Задачи дня» (spec 2026-09-22). /proposal and /confirm before the
		// {day} route is irrelevant for chi (different methods), but kept first
		// so a reader sees the fixed paths before the pattern.
		r.Get("/api/day-tasks", daytasks.ListHandler)
		r.Get("/api/day-tasks/proposal", daytasks.ProposalHandler)
		r.Post("/api/day-tasks/confirm", daytasks.ConfirmHandler)
		r.Post("/api/day-tasks", daytasks.AddHandler)
		r.Delete("/api/day-tasks/{day}/{task_id}", daytasks.RemoveHandler)
```

и импорт `"neuroboost/api-go/internal/daytasks"`.

- [ ] **Step 5: Full verify** — оба модуля (проектное правило):

```bash
cd api-go && gofmt -l internal/daytasks cmd/api && go build ./... && go vet ./... \
  && DATABASE_URL=… go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL|---)" | sort | uniq -c
cd ../bot && go build ./... && go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL)"
```

Expected: ноль `FAIL`; `daytasks` — `ok` с реальным прогоном (проверить `-v` на отсутствие SKIP).

- [ ] **Step 6: Sabotage** — в `respondErr` поменять `StatusConflict` у `TOO_LATE` на
  `StatusBadRequest` → `TestHTTPErrorsHaveTheirCodes` обязан упасть на «remove from yesterday». Вернуть.

- [ ] **Step 7: Commit** (без push — push пересобирает staging и сбрасывает dev-базу; только по слову Дениса)

```bash
git add api-go/internal/daytasks/handlers.go api-go/internal/daytasks/db_test.go api-go/cmd/api/main.go
git commit -m "feat(api): day-tasks endpoints — list, propose, confirm, add, remove before noon"
```

---

## Самопроверка плана против спеки

| Спека | Задача |
|---|---|
| §1 набор, подтверждение, ⬛ без него | T3 (`Confirm`, `Level` только при `Confirmed`) |
| §2 «сделано» в зоне, серия, только открытая | T3 (`doneOnDay`, `Add`) |
| §3 уровни N = 3…7 | T1 |
| §4 предложение по порядку | T4 |
| §5 полдень | T2 (`CanRemove`), T3 (`Remove`), T5 (409) |
| §7 таблицы, `settings.day_tasks_target`, ручки | T3, T2 (`Target`), T5 |
| §6 клетка календаря, §8 бот, онбординг | **не D1** — это D2–D4 |

Запись `day_tasks_target` идёт существующим `PATCH /api/auth/me` (бот — через `MergeSettings`,
gotcha 21); новой ручки для неё нет.
