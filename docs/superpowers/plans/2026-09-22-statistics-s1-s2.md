<!-- паспорт: тип=план | статус=действует | строк=1222 | ~токенов=9782 | обновлён=по git -->

# Статистика S1 + S2 — данные и расчёт сетки · план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** API отдаёт отмеченные дни серий за диапазон, бот умеет их читать вместе с рефлексиями, а
чистый пакет `statgrid` превращает события/задачи/рефлексии в сетку уровней `·▁▂▃▄▅▆▇█` для
любого периода и шкалы. Экрана ещё нет — это S3.

**Architecture:** S1 — одна ручка чтения в `api-go/internal/tasks` и два метода клиента бота.
S2 — новый пакет `bot/internal/statgrid` без зависимостей от Telegram и хэндлеров: `Layout` строит
ячейки периода в зоне пользователя, `Busy`/`Planned` меряют время в ячейке (объединение и сумма),
`Levels` переводит значения в уровни по шкале. S3 соберёт из этого экран.

**Tech Stack:** Go (api-go: chi, pgx; bot: stdlib).

**Spec:** `docs/superpowers/specs/2026-09-22-statistics-design.md` §1–§3, §5, §6.

## Global Constraints

- 🔴 **Без миграций.** Нужна — остановиться и спросить.
- 🔴 Всякое «день», «неделя», «час» — в **зоне пользователя** (`*time.Location` параметром), не
  сервера и не машины. Границы суток — `time.Date(y, m, d, 0,0,0,0, loc)` и `AddDate`, не `+24h`
  (переход на зимнее время).
- 🔴 «Занято» — **объединение** интервалов, «В планах» — **сумма** (спека §1, слова Дениса 20.09).
- Символы: `·` и `▁▂▃▄▅▆▇█`; больше 100 % → `█`.
- Шкала: `day24` (по умолчанию) · `work` · `peak` (спека §3).
- Задача: время = `actual_minutes`, иначе `estimated_minutes`, иначе **30 мин** (спека §2).
- DB-тесты api-go: `DATABASE_URL` + `-count=1`; пакет `tasks` уже гоняет тесты на UTC-часах
  (`main_test.go`), «сегодня» пользователя — `userToday()`.
- Проверять **ноль `FAIL`**, саботаж каждого нового теста компилируется и читается целиком.
- Правки — Edit/Write; `git add` конкретных файлов; **не пушить** — Денис проверяет dev.

---

### Task 1 (S1): `GET /api/tasks/occurrences`

**Files:**
- Modify: `api-go/internal/tasks/occurrence.go` (функция `ListOccurrences`)
- Modify: `api-go/internal/tasks/occurrence_handlers.go` (`ListOccurrencesHandler`)
- Modify: `api-go/cmd/api/main.go` (роут рядом с `r.Get("/api/tasks", …)`)
- Create: `api-go/internal/tasks/occurrence_list_test.go`

**Interfaces — Produces:**

```go
type OccurrenceRow struct {
	TaskID     string `json:"task_id"`
	Occurrence string `json:"occurrence"` // YYYY-MM-DD
	State      string `json:"state"`      // done | skipped
}
var ErrRangeTooLarge = errors.New("range must be at most 366 days")
func ListOccurrences(ctx context.Context, userID string, from, to time.Time) ([]OccurrenceRow, error)
// GET /api/tasks/occurrences?from=YYYY-MM-DD&to=YYYY-MM-DD  (both inclusive)
// 200 {"data":[…]} · 400 INVALID_RANGE (unparseable / to<from) · 400 RANGE_TOO_LARGE
```

- [ ] **Step 1: Падающий тест** — `occurrence_list_test.go`:

```go
package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"neuroboost/api-go/internal/middleware"
)

func callListOccurrences(userID, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/occurrences?"+query, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
	rec := httptest.NewRecorder()
	ListOccurrencesHandler(rec, req)
	return rec
}

// Statistics counts the days a series was actually done (spec 22.09 §5). The
// days come through the same door the bot will use, and only the caller's.
func TestOccurrencesAreListedForARangeAndOnlyTheCallers(t *testing.T) {
	d, ctx, user := repeatDB(t)
	stranger := seedTaskUser(t, ctx, d, "stranger")

	day0 := userToday()
	task := seriesTask(t, ctx, user, "FREQ=DAILY", day0)
	for _, off := range []int{0, 1, 3} {
		if err := MarkOccurrence(ctx, user, task.ID, day0.AddDate(0, 0, off), StateDone); err != nil {
			t.Fatalf("mark +%d: %v", off, err)
		}
	}
	if err := MarkOccurrence(ctx, user, task.ID, day0.AddDate(0, 0, 2), StateSkipped); err != nil {
		t.Fatalf("skip: %v", err)
	}

	q := "from=" + day0.Format("2006-01-02") + "&to=" + day0.AddDate(0, 0, 2).Format("2006-01-02")
	rec := callListOccurrences(user, q)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Data []OccurrenceRow `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Data) != 3 {
		t.Fatalf("rows = %+v, want 3 (days 0,1 done; 2 skipped; 3 is outside the range)", got.Data)
	}
	states := map[string]string{}
	for _, r := range got.Data {
		states[r.Occurrence] = r.State
	}
	if states[day0.AddDate(0, 0, 2).Format("2006-01-02")] != StateSkipped {
		t.Errorf("states = %v", states)
	}

	other := callListOccurrences(stranger, q)
	var none struct {
		Data []OccurrenceRow `json:"data"`
	}
	_ = json.Unmarshal(other.Body.Bytes(), &none)
	if other.Code != http.StatusOK || len(none.Data) != 0 {
		t.Errorf("a stranger sees %d rows (status %d)", len(none.Data), other.Code)
	}
}

func TestOccurrencesRefuseABadOrHugeRange(t *testing.T) {
	_, _, user := repeatDB(t)
	for q, code := range map[string]string{
		"from=2026-01-01&to=2027-06-01": "RANGE_TOO_LARGE",
		"from=2026-02-01&to=2026-01-01": "INVALID_RANGE",
		"from=yesterday&to=2026-01-01":  "INVALID_RANGE",
		"":                              "INVALID_RANGE",
	} {
		rec := callListOccurrences(user, q)
		if rec.Code != http.StatusBadRequest || !json.Valid(rec.Body.Bytes()) ||
			!containsCode(rec.Body.Bytes(), code) {
			t.Errorf("%q → %d %s, want 400 %s", q, rec.Code, rec.Body.String(), code)
		}
	}
}

func containsCode(body []byte, code string) bool {
	var e struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	return json.Unmarshal(body, &e) == nil && e.Error.Code == code
}
```

⚠ `seriesTask` и `userToday` уже есть в пакете (`convert_db_test.go`, `main_test.go`).
⚠ `MarkOccurrence(ctx, userID, taskID, day time.Time, state)` проверяет, что день в серии — все
четыре дня ежедневной серии в ней.

- [ ] **Step 2:** `DATABASE_URL=… go test -count=1 -run Occurrences ./internal/tasks/` → не компилируется.

- [ ] **Step 3: Реализация.** В `occurrence.go`:

```go
// OccurrenceRow is one answered day of a series.
type OccurrenceRow struct {
	TaskID     string `json:"task_id"`
	Occurrence string `json:"occurrence"`
	State      string `json:"state"`
}

// ErrRangeTooLarge bounds a read that statistics repeats per year for «Всё».
var ErrRangeTooLarge = errors.New("range must be at most 366 days")

// ListOccurrences returns the answered days of every series the caller can see.
//
// READ access (CalendarIDsFor), unlike marking a day: statistics shows a
// shared calendar's series to everyone who can see it.
func ListOccurrences(ctx context.Context, userID string, from, to time.Time) ([]OccurrenceRow, error) {
	if to.Before(from) {
		return nil, ErrInvalidRange
	}
	if to.Sub(from) > 366*24*time.Hour {
		return nil, ErrRangeTooLarge
	}
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT o.task_id::text, to_char(o.occurrence, 'YYYY-MM-DD'), o.state
		  FROM task_occurrence o
		  JOIN task t ON t.id = o.task_id
		 WHERE t.calendar_id = ANY($1) AND o.occurrence BETWEEN $2 AND $3
		 ORDER BY o.occurrence`,
		calIDs, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OccurrenceRow{}
	for rows.Next() {
		var r OccurrenceRow
		if err := rows.Scan(&r.TaskID, &r.Occurrence, &r.State); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
```

`ErrInvalidRange` — объявить рядом: `var ErrInvalidRange = errors.New("from must be a date not after to")`
(если в пакете уже есть одноимённая — использовать её).

⚠ `to_char` на колонке `DATE` не зависит от зоны сессии — это не `timestamptz`; дата выходит та,
что записана (урок `learning-my-own-query-lied-twice-in-one-night` был про `timestamptz`).

В `occurrence_handlers.go`:

```go
// ListOccurrencesHandler handles GET /api/tasks/occurrences?from=&to= (dates, inclusive).
func ListOccurrencesHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}
	from, ferr := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	to, terr := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if ferr != nil || terr != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "from and to must be YYYY-MM-DD")
		return
	}
	rows, err := ListOccurrences(r.Context(), userID, from, to)
	switch {
	case errors.Is(err, ErrRangeTooLarge):
		util.RespondError(w, http.StatusBadRequest, "RANGE_TOO_LARGE", err.Error())
	case errors.Is(err, ErrInvalidRange):
		util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", err.Error())
	case err != nil:
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list occurrences")
	default:
		util.RespondJSON(w, http.StatusOK, rows)
	}
}
```

`main.go`: `r.Get("/api/tasks/occurrences", t.ListOccurrencesHandler)` — **до** `r.Get("/api/tasks/{id}", …)`
(chi и так предпочтёт статический сегмент, но порядок читается).

- [ ] **Step 4:** тесты → PASS; `go test -count=1 ./internal/tasks/ ./internal/calendars/` → ноль `FAIL`.
- [ ] **Step 5: Саботаж:** `CalendarIDsFor` → запрос без `WHERE t.calendar_id = ANY($1)` (оставив `$1` в запросе как `$1::text[] IS NOT NULL`) → «a stranger sees» обязан упасть. Откатить.
- [ ] **Step 6: Commit** — `feat(api): GET /api/tasks/occurrences — the answered days of series, for statistics`

---

### Task 2 (S1): клиент бота — дни серий, рефлексии, поля задачи

**Files:**
- Modify: `bot/internal/api/client.go` (`type Task`: `ActualMinutes`, `RepeatAnchor`)
- Create: `bot/internal/api/stats.go`, `bot/internal/api/stats_test.go`

**Interfaces — Produces:**

```go
// in Task:
ActualMinutes int    `json:"actual_minutes,omitempty"`
RepeatAnchor  string `json:"repeat_anchor,omitempty"` // RFC3339

type TaskOccurrence struct {
	TaskID     string `json:"task_id"`
	Occurrence string `json:"occurrence"`
	State      string `json:"state"`
}
func (c *Client) TaskOccurrences(token, from, to string) ([]TaskOccurrence, error) // YYYY-MM-DD

type Reflection struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"` // RFC3339
}
func (c *Client) Reflections(token string) ([]Reflection, error)
```

- [ ] **Step 1: Падающий тест** — `stats_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Both answers are {"data": …}; a method that decodes the bare body reports
// success with nothing in it — the OccurrenceResult lesson of 21.09.
func TestStatsReadsTheEnvelopes(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/tasks/occurrences":
			query = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"data":[{"task_id":"t1","occurrence":"2026-09-21","state":"done"}]}`))
		case "/api/reflections":
			_, _ = w.Write([]byte(`{"data":[{"id":"r1","created_at":"2026-09-21T18:00:00Z"}]}`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL)

	occ, err := c.TaskOccurrences("tok", "2026-09-21", "2026-09-27")
	if err != nil || len(occ) != 1 || occ[0].State != "done" {
		t.Errorf("occurrences = %+v, %v", occ, err)
	}
	if query != "from=2026-09-21&to=2026-09-27" {
		t.Errorf("query = %q", query)
	}
	refl, err := c.Reflections("tok")
	if err != nil || len(refl) != 1 || refl[0].CreatedAt == "" {
		t.Errorf("reflections = %+v, %v", refl, err)
	}
}
```

- [ ] **Step 2:** `cd bot && go test -count=1 ./internal/api/` → не компилируется.
- [ ] **Step 3: Реализация** — `stats.go`:

```go
package api

import "net/url"

// What statistics reads beyond events and tasks (spec 22.09 §5, §2).

// TaskOccurrence is one answered day of a repeating task.
type TaskOccurrence struct {
	TaskID     string `json:"task_id"`
	Occurrence string `json:"occurrence"`
	State      string `json:"state"`
}

// TaskOccurrences lists answered days in [from, to], dates YYYY-MM-DD. The API
// refuses more than 366 days; «Всё» asks year by year.
func (c *Client) TaskOccurrences(token, from, to string) ([]TaskOccurrence, error) {
	var resp struct {
		Data []TaskOccurrence `json:"data"`
	}
	params := url.Values{}
	params.Set("from", from)
	params.Set("to", to)
	err := c.get("/api/tasks/occurrences", token, params, &resp)
	return resp.Data, err
}

// Reflection is what statistics needs of one: the day it was written.
type Reflection struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

// Reflections lists every reflection of the caller.
func (c *Client) Reflections(token string) ([]Reflection, error) {
	var resp struct {
		Data []Reflection `json:"data"`
	}
	err := c.get("/api/reflections", token, nil, &resp)
	return resp.Data, err
}
```

⚠ Проверить, что `c.get` принимает `nil` для `params`; если нет — `url.Values{}`.
⚠ `url.Values.Encode` сортирует ключи: `from` раньше `to`, как ждёт тест.

В `type Task` добавить `ActualMinutes` и `RepeatAnchor` (как в Interfaces).

- [ ] **Step 4:** тест → PASS. **Саботаж:** в `Reflections` декодировать в `[]Reflection` без конверта → `FAIL`; откатить.
- [ ] **Step 5: Commit** — `feat(bot): read series days, reflections and task times for statistics`

---

### Task 3 (S2): время в ячейке — объединение и сумма

**Files:**
- Create: `bot/internal/statgrid/span.go`, `bot/internal/statgrid/span_test.go`

**Interfaces — Produces:**

```go
package statgrid
type Span struct{ Start, End time.Time }
// Busy is the union of spans clipped to [from, to): two events at one time count once.
func Busy(spans []Span, from, to time.Time) time.Duration
// Planned is the plain sum of spans clipped to [from, to).
func Planned(spans []Span, from, to time.Time) time.Duration
```

- [ ] **Step 1: Падающие тесты** — `span_test.go`:

```go
package statgrid

import (
	"testing"
	"time"
)

func at(h, m int) time.Time { return time.Date(2026, 9, 21, h, m, 0, 0, time.UTC) }

// Spec §7, first line: 10:00–11:00 and 10:30–11:30 → busy 1.5 h, planned 2 h.
func TestOverlapsCountOnceInBusyAndTwiceInPlanned(t *testing.T) {
	spans := []Span{{at(10, 0), at(11, 0)}, {at(10, 30), at(11, 30)}}
	from, to := at(0, 0), at(23, 59)
	if b := Busy(spans, from, to); b != 90*time.Minute {
		t.Errorf("busy = %v, want 1h30m", b)
	}
	if p := Planned(spans, from, to); p != 2*time.Hour {
		t.Errorf("planned = %v, want 2h", p)
	}
}

func TestSpansAreClippedToTheCell(t *testing.T) {
	spans := []Span{{at(9, 30), at(12, 0)}}
	if b := Busy(spans, at(10, 0), at(11, 0)); b != time.Hour {
		t.Errorf("busy in 10–11 = %v, want 1h", b)
	}
	if b := Busy(spans, at(13, 0), at(14, 0)); b != 0 {
		t.Errorf("busy outside = %v", b)
	}
}

func TestContainedAndTouchingSpans(t *testing.T) {
	spans := []Span{{at(8, 0), at(12, 0)}, {at(9, 0), at(10, 0)}, {at(12, 0), at(13, 0)}}
	if b := Busy(spans, at(0, 0), at(23, 0)); b != 5*time.Hour {
		t.Errorf("busy = %v, want 5h (contained counts once, touching adds)", b)
	}
}
```

- [ ] **Step 2:** `cd bot && go test -count=1 ./internal/statgrid/` → не компилируется.
- [ ] **Step 3: Реализация** — `span.go`:

```go
// Package statgrid turns events, tasks and reflections into the numbers the
// statistics screen draws (spec 22.09). Pure: no Telegram, no API, no clock —
// every «day» and «hour» arrives already placed in the user's zone.
package statgrid

import (
	"sort"
	"time"
)

// Span is one booked interval.
type Span struct{ Start, End time.Time }

func clip(s Span, from, to time.Time) (Span, bool) {
	if s.Start.Before(from) {
		s.Start = from
	}
	if s.End.After(to) {
		s.End = to
	}
	return s, s.End.After(s.Start)
}

// Busy is the time covered by at least one span inside [from, to).
//
// 🔴 A union, not a sum — Denis 20.09: «не считал дважды события в одно и то
// же время». Planned is the sum; the gap between them is the overlap.
func Busy(spans []Span, from, to time.Time) time.Duration {
	var in []Span
	for _, s := range spans {
		if c, ok := clip(s, from, to); ok {
			in = append(in, c)
		}
	}
	sort.Slice(in, func(i, j int) bool { return in[i].Start.Before(in[j].Start) })
	var total time.Duration
	var cur Span
	for i, s := range in {
		if i == 0 {
			cur = s
			continue
		}
		if !s.Start.After(cur.End) {
			if s.End.After(cur.End) {
				cur.End = s.End
			}
			continue
		}
		total += cur.End.Sub(cur.Start)
		cur = s
	}
	if len(in) > 0 {
		total += cur.End.Sub(cur.Start)
	}
	return total
}

// Planned is the plain sum of the spans' time inside [from, to).
func Planned(spans []Span, from, to time.Time) time.Duration {
	var total time.Duration
	for _, s := range spans {
		if c, ok := clip(s, from, to); ok {
			total += c.End.Sub(c.Start)
		}
	}
	return total
}
```

- [ ] **Step 4:** PASS. **Саботаж:** `Busy` вернуть `Planned(spans, from, to)` → первый тест `FAIL`; откатить.
- [ ] **Step 5: Commit** — `feat(bot): statgrid — busy time is a union, planned time a sum`

---

### Task 4 (S2): ячейки периода

**Files:**
- Create: `bot/internal/statgrid/layout.go`, `bot/internal/statgrid/layout_test.go`

**Interfaces — Produces:**

```go
type Period string
const (Week Period = "week"; Month Period = "month"; Year Period = "year"; All Period = "all")

type Scale struct {
	Kind      string // "day24" | "work" | "peak"
	WorkStart int    // hour, for "work"
	WorkEnd   int
}

type Cell struct {
	From, To time.Time
	Out      bool // belongs to a neighbouring period (a month row's days outside the month)
}
type Row struct {
	Label int // Week: ISO weekday 0..6; Month: ISO week number; Year: month 1..12; All: year
	From, To time.Time
	Cells []Cell
}
type Grid struct {
	Period    Period
	From, To  time.Time // the whole period
	ColHours  []int     // Week only: the hour each column starts at
	Rows      []Row
}

// Layout builds the grid of `p` that contains `now` shifted by `offset` periods,
// in `loc`. firstYear is where «All» starts.
func Layout(p Period, now time.Time, offset int, loc *time.Location, sc Scale, firstYear int) Grid
```

Правила (спека §1):
- **Week**: ISO-неделя (пн). 7 строк, у каждой ячейки по часу. Часы колонок: `day24`/`peak` → 0…23;
  `work` → `WorkStart…WorkEnd-1`.
- **Month**: месяц; строки — ISO-недели, пересекающие месяц; ячейки — 7 дней недели; дни вне месяца
  `Out: true`.
- **Year**: год; 12 строк-месяцев; ячейки — куски месяца, **режущиеся на каждом понедельнике**
  (первый кусок — с 1-го числа до первого понедельника). 4–6 ячеек.
- **All**: строки — годы `firstYear…год(now)`; 12 ячеек-месяцев. `offset` игнорируется.
- Все границы — `time.Date(…, loc)` + `AddDate`: сутки в день перехода на зимнее время — 25 часов.

- [ ] **Step 1: Падающие тесты** — `layout_test.go`:

```go
package statgrid

import (
	"testing"
	"time"
)

var msk = mustLoc("Europe/Moscow")

func mustLoc(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

var day24 = Scale{Kind: "day24"}

// 22.09.2026 is a Tuesday.
var tue = time.Date(2026, 9, 22, 15, 0, 0, 0, msk)

func TestAWeekIsSevenDaysOfTwentyFourHours(t *testing.T) {
	g := Layout(Week, tue, 0, msk, day24, 2026)
	if len(g.Rows) != 7 || len(g.Rows[0].Cells) != 24 || len(g.ColHours) != 24 {
		t.Fatalf("rows=%d cells=%d", len(g.Rows), len(g.Rows[0].Cells))
	}
	if !g.Rows[0].From.Equal(time.Date(2026, 9, 21, 0, 0, 0, 0, msk)) {
		t.Errorf("week starts %v, want Monday 21.09", g.Rows[0].From)
	}
	if c := g.Rows[1].Cells[10]; !c.From.Equal(time.Date(2026, 9, 22, 10, 0, 0, 0, msk)) || c.To.Sub(c.From) != time.Hour {
		t.Errorf("tue 10:00 cell = %v–%v", c.From, c.To)
	}
}

func TestWorkScaleShowsOnlyTheWorkingHours(t *testing.T) {
	g := Layout(Week, tue, 0, msk, Scale{Kind: "work", WorkStart: 8, WorkEnd: 20}, 2026)
	if len(g.ColHours) != 12 || g.ColHours[0] != 8 || len(g.Rows[0].Cells) != 12 {
		t.Errorf("cols = %v", g.ColHours)
	}
}

func TestOffsetsWalkBothWays(t *testing.T) {
	back := Layout(Week, tue, -3, msk, day24, 2026)
	if !back.From.Equal(time.Date(2026, 8, 31, 0, 0, 0, 0, msk)) {
		t.Errorf("-3 weeks from = %v", back.From)
	}
	next := Layout(Month, tue, 1, msk, day24, 2026)
	if next.From.Month() != time.October {
		t.Errorf("+1 month = %v", next.From)
	}
}

// September 2026 starts on a Tuesday and ends on a Wednesday: five ISO weeks,
// the first with one day outside (Mon 31.08), the last with four.
func TestAMonthIsItsWeeksWithTheOutsideDaysMarked(t *testing.T) {
	g := Layout(Month, tue, 0, msk, day24, 2026)
	if len(g.Rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(g.Rows))
	}
	if !g.Rows[0].Cells[0].Out || g.Rows[0].Cells[1].Out {
		t.Errorf("first row: mon out=%v tue out=%v", g.Rows[0].Cells[0].Out, g.Rows[0].Cells[1].Out)
	}
	if !g.Rows[4].Cells[3].Out || g.Rows[4].Cells[2].Out {
		t.Errorf("last row: wed out=%v thu out=%v", g.Rows[4].Cells[2].Out, g.Rows[4].Cells[3].Out)
	}
}

// A year is twelve months; September 2026 cut at Mondays: 1–6, 7–13, 14–20,
// 21–27, 28–30 → five pieces.
func TestAYearIsMonthsCutAtMondays(t *testing.T) {
	g := Layout(Year, tue, 0, msk, day24, 2026)
	if len(g.Rows) != 12 {
		t.Fatalf("rows = %d", len(g.Rows))
	}
	sep := g.Rows[8]
	if len(sep.Cells) != 5 || sep.Cells[1].From.Day() != 7 || sep.Cells[4].To.Day() != 1 {
		t.Errorf("september cells = %d, second from %d", len(sep.Cells), sep.Cells[1].From.Day())
	}
}

func TestAllIsYearsTimesMonths(t *testing.T) {
	g := Layout(All, tue, 5, msk, day24, 2025)
	if len(g.Rows) != 2 || g.Rows[0].Label != 2025 || len(g.Rows[1].Cells) != 12 {
		t.Errorf("rows = %d, first = %d", len(g.Rows), g.Rows[0].Label)
	}
}

// 🔴 The day the clocks go back is 25 hours long in New York. A layout built
// with +24h would put the next midnight at 23:00.
func TestDayBoundariesSurviveDST(t *testing.T) {
	ny := mustLoc("America/New_York")
	g := Layout(Week, time.Date(2026, 11, 1, 12, 0, 0, 0, ny), 0, ny, day24, 2026)
	sun := g.Rows[6]
	if sun.To.Sub(sun.From) != 25*time.Hour || sun.To.In(ny).Hour() != 0 {
		t.Errorf("sunday 01.11 = %v → %v", sun.From, sun.To)
	}
}
```

- [ ] **Step 2:** прогнать → не компилируется.
- [ ] **Step 3: Реализация** — `layout.go`:

```go
package statgrid

import "time"

// Period is what one screen of statistics covers.
type Period string

const (
	Week  Period = "week"
	Month Period = "month"
	Year  Period = "year"
	All   Period = "all"
)

// Scale is what a full bar means (spec §3): 24 hours by default, the working
// window, or the fullest cell on the screen.
type Scale struct {
	Kind      string
	WorkStart int
	WorkEnd   int
}

// Cell is one bar.
type Cell struct {
	From, To time.Time
	Out      bool
}

// Row is one line of the screen.
type Row struct {
	Label    int
	From, To time.Time
	Cells    []Cell
}

// Grid is one screen.
type Grid struct {
	Period   Period
	From, To time.Time
	ColHours []int
	Rows     []Row
}

func midnight(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

func monday(day time.Time) time.Time {
	return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
}

// Layout builds the cells of one period in the user's zone.
func Layout(p Period, now time.Time, offset int, loc *time.Location, sc Scale, firstYear int) Grid {
	today := midnight(now, loc)
	switch p {
	case Month:
		return layoutMonth(time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, offset, 0))
	case Year:
		return layoutYear(today.Year()+offset, loc)
	case All:
		return layoutAll(firstYear, today.Year(), loc)
	default:
		return layoutWeek(monday(today).AddDate(0, 0, 7*offset), sc)
	}
}

func layoutWeek(start time.Time, sc Scale) Grid {
	hours := make([]int, 0, 24)
	first, last := 0, 24
	if sc.Kind == "work" && sc.WorkEnd > sc.WorkStart {
		first, last = sc.WorkStart, sc.WorkEnd
	}
	for h := first; h < last; h++ {
		hours = append(hours, h)
	}
	g := Grid{Period: Week, From: start, To: start.AddDate(0, 0, 7), ColHours: hours}
	for d := 0; d < 7; d++ {
		day := start.AddDate(0, 0, d)
		row := Row{Label: d, From: day, To: day.AddDate(0, 0, 1)}
		for _, h := range hours {
			// Built from the date, not by adding hours: on a DST day the wall
			// clock and elapsed time disagree, and the wall clock is what the
			// column header says.
			from := time.Date(day.Year(), day.Month(), day.Day(), h, 0, 0, 0, day.Location())
			row.Cells = append(row.Cells, Cell{From: from, To: from.Add(time.Hour)})
		}
		g.Rows = append(g.Rows, row)
	}
	return g
}

func layoutMonth(first time.Time) Grid {
	next := first.AddDate(0, 1, 0)
	g := Grid{Period: Month, From: first, To: next}
	for wk := monday(first); wk.Before(next); wk = wk.AddDate(0, 0, 7) {
		_, isoWeek := wk.ISOWeek()
		row := Row{Label: isoWeek, From: wk, To: wk.AddDate(0, 0, 7)}
		for d := 0; d < 7; d++ {
			day := wk.AddDate(0, 0, d)
			row.Cells = append(row.Cells, Cell{From: day, To: day.AddDate(0, 0, 1),
				Out: day.Before(first) || !day.Before(next)})
		}
		g.Rows = append(g.Rows, row)
	}
	return g
}

func layoutYear(year int, loc *time.Location) Grid {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	g := Grid{Period: Year, From: start, To: start.AddDate(1, 0, 0)}
	for m := 0; m < 12; m++ {
		first := start.AddDate(0, m, 0)
		next := first.AddDate(0, 1, 0)
		row := Row{Label: m + 1, From: first, To: next}
		cellFrom := first
		for day := first.AddDate(0, 0, 1); day.Before(next); day = day.AddDate(0, 0, 1) {
			if day.Weekday() == time.Monday {
				row.Cells = append(row.Cells, Cell{From: cellFrom, To: day})
				cellFrom = day
			}
		}
		row.Cells = append(row.Cells, Cell{From: cellFrom, To: next})
		g.Rows = append(g.Rows, row)
	}
	return g
}

func layoutAll(firstYear, lastYear int, loc *time.Location) Grid {
	if firstYear > lastYear {
		firstYear = lastYear
	}
	g := Grid{Period: All,
		From: time.Date(firstYear, 1, 1, 0, 0, 0, 0, loc),
		To:   time.Date(lastYear+1, 1, 1, 0, 0, 0, 0, loc)}
	for y := firstYear; y <= lastYear; y++ {
		start := time.Date(y, 1, 1, 0, 0, 0, 0, loc)
		row := Row{Label: y, From: start, To: start.AddDate(1, 0, 0)}
		for m := 0; m < 12; m++ {
			from := start.AddDate(0, m, 0)
			row.Cells = append(row.Cells, Cell{From: from, To: from.AddDate(0, 1, 0)})
		}
		g.Rows = append(g.Rows, row)
	}
	return g
}
```

⚠ Тест «September cut at Mondays»: 1.09.2026 — вторник; понедельники 7, 14, 21, 28 → куски
1–6, 7–13, 14–20, 21–27, 28–30 = 5; `Cells[4].To` = 1 октября → `.Day() == 1`.

- [ ] **Step 4:** PASS. **Саботаж — два:** (1) в `layoutWeek` `row.To = day.Add(24*time.Hour)` → DST-тест `FAIL`; (2) в `layoutMonth` убрать `Out:` → тест месяца `FAIL`. Откатить.
- [ ] **Step 5: Commit** — `feat(bot): statgrid — the cells of a week, month, year and all time, in the user's zone`

---

### Task 5 (S2): уровни и ёмкость

**Files:**
- Create: `bot/internal/statgrid/levels.go`, `bot/internal/statgrid/levels_test.go`

**Interfaces — Produces:**

```go
// Capacity is how much time a cell holds at full height, by scale.
func Capacity(c Cell, sc Scale) time.Duration
// Level maps value/capacity to 0..8 (0 = nothing; any time > 0 is at least 1).
func Level(value, capacity time.Duration) int
// Levels fills a grid: value(cell) per cell, scaled by sc ("peak" → the largest value = full).
func Levels(g Grid, sc Scale, value func(Cell) time.Duration) [][]int
// Glyph: 0 → "·", 1..8 → "▁▂▃▄▅▆▇█".
func Glyph(level int) string
```

Ёмкость: ячейка длиной до часа включительно — её длительность; иначе `day24` → длительность ячейки
(сутки с учётом DST), `work` → `(WorkEnd−WorkStart)` часов × число суток в ячейке.

- [ ] **Step 1: Падающие тесты** — `levels_test.go`:

```go
package statgrid

import (
	"testing"
	"time"
)

func TestLevelsAreEightStepsAndNothingIsADot(t *testing.T) {
	for _, c := range []struct {
		v, cap time.Duration
		want   int
	}{
		{0, time.Hour, 0},
		{time.Minute, time.Hour, 1}, // anything booked is visible
		{30 * time.Minute, time.Hour, 4},
		{time.Hour, time.Hour, 8},
		{3 * time.Hour, time.Hour, 8}, // over 100% → full
	} {
		if got := Level(c.v, c.cap); got != c.want {
			t.Errorf("Level(%v/%v) = %d, want %d", c.v, c.cap, got, c.want)
		}
	}
	if Glyph(0) != "·" || Glyph(8) != "█" || Glyph(1) != "▁" {
		t.Errorf("glyphs %q %q %q", Glyph(0), Glyph(1), Glyph(8))
	}
}

// Spec §7: 12 h booked in a day is half a bar at 24 h and a full one at 08–20.
func TestTheScaleDecidesTheHeight(t *testing.T) {
	g := Layout(Month, tue, 0, msk, day24, 2026)
	twelve := func(c Cell) time.Duration {
		if c.From.Day() == 22 && c.From.Month() == time.September {
			return 12 * time.Hour
		}
		return 0
	}
	find := func(lv [][]int) int {
		for r, row := range g.Rows {
			for i, c := range row.Cells {
				if c.From.Day() == 22 && c.From.Month() == time.September {
					return lv[r][i]
				}
			}
		}
		return -1
	}
	if l := find(Levels(g, day24, twelve)); l != 4 {
		t.Errorf("24h scale: level %d, want 4", l)
	}
	if l := find(Levels(g, Scale{Kind: "work", WorkStart: 8, WorkEnd: 20}, twelve)); l != 8 {
		t.Errorf("work scale: level %d, want 8", l)
	}
	if l := find(Levels(g, Scale{Kind: "peak"}, twelve)); l != 8 {
		t.Errorf("peak scale: the fullest cell is full, got %d", l)
	}
}
```

- [ ] **Step 2:** прогнать → не компилируется.
- [ ] **Step 3: Реализация** — `levels.go`:

```go
package statgrid

import "time"

var glyphs = []string{"·", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// Glyph is the character for a level. Denis 20.09 on the old «▪»: «странный
// выбор черных квадратов» — height by time replaced it on 21.09.
func Glyph(level int) string {
	if level < 0 {
		level = 0
	}
	if level > 8 {
		level = 8
	}
	return glyphs[level]
}

// Level maps a share of capacity to 0..8. Anything booked is at least 1: a
// ten-minute call must not vanish from the screen.
func Level(value, capacity time.Duration) int {
	if value <= 0 || capacity <= 0 {
		return 0
	}
	l := int((value*8 + capacity - 1) / capacity) // ceil
	if l < 1 {
		l = 1
	}
	if l > 8 {
		l = 8
	}
	return l
}

// Capacity is a cell's time at full height under the scale.
func Capacity(c Cell, sc Scale) time.Duration {
	length := c.To.Sub(c.From)
	if length <= time.Hour || sc.Kind != "work" || sc.WorkEnd <= sc.WorkStart {
		return length
	}
	days := 0
	for d := c.From; d.Before(c.To); d = d.AddDate(0, 0, 1) {
		days++
	}
	return time.Duration(days*(sc.WorkEnd-sc.WorkStart)) * time.Hour
}

// Levels fills the grid. Cells outside the period stay 0.
func Levels(g Grid, sc Scale, value func(Cell) time.Duration) [][]int {
	vals := make([][]time.Duration, len(g.Rows))
	var peak time.Duration
	for r, row := range g.Rows {
		vals[r] = make([]time.Duration, len(row.Cells))
		for i, c := range row.Cells {
			if c.Out {
				continue
			}
			v := value(c)
			vals[r][i] = v
			if v > peak {
				peak = v
			}
		}
	}
	out := make([][]int, len(g.Rows))
	for r, row := range g.Rows {
		out[r] = make([]int, len(row.Cells))
		for i, c := range row.Cells {
			capacity := Capacity(c, sc)
			if sc.Kind == "peak" {
				capacity = peak
			}
			out[r][i] = Level(vals[r][i], capacity)
		}
	}
	return out
}
```

- [ ] **Step 4:** PASS. **Саботаж:** в `Levels` игнорировать `sc` (`capacity := c.To.Sub(c.From)`) → «work scale» `FAIL`; откатить.
- [ ] **Step 5: Commit** — `feat(bot): statgrid — levels by scale, 24 hours by default`

---

### Task 6 (S2): что меряется — события, задачи, рефлексии

**Files:**
- Create: `bot/internal/statgrid/measure.go`, `bot/internal/statgrid/measure_test.go`

**Interfaces — Consumes:** `api.Event`, `api.Task` (с `ActualMinutes`, `EstimatedMinutes`,
`CompletedAt`, `Rrule`, `RepeatAnchor`), `api.TaskOccurrence`, `api.Reflection` (Task 2).
**Produces:**

```go
// EventSpans: timed events as spans; all-day ones are counted, not spanned (spec §1).
func EventSpans(events []api.Event) (spans []Span, allDay int)
// TaskMinutes: actual, else estimate, else 30 (spec §2).
func TaskMinutes(t api.Task) int
type Point struct{ At time.Time; Dur time.Duration }
// TaskPoints: one point per closed one-off task (at completion) and per «done» day of a
// series (at that day's local midnight), each worth TaskMinutes.
func TaskPoints(tasks []api.Task, occ []api.TaskOccurrence, loc *time.Location) []Point
// ReflectionDays: the set of local days (YYYY-MM-DD) with a reflection.
func ReflectionDays(refl []api.Reflection, loc *time.Location) map[string]bool
// SumPoints: total Dur of points inside [c.From, c.To).
func SumPoints(points []Point, c Cell) time.Duration
```

- [ ] **Step 1: Падающие тесты** — `measure_test.go`:

```go
package statgrid

import (
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

func sp(s string) *string { return &s }

func TestAllDayEventsAreCountedNotSpanned(t *testing.T) {
	spans, allDay := EventSpans([]api.Event{
		{StartsAt: "2026-09-21T07:00:00Z", EndsAt: "2026-09-21T08:00:00Z"},
		{StartsAt: "2026-09-21T00:00:00Z", EndsAt: "2026-09-22T00:00:00Z", AllDay: true},
	})
	if len(spans) != 1 || allDay != 1 {
		t.Errorf("spans=%d allDay=%d", len(spans), allDay)
	}
}

// Spec §2 / Denis 22.09: estimate of what was closed; logged time wins; 30 min
// when there is neither.
func TestATaskIsWorthItsTime(t *testing.T) {
	if m := TaskMinutes(api.Task{EstimatedMinutes: 90}); m != 90 {
		t.Errorf("estimate: %d", m)
	}
	if m := TaskMinutes(api.Task{EstimatedMinutes: 90, ActualMinutes: 50}); m != 50 {
		t.Errorf("logged wins: %d", m)
	}
	if m := TaskMinutes(api.Task{}); m != 30 {
		t.Errorf("default: %d", m)
	}
}

// 🔴 A task closed at 00:30 Moscow belongs to that Moscow day, not to the
// previous UTC one — and a series day counts on its own date.
func TestTaskPointsLandOnTheUsersDay(t *testing.T) {
	pts := TaskPoints([]api.Task{
		{ID: "a", Status: "DONE", CompletedAt: "2026-09-21T21:30:00Z", EstimatedMinutes: 60},
		{ID: "s", Status: "TODO", Rrule: "FREQ=DAILY"},
		{ID: "open", Status: "TODO"},
	}, []api.TaskOccurrence{
		{TaskID: "s", Occurrence: "2026-09-22", State: "done"},
		{TaskID: "s", Occurrence: "2026-09-23", State: "skipped"},
	}, msk)
	day22 := Cell{From: time.Date(2026, 9, 22, 0, 0, 0, 0, msk), To: time.Date(2026, 9, 23, 0, 0, 0, 0, msk)}
	if got := SumPoints(pts, day22); got != 90*time.Minute {
		t.Errorf("22.09 = %v, want 1h30 (60 closed at 00:30 MSK + a 30-min series day)", got)
	}
	day23 := Cell{From: day22.To, To: day22.To.AddDate(0, 0, 1)}
	if got := SumPoints(pts, day23); got != 0 {
		t.Errorf("a skipped day counted: %v", got)
	}
}

func TestReflectionDaysAreLocal(t *testing.T) {
	days := ReflectionDays([]api.Reflection{{CreatedAt: "2026-09-21T22:10:00Z"}}, msk)
	if !days["2026-09-22"] || days["2026-09-21"] {
		t.Errorf("days = %v, want 22.09 in Moscow", days)
	}
}
```

- [ ] **Step 2:** прогнать → не компилируется.
- [ ] **Step 3: Реализация** — `measure.go`:

```go
package statgrid

import (
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

// EventSpans turns timed events into spans. All-day events are not «busy
// hours» — a birthday is not twenty-four booked hours — so they are counted
// apart (spec §1).
func EventSpans(events []api.Event) (spans []Span, allDay int) {
	for _, e := range events {
		if e.AllDay {
			allDay++
			continue
		}
		s, err1 := time.Parse(time.RFC3339, e.StartsAt)
		en, err2 := time.Parse(time.RFC3339, e.EndsAt)
		if err1 != nil || err2 != nil || !en.After(s) {
			continue
		}
		spans = append(spans, Span{Start: s, End: en})
	}
	return spans, allDay
}

// defaultTaskMinutes is what a closed task without an estimate is worth.
const defaultTaskMinutes = 30

// TaskMinutes is a task's time: logged, else estimated, else 30 minutes.
func TaskMinutes(t api.Task) int {
	switch {
	case t.ActualMinutes > 0:
		return t.ActualMinutes
	case t.EstimatedMinutes > 0:
		return t.EstimatedMinutes
	default:
		return defaultTaskMinutes
	}
}

// Point is time that belongs to one moment.
type Point struct {
	At  time.Time
	Dur time.Duration
}

// TaskPoints places done work in time: a closed one-off task at the moment it
// was closed, each «done» day of a series at that day's local midnight.
func TaskPoints(tasks []api.Task, occ []api.TaskOccurrence, loc *time.Location) []Point {
	byID := map[string]api.Task{}
	var pts []Point
	for _, t := range tasks {
		byID[t.ID] = t
		if t.Rrule != "" || t.Status != "DONE" || t.CompletedAt == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339, t.CompletedAt)
		if err != nil {
			continue
		}
		pts = append(pts, Point{At: at, Dur: time.Duration(TaskMinutes(t)) * time.Minute})
	}
	for _, o := range occ {
		if o.State != "done" {
			continue
		}
		day, err := time.ParseInLocation("2006-01-02", o.Occurrence, loc)
		if err != nil {
			continue
		}
		pts = append(pts, Point{At: day, Dur: time.Duration(TaskMinutes(byID[o.TaskID])) * time.Minute})
	}
	return pts
}

// ReflectionDays is the set of the user's days with a reflection.
func ReflectionDays(refl []api.Reflection, loc *time.Location) map[string]bool {
	days := map[string]bool{}
	for _, r := range refl {
		at, err := time.Parse(time.RFC3339, r.CreatedAt)
		if err != nil {
			continue
		}
		days[at.In(loc).Format("2006-01-02")] = true
	}
	return days
}

// SumPoints is the time of the points inside the cell.
func SumPoints(points []Point, c Cell) time.Duration {
	var total time.Duration
	for _, p := range points {
		if !p.At.Before(c.From) && p.At.Before(c.To) {
			total += p.Dur
		}
	}
	return total
}
```

⚠ Серия с неизвестной задачей (`byID` пусто — задача удалена или чужая) даёт `TaskMinutes(api.Task{})`
= 30 мин: день был сделан, время неизвестно. Так и задумано.

- [ ] **Step 4:** PASS. **Саботаж — два:** (1) в `ReflectionDays` `at.Format` без `.In(loc)` → `FAIL`; (2) в `TaskPoints` не пропускать `skipped` → `FAIL`. Откатить.
- [ ] **Step 5: Проверка модуля:** `cd bot && go build ./... && go vet ./... && go test -count=1 ./... 2>&1 | grep -E '^(--- FAIL|FAIL)'` → пусто; `gofmt -l internal/statgrid` → пусто.
- [ ] **Step 6: Commit** — `feat(bot): statgrid — what events, tasks and reflections are worth on the grid`

---

## Покрытие спеки → задачи

| Спека | Задача |
|---|---|
| §5 ручка дней серий | 1 |
| чтение рефлексий, `actual_minutes`, `repeat_anchor` | 2 |
| §1 «Занято» = объединение, «В планах» = сумма | 3 |
| §1 строки/столбики неделя · месяц · год · всё, листание ← → | 4 |
| §1 символы, >100 % → █ | 5 |
| §3 шкала day24 / work / peak | 4 (колонки недели), 5 (ёмкость) |
| §2 задачи — время закрытого, 30 мин по умолчанию, дни серий | 6 |
| §2 рефлексии по дням | 6 |
| «Серии: N из M» — M по правилу и якорю | ⚠ **не здесь**: S3 (нужен разбор `rrule` в боте — `parse`/`keyboards.RepeatCodes` уже знают правила; решить в плане S3) |
| §1 экран, кнопки, `<pre>`, итоги · §3 кнопка/настройка/онбординг · §4 календарь | S3–S5, отдельные планы |
