# «Задачи дня» D3 — цвет везде и переключатели · Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** цвет дня в клетке месячного календаря и на 📅 Сегодня; задачи дня можно выключить (в настройках и в онбординге); дни до первого взятого по умолчанию не красятся.

**Architecture:** сервер один раз решает «до начала» (`before_start` в каждом дне `GET /api/day-tasks`), бот только рисует. Две новые настройки верхнего уровня (`day_tasks_enabled`, `day_tasks_paint_before`) пишутся через чтение-слияние-запись. Включённость кэшируется в `UserState`, как зона и символ приоритета, чтобы меню не ходило в API на каждый показ.

**Tech Stack:** Go — `api-go` (chi, pgx/v5), `bot` (go-telegram-bot-api v5).

**Spec:** `docs/superpowers/specs/2026-09-22-day-tasks-design.md` §11 (одобрено Денисом 24.09), §3 (уровни), §6 (клетка).

## Global Constraints

- Ключи настроек **верхнего уровня**: `day_tasks_enabled` (нет ключа = включено), `day_tasks_paint_before` (нет ключа = не красить).
- Запись настроек — только `api.PatchSettings` (читает, сливает, пишет; при ошибке чтения не пишет — gotcha 21).
- Уровень и «до начала» считает **сервер**; бот рисует `format.DayLevel(level)` (⬛🟫🟥🟧🟨🟩).
- Клетка: цвет, пробел, дата, пробел, занятость → `🟩 21 ▅` (§6); дни чужого месяца (`·21`) без цвета.
- Будущий день — без цвета; сегодня — цвет только если день взят.
- Выключено = нет 📌 в меню и на карточке, нет цвета в клетке и строки на Сегодня; данные не трогаются.
- В пользовательских строках — **никаких длинных тире** (`dash_scan_test.go` краснеет сам).
- Каждая новая клавиатура — с `HelpButton` или в исключениях `help_scan_test.go` (потолок `maxWithoutHelp = 65` не поднимать).
- Каждый новый префикс callback'а — в списке `routed` в `handlers/schedule_test.go`.
- Оба модуля: `go build ./... && go vet ./... && go test ./...`; api-go тесты с `DATABASE_URL` (без него DB-тесты скипаются — это не зелёный).
- Каждая починка — сабботаж: компилируется, применяется ровно один раз, валит **свой** тест.
- Коммит на задачу, `git add` конкретных файлов; push — только по слову Дениса.

## Review Focus

1. **Чтение настроек упало** → задачи дня считаются включёнными (ничего не прячем из-за сети) и ничего не пишется.
2. **Старая кнопка `dt_…` из чата при выключенных** → фраза «📌 Задачи дня выключены» и кнопка включить, а не экран и не запись.
3. **Человек ответил в онбординге** (ключ `day_tasks_enabled` уже есть) → одноразовый вопрос старым пользователям ему не показывается.
4. **Сегодня ещё не взят** → в клетке сегодня нет цвета (не ⬛), на Сегодня строка «📌 День ещё не взят».
5. **Переключение не стирает другие настройки** (gotcha 21): после записи `lang`, `stats_scale`, `day_tasks_target` на месте.

---

## Files

| Файл | Что |
|---|---|
| `api-go/internal/daytasks/store.go` | `Day.BeforeStart`, расчёт в `List` |
| `api-go/internal/daytasks/db_test.go` | тесты `before_start` через HTTP |
| `bot/internal/api/daytasks.go` | `Day.BeforeStart` |
| `bot/internal/state/state.go` | `DayTasksOn`, `DayTasksKnown` |
| `bot/internal/handlers/dayprefs.go` (new) | `dayPrefs`, `readDayPrefs`, `h.dayPrefs`, `h.dayTasksOn`, `h.setDayPref`, `h.home` |
| `bot/internal/handlers/daycolour.go` (new) | `dayColours`, `todayDayLine` |
| `bot/internal/handlers/calendar.go` | `dayCell.Colour`, `cellLabel`, `showMonth` |
| `bot/internal/handlers/today.go` | строка и кнопка |
| `bot/internal/handlers/daytasks.go` | экран настроек, гард `dt_` |
| `bot/internal/handlers/onboarding.go`, `menu.go` | шаг онбординга, одноразовый вопрос |
| `bot/internal/keyboards/daytasks.go`, `keyboards.go`, `menu.go` | `DaySettings`, `DayOnboard`, `DayTasksOff`, `TodayScreen`, `TaskActions`, `HomeInlineFor`, `SettingsMenu` |

---

### Task 1: API — `before_start`

**Files:** Modify `api-go/internal/daytasks/store.go` · Test `api-go/internal/daytasks/db_test.go`

**Interfaces:** Produces JSON field `before_start` (bool) on every day of `GET /api/day-tasks`.

- [ ] **Step 1: failing test** — append to `db_test.go`:

```go
// Spec §11: the start is the first day the person took; days before it (or
// every day, for someone who never took one) come back before_start.
func TestBeforeStartIsTheFirstTakenDay(t *testing.T) {
	_, _, user := dayDB(t)
	today := Today(time.Now(), ny)
	iso, yest := today.Format("2006-01-02"), today.AddDate(0, 0, -1).Format("2006-01-02")
	list := func() []Day {
		rec := callDay(t, ListHandler, user, http.MethodGet, "/api/day-tasks?from="+yest+"&to="+iso, nil, nil)
		if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"before_start"`)) {
			t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
		}
		var got struct{ Data []Day `json:"data"` }
		_ = json.Unmarshal(rec.Body.Bytes(), &got)
		return got.Data
	}
	if d := list(); !d[0].BeforeStart || !d[1].BeforeStart {
		t.Errorf("never took a day: want both before_start, got %+v", d)
	}
	id := newTask(t, user, map[string]any{"title": "дело", "priority": 1})
	if rec := callDay(t, ConfirmHandler, user, http.MethodPost, "/api/day-tasks/confirm",
		map[string]any{"day": iso, "task_ids": []string{id}}, nil); rec.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body.String())
	}
	if d := list(); !d[0].BeforeStart || d[1].BeforeStart {
		t.Errorf("took today: want yesterday before, today not; got %+v", d)
	}
}
```

- [ ] **Step 2:** `cd api-go && DATABASE_URL=… go test ./internal/daytasks/ -run TestBeforeStartIsTheFirstTakenDay -count=1` → FAIL (`BeforeStart` undefined).

- [ ] **Step 3: implement** — in `store.go` add to `Day` after `Level`:

```go
	// BeforeStart: the day is earlier than the first day this person ever
	// took, or they never took one (spec §11). One rule, on the server, so the
	// web cannot decide «the start» differently later.
	BeforeStart bool `json:"before_start"`
```

and in `List`, right after `userZoneAndTarget` and the `out` loop, **before** the `confirmed` query (the later `len(promTasks) == 0` early return must not skip it):

```go
	var first *string
	if err := db.Pool.QueryRow(ctx,
		`SELECT to_char(MIN(day), 'YYYY-MM-DD') FROM day_commitment_day WHERE user_id = $1`,
		userID).Scan(&first); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].BeforeStart = first == nil || out[i].Day < *first
	}
```

(`Day` strings are `YYYY-MM-DD`, so string order is date order; both are the user's own dates, no zone conversion.)

- [ ] **Step 4:** the test passes; full `go build ./... && go vet ./... && go test ./... -count=1` with `DATABASE_URL`. Scoping scanner stays green (read by `user_id`).
- [ ] **Step 5: sabotage** `first == nil || out[i].Day < *first` → `first == nil` → own test red; restore.
- [ ] **Step 6: commit** `feat(api): day-tasks days say before_start — earlier than the first day taken`

---

### Task 2: Bot — prefs, cache, the home keyboard

**Files:** Create `bot/internal/handlers/dayprefs.go`, `dayprefs_test.go` · Modify `bot/internal/api/daytasks.go`, `bot/internal/state/state.go`, `bot/internal/keyboards/menu.go`, every `keyboards.HomeInline(h.lang(chatID))` call in `bot/internal/handlers/*.go` (30 sites)

**Interfaces — Produces:**
- `api.Day.BeforeStart bool` (`json:"before_start"`)
- `type dayPrefs struct { On, PaintBefore bool; Target int }`
- `func readDayPrefs(s map[string]any) dayPrefs` — pure: absent `day_tasks_enabled` → `On=true`; absent `day_tasks_paint_before` → `false`; `day_tasks_target` float64 in 3..7 else 5
- `func (h *Handler) dayPrefs(chatID int64) dayPrefs` — `MySettings`; on error returns `dayPrefs{On: true, Target: 5}`; refreshes the cache
- `func (h *Handler) dayTasksOn(chatID int64) bool` — cache (`us.DayTasksKnown`) or `dayPrefs`
- `func (h *Handler) setDayPref(chatID int64, key string, v any) error` — `PatchSettings(token, {key: v})`; on success, if key is `day_tasks_enabled`, sets the cache
- `func (h *Handler) home(chatID int64) tgbotapi.InlineKeyboardMarkup` — `keyboards.HomeInlineFor(h.lang(chatID), h.dayTasksOn(chatID))`
- `keyboards.HomeInlineFor(lang i18n.Lang, dayTasks bool)`; `HomeInline(lang)` stays as `HomeInlineFor(lang, true)` for non-handler callers

- [ ] **Step 1: failing tests** (`dayprefs_test.go`):

```go
func TestReadDayPrefsDefaults(t *testing.T) {
	p := readDayPrefs(map[string]any{})
	if !p.On || p.PaintBefore || p.Target != 5 {
		t.Errorf("empty settings: %+v, want on, not painting, 5", p)
	}
	p = readDayPrefs(map[string]any{"day_tasks_enabled": false, "day_tasks_paint_before": true, "day_tasks_target": float64(3)})
	if p.On || !p.PaintBefore || p.Target != 3 {
		t.Errorf("set settings: %+v", p)
	}
}

// Review Focus 1: a failed read hides nothing.
func TestAFailedReadKeepsDayTasksOn(t *testing.T) {
	a := &dayAPI{meDown: true}
	h, _, chat := dayHandler(t, a)
	if !h.dayTasksOn(chat) {
		t.Error("a failed settings read switched day tasks off")
	}
}

// Review Focus 5: switching keeps every other key.
func TestSwitchingKeepsOtherSettings(t *testing.T) {
	h, _, patched := scaleAPI(t)
	if err := h.setDayPref(970, "day_tasks_enabled", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(*patched, `"day_tasks_enabled":false`) || !strings.Contains(*patched, `"lang":"ru"`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	if h.dayTasksOn(970) {
		t.Error("cache still says on")
	}
}

func TestTheHomeHidesDayTasksWhenOff(t *testing.T) {
	if m := fmt.Sprint(keyboards.HomeInlineFor(i18n.RU, false)); strings.Contains(m, "dt_d_today") {
		t.Errorf("off, and the menu still has 📌: %s", m)
	}
	if m := fmt.Sprint(keyboards.HomeInlineFor(i18n.RU, true)); !strings.Contains(m, "dt_d_today") {
		t.Errorf("on, and 📌 is missing: %s", m)
	}
}
```

Extend the `dayAPI` fake in `daytasks_test.go`: field `settings map[string]any` (top-level keys merged into the `/api/auth/me` settings object) and `meDown bool` (every `GET /api/auth/me` answers 500). Keep `bot.onboarded` and `bot.lang` in the response.

- [ ] **Step 2:** run → FAIL (undefined).
- [ ] **Step 3: implement.** `state.go`: `DayTasksOn, DayTasksKnown bool` beside `PriorityStyleKnown`. `menu.go`: `HomeInlineFor` = the current body with the `📌 Задачи дня` button only when `dayTasks`; when off the last row is `⚙️ Настройки` alone. `dayprefs.go` per the interfaces above. Replace every handler call `keyboards.HomeInline(h.lang(chatID))` with `h.home(chatID)` (script via Write + python, counted: 30 replacements, then `grep -rn "keyboards.HomeInline(" bot/internal/handlers --include=*.go | grep -v _test` gives 0 outside `dayprefs.go`). Add a scan test `TestHandlersUseTheHomeMethod` in `dayprefs_test.go` that reads `*.go` in `handlers/` (not `_test.go`, not `dayprefs.go`) and fails on `keyboards.HomeInline(`, with a floor of ≥ 20 files scanned.
- [ ] **Step 4:** bot `go build ./... && go vet ./... && go test ./... -count=1` green.
- [ ] **Step 5: sabotage** — in `readDayPrefs` make absent `day_tasks_enabled` read as false → `TestReadDayPrefsDefaults` red; in `dayPrefs` return `On:false` on error → `TestAFailedReadKeepsDayTasksOn` red; restore each.
- [ ] **Step 6: commit** `feat(bot): day-tasks switch read and written through the settings merge; the home menu hides 📌 when off`

---

### Task 3: Bot — colour in the month cell

**Files:** Create `bot/internal/handlers/daycolour.go`, `daycolour_test.go` · Modify `bot/internal/handlers/calendar.go`, `calendar_test.go`

**Interfaces:** Consumes `api.Day.BeforeStart`, `h.dayPrefs`. Produces `func dayColours(days []api.Day, today time.Time, paintBefore bool) map[string]string` (key `YYYY-MM-DD` → square) and `dayCell.Colour string`.

- [ ] **Step 1: failing tests** (`daycolour_test.go`):

```go
func TestDayColoursFollowTheRules(t *testing.T) {
	today := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	days := []api.Day{
		{Day: "2026-09-21", BeforeStart: true},             // before the start
		{Day: "2026-09-22", Level: 0},                      // after the start, not taken → ⬛
		{Day: "2026-09-23", Confirmed: true, Level: 3},     // 🟧
		{Day: "2026-09-24"},                                // today, not taken → none
		{Day: "2026-09-25", Confirmed: true, Level: 5},     // future → none
	}
	got := dayColours(days, today, false)
	want := map[string]string{"2026-09-22": "⬛", "2026-09-23": "🟧"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("not painting before: %v, want %v", got, want)
	}
	if got := dayColours(days, today, true); got["2026-09-21"] != "⬛" {
		t.Errorf("painting before: 21st = %q, want ⬛", got["2026-09-21"])
	}
	days[3].Confirmed, days[3].Level = true, 1
	if got := dayColours(days, today, false); got["2026-09-24"] != "🟫" {
		t.Errorf("today taken: %q, want 🟫", got["2026-09-24"])
	}
}

func TestTheCellCarriesColourDateAndBar(t *testing.T) {
	d := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	if got := cellLabel(dayCell{Date: d, InMonth: true, Level: 5, Colour: "🟩"}); got != "🟩 21 "+statgrid.Glyph(5) {
		t.Errorf("full cell = %q", got)
	}
	if got := cellLabel(dayCell{Date: d, InMonth: true, Colour: "🟧"}); got != "🟧 21" {
		t.Errorf("no bar = %q", got)
	}
	if got := cellLabel(dayCell{Date: d, InMonth: false, Colour: "🟧"}); got != "·21" {
		t.Errorf("other month = %q, want no colour", got)
	}
	if got := cellLabel(dayCell{Date: d, InMonth: true, IsToday: true, Colour: "🟧"}); got != "🟧 🔸21" {
		t.Errorf("today = %q", got)
	}
}
```

Plus a handler test in `calendar_test.go` through the `dayAPI` fake: a confirmed today with `level: 3` → the grid markup contains `🟧 🔸`; with `settings: {"day_tasks_enabled": false}` → no `🟧` anywhere in the markup.

- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3: implement.** `daycolour.go`:

```go
// dayColours is the square each day gets (spec §11): the future never,
// today only once taken, a day before the start only if the person chose so.
func dayColours(days []api.Day, today time.Time, paintBefore bool) map[string]string {
	t := today.Format("2006-01-02")
	out := map[string]string{}
	for _, d := range days {
		switch {
		case d.Day > t:
		case d.Day == t && !d.Confirmed:
		case d.BeforeStart && !paintBefore:
		default:
			out[d.Day] = format.DayLevel(d.Level)
		}
	}
	return out
}
```

`calendar.go`: `dayCell` gets `Colour string`; `buildMonth` takes `colours map[string]string` as a new last parameter and fills `Colour: colours[key]`; `cellLabel`:

```go
func cellLabel(c dayCell) string {
	n := strconv.Itoa(c.Date.Day())
	if !c.InMonth {
		return "·" + n
	}
	parts := []string{}
	if c.Colour != "" {
		parts = append(parts, c.Colour)
	}
	switch {
	case c.IsToday:
		parts = append(parts, "🔸"+n)
	case c.Level > 0:
		// A space between the number and the bar (Denis, 23.09).
		parts = append(parts, n, statgrid.Glyph(c.Level))
	default:
		parts = append(parts, n)
	}
	return strings.Join(parts, " ")
}
```

`showMonth`: after the events load, if `p := h.dayPrefs(chatID); p.On`, call `h.api.DayTasks(us.AuthToken, gridStart.Format("2006-01-02"), gridStart.AddDate(0, 0, 41).Format("2006-01-02"))`; on success `colours = dayColours(days, now, p.PaintBefore)`; on error draw without colour (same rule as the bars). Update every existing `buildMonth(` call in tests to pass `nil`.
- [ ] **Step 4:** bot suite green.
- [ ] **Step 5: sabotage** — drop the `d.Day == t && !d.Confirmed` case → today-not-taken test red; drop `c.Colour` from `cellLabel` → cell test red; restore.
- [ ] **Step 6: commit** `feat(bot): the month cell shows the day's colour — colour, date, bar`

---

### Task 4: Bot — 📅 Сегодня line and button

**Files:** Modify `bot/internal/handlers/today.go`, `daycolour.go`, `bot/internal/keyboards/keyboards.go` (`TodayScreen`) · Test `daycolour_test.go`, `keyboards/keyboards_test.go`

**Interfaces:** Produces `func todayDayLine(lang i18n.Lang, d api.Day) string`; `keyboards.TodayScreen(lang i18n.Lang, dayTasks bool)`.

- [ ] **Step 1: failing tests:**

```go
func TestTodayLine(t *testing.T) {
	taken := api.Day{Confirmed: true, Done: 3, Target: 5, Level: 3}
	if got := todayDayLine(i18n.RU, taken); got != "📌 🟧 3 из 5" {
		t.Errorf("taken = %q", got)
	}
	if got := todayDayLine(i18n.RU, api.Day{Target: 5}); got != "📌 День ещё не взят" {
		t.Errorf("not taken = %q", got)
	}
}
```

and in `keyboards_test.go`: `TodayScreen(i18n.RU, true)` contains `dt_d_today`; `TodayScreen(i18n.RU, false)` does not.

- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3: implement.**

```go
func todayDayLine(lang i18n.Lang, d api.Day) string {
	if !d.Confirmed {
		return i18n.T(lang, "📌 День ещё не взят", "📌 The day is not taken yet")
	}
	return fmt.Sprintf(i18n.T(lang, "📌 %s %d из %d", "📌 %s %d of %d"), format.DayLevel(d.Level), d.Done, d.Target)
}
```

`TodayScreen(lang, dayTasks)`: first row `📋 Задачи` plus, when `dayTasks`, `📌 Задачи дня` (`dt_d_today`); second row `« Меню`. In `handleToday`, after the title block and before `📅 События`: when `h.dayTasksOn(chatID)`, read `DayTasks(today, today)` (today = `h.userToday(chatID)`); on success append `todayDayLine(...) + "\n\n"`; on error append nothing. Pass `h.dayTasksOn(chatID)` to `TodayScreen`.
- [ ] **Step 4:** suite green.
- [ ] **Step 5: sabotage** `!d.Confirmed` → `d.Confirmed` → own test red; restore.
- [ ] **Step 6: commit** `feat(bot): Today shows the day's colour and opens day tasks`

---

### Task 5: Bot — ⚙️ → 📌 Задачи дня settings screen

**Files:** Modify `bot/internal/keyboards/daytasks.go` (`DayTarget` → `DaySettings`), `keyboards.go` (`SettingsMenu` label), `bot/internal/handlers/daytasks.go` (`handleDayTarget` → `handleDaySettings`), `handler.go` (routes), `schedule_test.go` (`routed` += `"dts_"`), `daytarget_test.go`, `daytasks_test.go` · help text `HelpDayTasks` gets one line about the switch

**Interfaces:** Produces `keyboards.DaySettings(lang i18n.Lang, p DaySettingsView)` with `type DaySettingsView struct { On, PaintBefore bool; Target int }`; callbacks `settings_dtn` (show), `dtn_<3..7>`, `dts_on`, `dts_off`, `dts_pb_on`, `dts_pb_off`; handler `func (h *Handler) handleDaySettings(chatID int64, messageID int, action string)` where action ∈ `"" | "n:<3..7>" | "on" | "off" | "pb_on" | "pb_off"`.

- [ ] **Step 1: failing tests** (`daytarget_test.go`, using `scaleAPI` as the existing target tests do):

```go
func TestTheDaySettingsScreen(t *testing.T) {
	h, fake, patched := scaleAPI(t)
	h.handleDaySettings(980, 0, "off")
	if !strings.Contains(*patched, `"day_tasks_enabled":false`) || !strings.Contains(*patched, `"lang":"ru"`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	m := fake.last(t).Markup
	if strings.Contains(m, "dtn_") || !strings.Contains(m, "dts_on") {
		t.Errorf("off: the target row must go, «включить» must stay: %s", m)
	}
	h.handleDaySettings(980, 0, "on")
	h.handleDaySettings(980, 0, "pb_on")
	if !strings.Contains(*patched, `"day_tasks_paint_before":true`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	if m := fake.last(t).Markup; !strings.Contains(m, "dts_pb_off") || !strings.Contains(m, "dtn_") {
		t.Errorf("on + painting: %s", m)
	}
}
```

The two existing target tests switch to `handleDaySettings(…, "n:4")` / `"n:9"` with the same assertions (`✓ 4`, nothing written for 9).

- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3: implement.** Keyboard rows: `[✅ Включены]` (`dts_off`) or `[❌ Выключены · включить]` (`dts_on`); when on: the 3…7 row (`dtn_`, `✓` on current), `[🎨 Дни до начала: не красить]` (`dts_pb_on`) or `[🎨 Дни до начала: ⬛]` (`dts_pb_off`); last row `HelpButton(lang, HelpDayTasks)` + `« Настройки`. Screen text:

```
📌 <b>Задачи дня</b>

Каждый день берёшь N дел, и день красится по сделанному. Цвет виден здесь, в календаре и на «Сегодня».

«Дни до начала»: красить ли дни до первого взятого. Не красить: они остаются как были.
```

(EN mirror.) Handler: parse action; `n:` validates 3..7 (else return without writing), writes `day_tasks_target`; `on`/`off` write `day_tasks_enabled`; `pb_on`/`pb_off` write `day_tasks_paint_before`; any write error → `❌ Не сохранилось: ` + `errorText`, no redraw; then read `h.dayPrefs` and draw — **but** tick the value just written (review M5, 23.09: the tick comes from the write, not a second read). `handler.go`: `settings_dtn` → `handleDaySettings(…, "")`; `dtn_` → `"n:"+rest`; `dts_` → `strings.TrimPrefix(data, "dts_")`. `SettingsMenu`: button label `📌 Задачи дня`, callback unchanged `settings_dtn`.
- [ ] **Step 4:** suite green (help scan, i18n scan, routed-prefix test, dash scan).
- [ ] **Step 5: sabotage** — make `off` write `true` → own test red; restore.
- [ ] **Step 6: commit** `feat(bot): ⚙️ 📌 Задачи дня: switch, tasks per day, days before the start`

---

### Task 6: Bot — hidden when off (card, old buttons)

**Files:** Modify `bot/internal/keyboards/keyboards.go` (`TaskActions`), `task_test.go`, `bot/internal/handlers/tasks.go:159`, `daytasks.go` (guard), `keyboards/daytasks.go` (`DayTasksOff`), `daytasks_test.go`

**Interfaces:** `keyboards.TaskActions(lang, taskID string, repeats bool, linkedEventID string, dayTasks bool)`; `keyboards.DayTasksOff(lang)` = `[✅ Включить]` (`dts_on`) + `« Меню` (`main_menu`) + `HelpButton(lang, HelpDayTasks)`.

- [ ] **Step 1: failing tests:**

```go
// Review Focus 2: an old 📌 button, day tasks off → a sentence, no screen, no write.
func TestAnOldDayTasksButtonWhenOff(t *testing.T) {
	a := &dayAPI{settings: map[string]any{"day_tasks_enabled": false}}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_put_"+moscowToday()+"_"+dtOne)
	if a.called("POST /api/day-tasks") {
		t.Errorf("wrote while off: %v", a.calls)
	}
	if got := fake.last(t); !strings.Contains(got.Text, "выключены") || !strings.Contains(got.Markup, "dts_on") {
		t.Errorf("off answered %q / %s", got.Text, got.Markup)
	}
}
```

and in `task_test.go`: `TaskActions(i18n.RU, id, false, "", false)` has no `dt_pin_`; with `true` it has.

- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3: implement.** `TaskActions` shows `📌 В задачи дня` only when `dayTasks`; `tasks.go:159` passes `h.dayTasksOn(chatID)`. At the top of `handleDayTasksCallback`, after the prefix check: `if !h.dayTasksOn(chatID) { h.editOrSend(chatID, messageID, h.t(chatID, "📌 Задачи дня выключены. Включить можно здесь или в ⚙️ Настройках.", "📌 Day tasks are off. Switch them on here or in ⚙️ Settings."), keyboards.DayTasksOff(h.lang(chatID))); return true }`.
- [ ] **Step 4:** suite green.
- [ ] **Step 5: sabotage** — remove the guard → own test red; restore.
- [ ] **Step 6: commit** `feat(bot): day tasks off hides 📌 on the card; old buttons say so`

---

### Task 7: Bot — onboarding step and the one-time question

**Files:** Modify `bot/internal/handlers/onboarding.go`, `statsview.go:155` (scale «Дальше →» goes to `ob_dt`), `menu.go`, `daytasks.go`, `keyboards/daytasks.go` (`DayOnboard`), `handler.go` (route `dtq_`), `schedule_test.go` (`routed` += `"dtq_"`), onboarding tests

**Interfaces:** `keyboards.DayOnboard(lang, onData, offData string)` = `[✅ Включить]` `[Не сейчас]` + `HelpButton(lang, HelpDayTasks)`; callbacks `ob_dt` (show), `ob_dt_on`, `ob_dt_off`, `dtq_on`, `dtq_off`; `func (h *Handler) askDayTasksOnce(chatID int64, messageID int) bool`.

Text (both entrances):

```
📌 <b>Задачи дня</b>

Каждое утро берёшь несколько дел на день. День красится по сделанному:
🟩 всё · 🟨 почти · 🟧 больше половины · 🟥 мало · ⬛ ничего

В календаре это выглядит так: 🟩 21 ▅

Включить? Поменять можно в ⚙️ Настройках.
```

- [ ] **Step 1: failing tests:**
  - onboarding: after `ob_sc_<kind>` the «Дальше →» button carries `ob_dt`; `ob_dt_off` writes `"day_tasks_enabled":false` and then shows the closing (`ob_finish` path); `ob_dt_on` writes `true` and shows the 3…7 row whose «Дальше →» is `ob_finish`.
  - Review Focus 3: `askDayTasksOnce` returns false when `settings` has `day_tasks_enabled` (any value); returns true once and writes `bot.day_tasks_asked = "1"` when the key is absent and the user is onboarded; a second call returns false.
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3: implement.** Onboarding cases in `handleOnboardCallback`: `ob_dt` (`us.FlowStep = "daytasks"`, show `DayOnboard(lang, "ob_dt_on", "ob_dt_off")`); `ob_dt_on` → `setDayPref(enabled, true)`, then the target row with prefix `ob_dtn_` and next `ob_finish` (add `ob_dtn_<n>` case writing `day_tasks_target`); `ob_dt_off` → `setDayPref(enabled, false)` then `onboardClosing`. `statsview.go:155` onboarding next: `"ob_dt"`. `askDayTasksOnce` mirrors `askPriorityOnce`: requires `us.Onboarded`; skip if `MySettings` fails or has `day_tasks_enabled`; skip if `BotSetting("day_tasks_asked") != ""`; write the asked flag first (fail → do not ask); show `DayOnboard(lang, "dtq_on", "dtq_off")`. `dtq_on`/`dtq_off` write the switch and open the menu. `handleMenu`: after `askPriorityOnce`, `if h.askDayTasksOnce(chatID, messageID) { return }`.
- [ ] **Step 4:** suite green.
- [ ] **Step 5: sabotage** — drop the «has `day_tasks_enabled`» skip → Review Focus 3 test red; restore.
- [ ] **Step 6: commit** `feat(bot): onboarding asks about day tasks; people onboarded earlier are asked once`

---

### Task 8: dev + check list

- [ ] Whole-branch review (fresh reviewer, Opus) against spec §11 and this plan's Review Focus; fix Critical/Important.
- [ ] Push `develop` **only on Denis's word**; CI green including e2e; `scripts/deploy-dev-bot.sh`.
- [ ] Check list `docs/proverka-bota-2026-09-24-zadachi-dnya-d3.md` from spec §11 «Приёмка D3» (8 items) plus: **width of `🟧 🔸24` on his phone** (spec §6 risk); open in Obsidian.
- [ ] Release notes `v0.4.11.6` in `bot/internal/release/notes.go` only after his check.
