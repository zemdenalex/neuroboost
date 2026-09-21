<!-- паспорт: тип=план | статус=действует | строк=334 | ~токенов=3333 | обновлён=по git -->

# Статистика S3 — экран · план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** «📊 Статистика» в боте — экран по спеке §1–§3: период × сущность, листание ← →, сетка
`·▁▂▃▄▅▆▇█` в `<pre>`, итоги, кнопка «📏 Шкала», которая работает.

**Architecture:** весь расчёт — в `statgrid` (S2) плюс `SeriesDays` (Task 1). Экран — чистая
функция `renderStatsScreen` (текст) и тонкий обработчик, который собирает данные из API и зовёт её.
Состояние экрана живёт **в кнопке**: `st_<период>_<сущность>_<сдвиг>` — flow не нужен, старые
сообщения остаются рабочими. Прежний экран (`BuildStats`, `renderStats`, их тесты) удаляется —
он заменён целиком; `weekdayShort` остаётся (его зовут календарь и «Сегодня»).

**Tech Stack:** Go, модуль `bot/`.

**Spec:** `docs/superpowers/specs/2026-09-22-statistics-design.md`. **Предыдущий план:**
`docs/superpowers/plans/2026-09-22-statistics-s1-s2.md` (API, `statgrid`).

## Global Constraints

- Весь видимый текст — `h.t` / `i18n.T` (скан `TestNoUntranslatedUserFacingText`); ошибки API —
  `h.errorText`.
- Новые префиксы кнопок — в роутере **и** в `routed` теста `TestEveryScheduleButtonHasAPrefixTheRouterKnows`;
  `callback_data` ≤ 64 байт.
- Всё в зоне пользователя (`h.location(chatID)`).
- Запись `settings.bot.stats_scale` — ключом внутри секции `bot`, как `SetBotLang` (MergeSettings
  мелкий: патч `{"bot": …}` целиком стёр бы язык и словарь).
- Ширина строки в `<pre>` ≤ 36 знаков.
- Ноль `FAIL`; саботаж каждого нового теста; не пушить до слова Дениса.

## Решения плана

1. 🟡 **Неделя × Задачи:** дни серий не рисуются столбиками (у дня серии нет часа — все встали бы в
   колонку 00), они — в строке «Серии: N из M». Месяц/год/всё — рисуются. Для этого у `Point`
   появляется поле `DayOnly`.
2. 🟡 **Неделя × Рефлексии** — строка `пн ✓ вт · …` вместо сетки часов (у рефлексии нет длительности,
   спека §2 так и говорит).
3. **«Всё»** начинается с самого раннего года, где есть события, — бот идёт назад по годам, пока год
   не пуст, не дальше 5 лет.
4. **«N из M» у серий** — только по дням **до сегодня включительно**: будущие дни ещё не могли быть
   сделаны.

---

### Task 1: сколько дней серии было положено

**Files:** Create `bot/internal/statgrid/series.go`, `series_test.go`.

**Produces:** `func SeriesDays(rrule, anchor string, from, to time.Time, loc *time.Location) int` —
вхождения в `[from, to)`. Поддержано: `FREQ=DAILY|WEEKLY|MONTHLY`, `INTERVAL`, `COUNT`, `UNTIL`
(ровно то, что пишет бот, `keyboards.RepeatCodes`). Непонятное правило → 0.

```go
// series_test.go
package statgrid

import (
	"testing"
	"time"
)

func TestSeriesDaysFollowTheRule(t *testing.T) {
	from := time.Date(2026, 9, 21, 0, 0, 0, 0, msk)
	to := from.AddDate(0, 0, 7)
	anchor := "2026-09-01T00:00:00+03:00"
	for _, c := range []struct {
		rule string
		want int
	}{
		{"FREQ=DAILY", 7},
		{"FREQ=DAILY;INTERVAL=2", 4},   // anchored on the 1st: 21, 23, 25, 27
		{"FREQ=WEEKLY", 1},             // Tuesdays: 22.09
		{"FREQ=MONTHLY", 0},            // the 1st
		{"FREQ=DAILY;COUNT=25", 5},     // 1..25 → 21..25
		{"FREQ=DAILY;UNTIL=2026-09-23", 3},
		{"FREQ=YEARLY", 0},             // not written by the bot: unknown → 0
	} {
		if got := SeriesDays(c.rule, anchor, from, to, msk); got != c.want {
			t.Errorf("%s: %d, want %d", c.rule, got, c.want)
		}
	}
}

func TestASeriesThatStartsLaterHasNoEarlierDays(t *testing.T) {
	from := time.Date(2026, 9, 21, 0, 0, 0, 0, msk)
	if got := SeriesDays("FREQ=DAILY", "2026-09-25T00:00:00+03:00", from, from.AddDate(0, 0, 7), msk); got != 3 {
		t.Errorf("got %d, want 3 (25, 26, 27)", got)
	}
}
```

```go
// series.go
package statgrid

import (
	"strconv"
	"strings"
	"time"
)

// SeriesDays counts the days a repeating task was due in [from, to) — the M of
// «Серии: N из M». The rule grammar is the bot's own (keyboards.RepeatCodes):
// DAILY/WEEKLY/MONTHLY with INTERVAL, COUNT and UNTIL. Anything else → 0: a
// guess would print a number nobody can check.
func SeriesDays(rrule, anchor string, from, to time.Time, loc *time.Location) int {
	a, err := time.Parse(time.RFC3339, anchor)
	if err != nil {
		return 0
	}
	start := midnight(a, loc)
	freq, interval, count := "", 1, 0
	var until time.Time
	for _, part := range strings.Split(rrule, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "FREQ":
			freq = kv[1]
		case "INTERVAL":
			if n, e := strconv.Atoi(kv[1]); e == nil && n > 0 {
				interval = n
			}
		case "COUNT":
			if n, e := strconv.Atoi(kv[1]); e == nil && n > 0 {
				count = n
			}
		case "UNTIL":
			if d, e := time.ParseInLocation("2006-01-02", kv[1][:min(10, len(kv[1]))], loc); e == nil {
				until = d.AddDate(0, 0, 1) // inclusive day
			}
		}
	}
	step := func(i int) time.Time {
		switch freq {
		case "DAILY":
			return start.AddDate(0, 0, i*interval)
		case "WEEKLY":
			return start.AddDate(0, 0, 7*i*interval)
		case "MONTHLY":
			return start.AddDate(0, i*interval, 0)
		}
		return time.Time{}
	}
	if step(0).IsZero() {
		return 0
	}
	n := 0
	for i := 0; ; i++ {
		d := step(i)
		if !d.Before(to) || (count > 0 && i >= count) || (!until.IsZero() && !d.Before(until)) {
			return n
		}
		if !d.Before(from) {
			n++
		}
	}
}
```

⚠ `min` для int есть в Go ≥ 1.21 как встроенная; модуль бота — проверить `go.mod`; если ниже — своя.

- [ ] Тест → не компилируется → реализация → PASS. **Саботаж:** `INTERVAL` игнорировать → `FAIL`. Commit `feat(bot): statgrid — how many days a series was due`.

### Task 2: день серии без часа

**Files:** Modify `bot/internal/statgrid/measure.go`, `measure_test.go`.

`Point` получает `DayOnly bool`; `TaskPoints` ставит его у дней серий. Новая
`func SumPointsTimed(points []Point, c Cell) time.Duration` — то же, что `SumPoints`, но без `DayOnly`.

```go
func TestSeriesDaysAreMarkedAsHavingNoHour(t *testing.T) {
	pts := TaskPoints(nil, []api.TaskOccurrence{{TaskID: "s", Occurrence: "2026-09-22", State: "done"}}, msk)
	if len(pts) != 1 || !pts[0].DayOnly {
		t.Fatalf("points = %+v", pts)
	}
	c := Cell{From: time.Date(2026, 9, 22, 0, 0, 0, 0, msk), To: time.Date(2026, 9, 22, 1, 0, 0, 0, msk)}
	if SumPointsTimed(pts, c) != 0 || SumPoints(pts, c) == 0 {
		t.Error("a series day must count by day, never in the 00:00 hour")
	}
}
```

- [ ] Тест → FAIL → реализация → PASS. **Саботаж:** не ставить `DayOnly` → `FAIL`. Commit.

### Task 3: настройка бота одним ключом

**Files:** Modify `bot/internal/api/lang.go` (или новый `bot/internal/api/botsettings.go`), тест рядом.

**Produces:** `func (c *Client) BotSetting(token, key string) (string, error)` и
`func (c *Client) SetBotSetting(token, key, value string) error` — чтение/запись одного ключа секции
`bot`, остальные ключи секции и blob целы.

```go
func TestSetBotSettingKeepsTheRestOfTheBotSection(t *testing.T) {
	var written string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPatch {
			b, _ := io.ReadAll(r.Body)
			written = string(b)
			_, _ = w.Write([]byte(`{"data":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"settings":{"work_start":"09:00","bot":{"lang":"ru","keywords":{"x":1}}}}}`))
	}))
	defer srv.Close()
	if err := NewClient(srv.URL).SetBotSetting("tok", "stats_scale", "work"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"stats_scale":"work"`, `"lang":"ru"`, `"keywords"`, `"work_start":"09:00"`} {
		if !strings.Contains(written, want) {
			t.Errorf("PATCH %s lost %s", written, want)
		}
	}
}
```

Реализация — копия `SetBotLang` с ключом параметром; `BotLang`/`SetBotLang` не трогать.

- [ ] Тест → FAIL → реализация → PASS. **Саботаж:** писать `{"bot": {key: value}}` без копии секции → `FAIL`. Commit.

### Task 4: экран — вид, кнопки, текст

**Files:**
- Create: `bot/internal/handlers/statsview.go`, `statsview_test.go`
- Create: `bot/internal/keyboards/stats.go`
- Modify: `bot/internal/handlers/stats.go` (удалить `StatsWeek`, `BuildStats`, `handleStats`, `renderStats`, `fieldLine` если больше не нужен, `min`; **оставить** `weekdayShort`)
- Delete: `bot/internal/handlers/stats_test.go` (тестирует удалённое) — перенести в `statsview_test.go` только проверку, что пустая неделя говорит «Пока пусто»
- Modify: `handler.go` (роут), `schedule_test.go` (`routed`)

**Interfaces:**

```go
type statsView struct {
	Period statgrid.Period // week|month|year|all
	Entity string          // "a" all · "e" events · "t" tasks · "r" reflections
	Offset int             // periods from the current one; ignored for all
}
func parseStatsView(s string) (statsView, bool) // "w_a_-3"; offset clamped to ±120
func (v statsView) code() string                // "w_a_-3"
// callbacks: "st_" + code (open/paginate) · "stsc_" + code (next scale, then redraw)

type statsData struct {
	Events []api.Event
	Tasks  []api.Task
	Occ    []api.TaskOccurrence
	Refl   []api.Reflection
}
func renderStatsScreen(lang i18n.Lang, v statsView, g statgrid.Grid, d statsData,
	sc statgrid.Scale, loc *time.Location, now time.Time) string

// keyboards
func StatsNav(lang i18n.Lang, period, entity string, offset int, scaleLabel string) tgbotapi.InlineKeyboardMarkup
```

**Экран** (спека §1): заголовок; `<pre>` с шапкой колонок и строками `метка сетка итог`; итоги
по сущности; для `r` × неделя — строка `пн ✓ вт · …`.

Заголовки: неделя — `Неделя 21–27 сентября`; месяц — `Сентябрь 2026`; год — `2026`; всё — `Всё время`.
Метки строк: неделя — `weekdayShort`; месяц — номер ISO-недели; год — `monthShort` (новая, через
`i18n.T` по месяцу); всё — год. Итог строки: `e`/`a` — занятые часы `5,5 ч`, `t` — время задач,
`r` — `N дн`; ноль — `—`.

Итоги:
- `a`/`e`: `Занято: X ч · В планах: Y ч`, `Весь день: N` (если >0).
- `a`/`t`: `✅ Закрыто: N · 📋 Открыто: M · 🔴 Просрочено: K`, `🔁 Серии: сделано N из M` (M —
  `SeriesDays` по дням периода до сегодня включительно; строки нет, если M = 0).
- `a`/`r`: `📝 Рефлексии: N из M дней` (M — дни периода до сегодня включительно).
- Пусто совсем (ни событий, ни задач, ни рефлексий в периоде) — `Пока пусто…`, кнопки остаются.

Кнопки:
```
[←] [✓ Неделя] [Месяц] [Год] [Всё] [→]      (у «Всё» стрелок нет)
[✓ Всё] [События] [Задачи] [Рефлексии]
[📏 Шкала: 24 ч] [« Меню]
```
Переход между периодами сбрасывает сдвиг в 0; смена сущности сдвиг сохраняет.

**Тесты** (`statsview_test.go`) — чистые, без Telegram:

```go
func TestStatsViewRoundTripsThroughAButton(t *testing.T)            // parse(code()) == v; bad input refused; offset clamped
func TestTheWeekShowsBusyHoursOnceAndPlannedTwice(t *testing.T)     // 10–11 + 10:30–11:30 → «Занято: 1,5 ч · В планах: 2 ч»
func TestTheWeekGridIsTheDaysAndHours(t *testing.T)                 // 7 строк в <pre>, у дня с 12:00–13:00 — не «·» в колонке 12
func TestTasksCountTheirTimeAndSeriesDays(t *testing.T)             // закрытая 90 мин → «1,5 ч»; серия DAILY с 3 «done» из 7 → «3 из 7»
func TestReflectionsInAWeekAreTicks(t *testing.T)                   // «пн ✓»
func TestAnEmptyPeriodSaysSoAndKeepsTheButtons(t *testing.T)
func TestEveryStatsButtonFitsAndIsRouted(t *testing.T)              // ≤64 байт; префиксы st_/stsc_ в router-скане (schedule_test.go)
```

(Код тестов пишется в этой задаче по этим строкам — утверждения перечислены, числа точные.)

- [ ] Тесты → FAIL → реализация → PASS → саботаж: (1) итог «Занято» считать `Planned` → FAIL; (2) у
  `r` × неделя рисовать сетку → FAIL. Commit `feat(bot): statistics is a screen you browse`.

### Task 5: обработчик и данные

**Files:** Modify `statsview.go`, `handler.go`; test `statsview_handler_test.go` (фейковый API).

- `handleStatsView(chatID, messageID int, v statsView)`: шкала из `BotSetting("stats_scale")`
  (пусто → `day24`), рабочие часы — `h.workHours`; сетка `statgrid.Layout`; события — `GetEvents`
  на `[g.From, g.To)` (для «Всё» — по году, и поиск первого года: назад, пока год не пуст, ≤ 5 лет);
  задачи — `GetTasks("")`; дни серий — `TaskOccurrences` (≤ 366 дней за вызов — по годам); рефлексии
  — `Reflections`. Ошибка событий — экран ошибки через `h.errorText`; ошибки остальных — пустые
  списки (итог без них лучше, чем никакого).
- `stsc_` — следующая шкала по кругу `day24 → work → peak → day24`, `SetBotSetting`, перерисовка.
- `stats` (кнопка меню) → `handleStatsView(…, statsView{Week, "a", 0})`.

Тест: фейковый API отдаёт событие 10:00–11:00 сегодня и записывает PATCH; `handleStatsView` →
текст содержит «Занято: 1 ч»; нажатие `stsc_w_a_0` → в PATCH `"stats_scale":"work"`, в новом
тексте кнопка «📏 Шкала: рабочие часы».

- [ ] Тест → FAIL → реализация → PASS. **Саботаж:** не писать шкалу → FAIL. Весь модуль — ноль `FAIL`.
  Commit `feat(bot): statistics reads its data and remembers the scale`.

---

## Покрытие

| Спека | Задача |
|---|---|
| §1 экран, `<pre>`, строки/столбики, итоги строк | 4 |
| §1 ← → в обе стороны, у «Всё» нет | 4 (кнопки), 5 (сдвиг в данных) |
| §1 Занято/В планах | 4 (через `statgrid.Busy/Planned`) |
| §2 сущности | 4 |
| §2 серии «N из M» | 1, 4 |
| §3 шкала + кнопка на экране | 3, 5 |
| §3 шкала в ⚙️ и онбординге | **S4**, отдельный план |
| §4 месячный календарь | **S5** |
| §6 «Всё» по годам, первый год | 5 |
