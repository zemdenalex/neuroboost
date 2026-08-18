# Bot Navigation, Calendar and Task Creation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The bot's menu stops being a caption that points at a reply button — every screen carries the actions you can take from it, the calendar day pages in place, and a task is created from one typed line with optional fields added by button.

**Architecture:** The reply keyboard shrinks to six permanent *entrances*; everything past an entrance is inline and edits the message it was pressed on. A new `editOrSend` helper carries the `messageID` that in-place editing needs. Task creation gains a pure parser modelled on the existing `internal/parse/event.go`, so the line-to-task logic is testable without Telegram.

**Tech Stack:** Go 1.22, `go-telegram-bot-api/v5 v5.5.1` (**not** changed in this plan), module `bot/` only.

**Spec:** `docs/superpowers/specs/2026-08-18-bot-navigation-and-task-creation-design.md`

## Global Constraints

- **Module `bot/` only.** No file under `api-go/` or `web/` is created or modified by this plan. The API already accepts `description`, `due_date`, `estimated_minutes`, `tags` (`api-go/internal/tasks/types.go`, `CreateTaskRequest`).
- **The library is NOT migrated here.** `go-telegram-bot-api/v5 v5.5.1` stays. Migration lives in `docs/superpowers/specs/2026-08-18-bot-telegram-native-and-sharing-design.md`.
- **Verification command, every task:** `cd bot && go build ./... && go vet ./... && go test ./...`
- 🔴 **`callback_data` is 64 bytes.** A UUID is 36. Every new keyboard gets a test asserting the length — never arithmetic in a comment.
- 🔴 **A test counts only once it has been shown red.** Break the thing it guards, watch it fail, restore. A green test that was never seen failing is not evidence.
- 🔴 **Priority is inverted:** 1 = Emergency, 5 = If Possible, 0 = Buffer. `!1` is the most urgent.
- 🔴 **No token in logs.** Errors go through `logsafe.Redact`.
- 🔴 **Claims about v0.2.1 cite a `bot.action` line** in `_legacy/snapshots-v0.0.1-v0.4.0/v0.2.1/apps/bot/src/index.mjs` — never a keyboard in `keyboards.mjs`.
- **Commits:** `git add` named files only. Never `-A`. Never a Claude/Anthropic co-author trailer.
- **Do not push to `develop`** and do not merge to `main` without Denis saying so; merge to `main` is a production release.

---

## File Structure

| File | Responsibility |
|---|---|
| `bot/internal/handlers/handler.go` (modify) | Routing; gains `editOrSend`; callback router learns the new prefixes |
| `bot/internal/handlers/menu.go` (create) | The `🏠 Меню` inline home screen and its summary |
| `bot/internal/handlers/agenda.go` (create) | The `📅 События` upcoming-events screen |
| `bot/internal/handlers/calendar.go` (modify) | Day view edits in place; day paging; "Today" jump |
| `bot/internal/handlers/newtask.go` (create) | The task card, `✅ Создать`, and the `📝 Подробнее` wizard |
| `bot/internal/handlers/tasks.go` (modify) | Empty state gets a button; card gains Срок/Оценка/Теги |
| `bot/internal/handlers/screens_test.go` (create) | The standing test that no screen points at a reply button |
| `bot/internal/keyboards/menu.go` (create) | Six-button reply keyboard; the home inline keyboard |
| `bot/internal/keyboards/keyboards.go` (modify) | `MonthGrid` gains "Today"; `DayView` gains paging |
| `bot/internal/keyboards/task.go` (create) | Task card keyboard, wizard step keyboards |
| `bot/internal/parse/task.go` (create) | `ParseTask` — one line to title/priority/due/minutes/tags |
| `bot/internal/parse/task_test.go` (create) | Parser tests |
| `bot/internal/api/client.go` (modify) | `CreateTaskReq` gains the optional fields; `Task` gains `Tags` |

---

## Task 1: `editOrSend` — one place that decides edit vs send

**Files:**
- Modify: `bot/internal/handlers/handler.go`
- Test: `bot/internal/handlers/editorsend_test.go` (create)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `func (h *Handler) editOrSend(chatID int64, messageID int, text string, kb tgbotapi.InlineKeyboardMarkup)` and `func shouldSendNew(messageID int, editErr error) bool`. Every later task navigates through `editOrSend` instead of `sendHTMLWithKeyboard`.

Today only `showMonth` and `handleCalendarNav` edit in place; everything else posts a new message, which is the mechanical cause of "menu doesn't show extra buttons" feeling like a wall of messages.

- [ ] **Step 1: Write the failing test**

```go
package handlers

import (
	"errors"
	"testing"
)

// shouldSendNew holds the whole edit-or-send decision, because the surrounding
// method needs a bot, a store and a live API to run at all — and the decision
// is the only part that can be wrong.
func TestShouldSendNewWhenThereIsNoMessageToEdit(t *testing.T) {
	if !shouldSendNew(0, nil) {
		t.Error("messageID 0 means the press came from a reply button: there is nothing to edit")
	}
}

func TestShouldNotSendNewOnASuccessfulEdit(t *testing.T) {
	if shouldSendNew(42, nil) {
		t.Error("a successful edit must not also post a message — that is the message spam being removed")
	}
}

func TestShouldSendNewWhenTelegramRefusesTheEdit(t *testing.T) {
	// 🔴 Telegram forbids editing a message older than 48 hours. Staying silent
	// there is the worst outcome: the user pressed a button and nothing at all
	// happened, which reads as a dead bot.
	err := errors.New("Bad Request: message can't be edited")
	if !shouldSendNew(42, err) {
		t.Error("a refused edit must fall back to a new message, not silence")
	}
}

func TestNotModifiedIsNotAFailure(t *testing.T) {
	// Tapping the same page twice produces byte-identical text and markup.
	// That is not an error, and it must not spawn a duplicate message.
	err := errors.New("Bad Request: message is not modified")
	if shouldSendNew(42, err) {
		t.Error("an identical re-render must be swallowed, not answered with a new message")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bot && go test ./internal/handlers/ -run TestShouldSendNew -v`
Expected: FAIL — `undefined: shouldSendNew`

- [ ] **Step 3: Write minimal implementation**

Add to `bot/internal/handlers/handler.go`:

```go
// shouldSendNew decides whether a navigation step must post a new message.
//
// Pulled out of editOrSend as a pure function on purpose: the method around it
// cannot run without a bot, a store and a reachable API, and this decision is
// the only part of it that can be wrong.
func shouldSendNew(messageID int, editErr error) bool {
	if messageID == 0 {
		// The press came from a reply-keyboard button or a command; there is no
		// message of ours to edit.
		return true
	}
	if editErr == nil {
		return false
	}
	// An identical re-render. Not a failure, and answering it with a fresh
	// message would double the screen on every second tap.
	if strings.Contains(editErr.Error(), "message is not modified") {
		return false
	}
	// Anything else — most often a message past Telegram's 48-hour edit window.
	// Falling back to a new message is the only outcome the user can see.
	return true
}

// editOrSend renders a screen onto the message it was triggered from, or posts
// a new one when there is nothing to edit.
func (h *Handler) editOrSend(chatID int64, messageID int, text string, kb tgbotapi.InlineKeyboardMarkup) {
	if messageID != 0 {
		edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, kb)
		edit.ParseMode = "HTML"
		_, err := h.bot.Send(edit)
		if !shouldSendNew(messageID, err) {
			return
		}
		if err != nil {
			log.Printf("edit in chat %d fell back to a new message: %s", chatID, logsafe.Redact(err))
		}
	}
	h.sendHTMLWithKeyboard(chatID, text, kb)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd bot && go test ./internal/handlers/ -run TestShouldSendNew -v && go test ./internal/handlers/ -run TestNotModified -v`
Expected: PASS

- [ ] **Step 5: Prove the tests can fail**

Temporarily change `if messageID == 0 { return true }` to `return false`. Run the tests: `TestShouldSendNewWhenThereIsNoMessageToEdit` must FAIL. Restore the line and confirm PASS. Record both outcomes in the task report — a test never seen red is not evidence.

- [ ] **Step 6: Full verification**

Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add bot/internal/handlers/handler.go bot/internal/handlers/editorsend_test.go
git commit -m "feat(bot): one place decides whether a screen edits its message or posts a new one"
```

---

## Task 2: Six reply entrances and a live `🏠 Меню`

**Files:**
- Create: `bot/internal/keyboards/menu.go`
- Create: `bot/internal/handlers/menu.go`
- Modify: `bot/internal/keyboards/keyboards.go` (delete the old `MainMenu`)
- Modify: `bot/internal/handlers/handler.go` (`HandleMessage` switch, `HandleCallback` routing)
- Modify: `bot/internal/handlers/commands.go` (`handleStart`)
- Test: `bot/internal/keyboards/menu_test.go` (create)

**Interfaces:**
- Consumes: `editOrSend` from Task 1.
- Produces: `keyboards.MainMenu() tgbotapi.ReplyKeyboardMarkup` (six buttons), `keyboards.HomeInline() tgbotapi.InlineKeyboardMarkup`, `func (h *Handler) handleMenu(chatID int64, messageID int)`, and the callback id `"main_menu"` (already routed).

Denis, verbatim: *«reply (keyboard buttons) must have menu, calendar, events, tasks… each of keyboard buttons open inline menus. Actually I think 6 buttons, the ones I said + add new (anything) and settings»*.

- [ ] **Step 1: Write the failing test**

```go
package keyboards

import (
	"strings"
	"testing"
)

// The six entrances, in order. This list is the decision itself, so the test
// states it literally rather than deriving it from the code under test — a
// test whose expectation is computed from its subject cannot disagree with it.
var wantEntrances = []string{
	"🏠 Меню", "🗓 Календарь",
	"📅 События", "📋 Задачи",
	"➕ Создать", "⚙️ Настройки",
}

func TestMainMenuHasExactlySixEntrances(t *testing.T) {
	kb := MainMenu()
	var got []string
	for _, row := range kb.Keyboard {
		for _, b := range row {
			got = append(got, b.Text)
		}
	}
	if len(got) != len(wantEntrances) {
		t.Fatalf("reply keyboard has %d buttons, want %d: %v", len(got), len(wantEntrances), got)
	}
	for i := range wantEntrances {
		if got[i] != wantEntrances[i] {
			t.Errorf("button %d is %q, want %q", i, got[i], wantEntrances[i])
		}
	}
}

func TestMainMenuCarriesNoActions(t *testing.T) {
	// 🔴 The rule the six were chosen by: the reply keyboard holds ENTRANCES,
	// never actions. "New Task", "Note" and "New Event" used to sit here, which
	// is why every inline screen ended up pointing back at it in prose.
	banned := []string{"Note", "Заметк", "New Task", "New Event", "Stats", "Статист", "Planning", "Планир"}
	kb := MainMenu()
	for _, row := range kb.Keyboard {
		for _, b := range row {
			for _, bad := range banned {
				if strings.Contains(b.Text, bad) {
					t.Errorf("reply button %q is an action, not an entrance — it belongs inline", b.Text)
				}
			}
		}
	}
}

func TestHomeInlineCallbacksFitTelegramsBudget(t *testing.T) {
	for _, row := range HomeInline().InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == nil {
				continue
			}
			if n := len(*b.CallbackData); n > 64 {
				t.Errorf("callback_data %q is %d bytes, Telegram allows 64", *b.CallbackData, n)
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bot && go test ./internal/keyboards/ -run TestMainMenu -v`
Expected: FAIL — the current `MainMenu` has nine buttons including `📝 Note`, `➕ New Task`, `📅 New Event`, `🗂 Planning`, `📊 Stats`.

- [ ] **Step 3: Write the keyboards**

Create `bot/internal/keyboards/menu.go`:

```go
package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainMenu is the reply keyboard, and it holds ENTRANCES ONLY.
//
// Denis, 18.08: six buttons — menu, calendar, events, tasks, create, settings —
// and "each of keyboard buttons open inline menus". The previous keyboard mixed
// entrances with actions (Note, New Task, New Event, Planning, Stats), and that
// is the mechanical reason inline screens kept saying "Use ➕ New Task" instead
// of offering a button: the action lived somewhere a screen could only point at.
func MainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Меню"),
			tgbotapi.NewKeyboardButton("🗓 Календарь"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📅 События"),
			tgbotapi.NewKeyboardButton("📋 Задачи"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Создать"),
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
		),
	)
}

// HomeInline is what 🏠 Меню actually offers — the screens that are no longer
// reply buttons.
func HomeInline() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎯 Сегодня", "today_focus"),
			tgbotapi.NewInlineKeyboardButtonData("🗂 Планирование", "planning"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "stats"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings_menu"),
		),
	)
}

// CreateMenu is what ➕ Создать offers. 📝 Заметка still writes a task today;
// the notes entity is a separate spec
// (specs/2026-08-18-notes-as-their-own-entity-design.md) and this button moves
// to it unchanged when that lands.
func CreateMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Задача", "new_task"),
			tgbotapi.NewInlineKeyboardButtonData("📅 Событие", "new_event"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Заметка", "new_note"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Меню", "main_menu"),
		),
	)
}
```

Delete the old `MainMenu` from `bot/internal/keyboards/keyboards.go`.

- [ ] **Step 4: Write the home screen handler**

Create `bot/internal/handlers/menu.go`:

```go
package handlers

import (
	"fmt"
	"sort"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// handleMenu draws the home screen — state first, buttons under it.
//
// 🔴 The summary must never be able to stop the screen from opening. This is a
// NAVIGATION screen: if the API is down, the user still needs the buttons that
// take them elsewhere. Both loads therefore degrade to a line of text rather
// than to an early return.
func (h *Handler) handleMenu(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	loc := h.location()
	now := time.Now().In(loc)
	from, to := dayBounds(now, loc)

	text := fmt.Sprintf("🧠 <b>NeuroBoost</b> · %s\n─────────────\n", now.Format("02.01.2006"))

	events, evErr := h.api.GetEvents(us.AuthToken, from, to)
	tasks, tErr := h.api.GetTasks(us.AuthToken, "TODO")

	switch {
	case evErr != nil && tErr != nil:
		text += "Не смог загрузить сводку.\n"
	default:
		text += fmt.Sprintf("Сегодня: %d событий · %d задач\n", len(events), len(tasks))
		if len(events) > 0 {
			sort.Slice(events, func(i, j int) bool { return events[i].StartsAt < events[j].StartsAt })
			text += fmt.Sprintf("Ближайшее: %s %s\n",
				format.FormatTime(events[0].StartsAt, h.cfg.Timezone),
				format.Escape(events[0].Title))
		}
	}

	h.editOrSend(chatID, messageID, text, keyboards.HomeInline())
}
```

- [ ] **Step 5: Rewire the routing**

In `bot/internal/handlers/handler.go`, replace the `switch msg.Text` block with the six entrances (a reply press has no message of ours to edit, so `messageID` is `0`):

```go
	switch msg.Text {
	case "🏠 Меню":
		h.handleMenu(chatID, 0)
	case "🗓 Календарь":
		h.handleCalendar(chatID, time.Now())
	case "📅 События":
		h.handleAgenda(chatID, 0)
	case "📋 Задачи":
		h.handleTasks(chatID, 0)
	case "➕ Создать":
		h.sendHTMLWithKeyboard(chatID, "Что создаём?", keyboards.CreateMenu())
	case "⚙️ Настройки":
		h.handleSettings(chatID)
	default:
		h.sendHTMLWithKeyboard(chatID, "Не понял. Вот меню:", keyboards.HomeInline())
	}
```

In `HandleCallback`, point `main_menu` at the new screen and add the entries `CreateMenu` needs:

```go
	case data == "main_menu":
		h.handleMenu(chatID, cb.Message.MessageID)
	case data == "stats":
		h.handleStats(chatID)
	case data == "create_menu":
		h.editOrSend(chatID, cb.Message.MessageID, "Что создаём?", keyboards.CreateMenu())
	case data == "new_task":
		h.startNewTaskFlow(chatID)
	case data == "new_event":
		h.startNewEventFlow(chatID)
	case data == "new_note":
		h.startNoteFlow(chatID)
```

In `bot/internal/handlers/commands.go`, `handleStart` sends the reply keyboard once and then hands over to the home screen:

```go
func (h *Handler) handleStart(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🧠 <b>NeuroBoost</b>")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboards.MainMenu()
	h.send(chatID, msg)
	h.handleMenu(chatID, 0)
}
```

⚠ `handleTasks` and `handleAgenda` take a `messageID` here. `handleAgenda` arrives in Task 3 and `handleTasks` gains its parameter in Task 4 — until then, add the parameter to `handleTasks` and ignore it, so this task compiles on its own.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd bot && go test ./internal/keyboards/ -v`
Expected: PASS

- [ ] **Step 7: Prove the test can fail**

Add `tgbotapi.NewKeyboardButton("📝 Note")` to a row of `MainMenu`. Run the tests: both `TestMainMenuHasExactlySixEntrances` and `TestMainMenuCarriesNoActions` must FAIL. Remove it and confirm PASS.

- [ ] **Step 8: Full verification**

Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add bot/internal/keyboards/menu.go bot/internal/keyboards/menu_test.go bot/internal/keyboards/keyboards.go bot/internal/handlers/menu.go bot/internal/handlers/handler.go bot/internal/handlers/commands.go
git commit -m "feat(bot): six reply entrances, and a home screen that shows state instead of a greeting"
```

---

## Task 3: `📅 События` — the upcoming list

**Files:**
- Create: `bot/internal/handlers/agenda.go`
- Test: `bot/internal/handlers/agenda_test.go` (create)

**Interfaces:**
- Consumes: `editOrSend` (Task 1); `api.Event` (`bot/internal/api/client.go`).
- Produces: `func (h *Handler) handleAgenda(chatID int64, messageID int)` and the pure `func agendaText(events []api.Event, now time.Time, tz string) string`.

`🗓 Календарь` answers "what is on the 24th". `📅 События` answers "what is next" — a list from today forward. Neither the bot nor the web has this today (`docs/razbor-mobilnyy-kalendar-2026-08-19.md` §3 names it as the one view missing from both).

- [ ] **Step 1: Write the failing test**

```go
package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

func TestAgendaGroupsByDayAndSaysWhichDay(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	events := []api.Event{
		{Title: "Созвон", StartsAt: "2026-08-18T11:00:00Z"},
		{Title: "Ужин", StartsAt: "2026-08-19T16:00:00Z"},
	}
	got := agendaText(events, now, "UTC")

	if !strings.Contains(got, "Сегодня") {
		t.Errorf("an event today is not labelled Сегодня:\n%s", got)
	}
	if !strings.Contains(got, "Завтра") {
		t.Errorf("an event tomorrow is not labelled Завтра:\n%s", got)
	}
	if strings.Index(got, "Созвон") > strings.Index(got, "Ужин") {
		t.Errorf("events are not in chronological order:\n%s", got)
	}
}

func TestAgendaSaysSoWhenThereIsNothing(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	got := agendaText(nil, now, "UTC")
	if got == "" {
		t.Error("an empty agenda rendered an empty message — the screen would look broken")
	}
	if !strings.Contains(got, "Ничего") {
		t.Errorf("an empty agenda does not say it is empty:\n%s", got)
	}
}

func TestAgendaEscapesTitlesForHTML(t *testing.T) {
	// 🔴 The message is sent with ParseMode HTML. A title containing < or & is
	// not a hypothetical: it breaks the WHOLE message, so the user sees nothing
	// at all rather than one odd title.
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	events := []api.Event{{Title: "R&D <срочно>", StartsAt: "2026-08-18T11:00:00Z"}}
	got := agendaText(events, now, "UTC")
	if strings.Contains(got, "<срочно>") {
		t.Errorf("an unescaped title reached an HTML message:\n%s", got)
	}
	if !strings.Contains(got, "&amp;") {
		t.Errorf("the ampersand was not escaped:\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bot && go test ./internal/handlers/ -run TestAgenda -v`
Expected: FAIL — `undefined: agendaText`

- [ ] **Step 3: Write the implementation**

Create `bot/internal/handlers/agenda.go`:

```go
package handlers

import (
	"fmt"
	"sort"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// agendaHorizon is how far forward 📅 События looks.
//
// Two weeks rather than "everything": the screen answers "what is next", and a
// list long enough to scroll answers a different question — that one belongs to
// the month grid.
const agendaHorizon = 14 * 24 * time.Hour

// agendaText renders the upcoming list. Pure, so the day labels and the
// ordering are testable without a bot or an API.
func agendaText(events []api.Event, now time.Time, tz string) string {
	if len(events) == 0 {
		return "📅 <b>Ближайшие события</b>\n─────────────\nНичего не запланировано на две недели вперёд."
	}

	sorted := make([]api.Event, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartsAt < sorted[j].StartsAt })

	loc := now.Location()
	if l, err := time.LoadLocation(tz); err == nil {
		loc = l
	}
	today := now.In(loc).Format("2006-01-02")
	tomorrow := now.In(loc).AddDate(0, 0, 1).Format("2006-01-02")

	text := "📅 <b>Ближайшие события</b>\n─────────────\n"
	lastDay := ""
	for _, e := range sorted {
		start, err := time.Parse(time.RFC3339, e.StartsAt)
		if err != nil {
			// A row we cannot place in time is still a row the user owns. Show
			// it without a heading rather than dropping it silently.
			text += fmt.Sprintf("🕐 — %s\n", format.Escape(e.Title))
			continue
		}
		day := start.In(loc).Format("2006-01-02")
		if day != lastDay {
			switch day {
			case today:
				text += "\n<b>Сегодня</b>\n"
			case tomorrow:
				text += "\n<b>Завтра</b>\n"
			default:
				text += fmt.Sprintf("\n<b>%s</b>\n", start.In(loc).Format("02.01, Mon"))
			}
			lastDay = day
		}
		text += fmt.Sprintf("🕐 %s — %s\n",
			format.FormatTime(e.StartsAt, tz), format.Escape(e.Title))
	}
	return text
}

func (h *Handler) handleAgenda(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	loc := h.location()
	now := time.Now().In(loc)
	from := now.UTC().Format(time.RFC3339)
	to := now.Add(agendaHorizon).UTC().Format(time.RFC3339)

	events, err := h.api.GetEvents(us.AuthToken, from, to)
	if err != nil {
		h.editOrSend(chatID, messageID,
			"⚠️ Не дозвонился до сервера. Попробуй через минуту.", keyboards.HomeInline())
		return
	}
	h.editOrSend(chatID, messageID, agendaText(events, now, h.cfg.Timezone), keyboards.AgendaActions())
}
```

Add to `bot/internal/keyboards/menu.go`:

```go
// AgendaActions sits under 📅 События. The screen offers what can be done from
// it rather than naming a reply button.
func AgendaActions() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Событие", "new_event"),
			tgbotapi.NewInlineKeyboardButtonData("🗓 Календарь", "cal_open"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Меню", "main_menu"),
		),
	)
}
```

Route `cal_open` in `HandleCallback`:

```go
	case data == "cal_open":
		h.handleCalendar(chatID, time.Now())
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd bot && go test ./internal/handlers/ -run TestAgenda -v`
Expected: PASS

- [ ] **Step 5: Prove the escaping test can fail**

Replace `format.Escape(e.Title)` with `e.Title` on the main line. `TestAgendaEscapesTitlesForHTML` must FAIL. Restore and confirm PASS.

- [ ] **Step 6: Full verification**

Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add bot/internal/handlers/agenda.go bot/internal/handlers/agenda_test.go bot/internal/keyboards/menu.go bot/internal/handlers/handler.go
git commit -m "feat(bot): an upcoming-events screen that answers what is next"
```

---

## Task 4: The standing test that no screen points at a reply button

**Files:**
- Create: `bot/internal/handlers/screens_test.go`
- Modify: `bot/internal/handlers/tasks.go`
- Modify: `bot/internal/handlers/calendar.go`

**Interfaces:**
- Consumes: `keyboards.TaskListEmpty()` and `keyboards.DayActions(date string, year, month int)` — both created here.
- Produces: nothing later tasks depend on, except the rule itself.

This is the task that keeps the spec's central rule alive after everyone forgets it. It must be shown red on today's code and it must find the remaining violations.

- [ ] **Step 1: Write the failing test**

```go
package handlers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 🔴 The rule: no screen tells the user to go press a reply button. If an
// action is available from a screen, that screen carries a BUTTON for it.
//
// This is a source scan rather than a behavioural test on purpose. The prose
// lives in string literals scattered across handlers, there is no single place
// to intercept it, and the failure mode is a sentence — not a state. A future
// edit that reintroduces "Use ➕ New Task" is exactly what this must catch.
var replyButtonProse = regexp.MustCompile(
	`(?i)(use\s+the\s+menu|use\s+/start|➕\s*New\s+Task|📅\s*New\s+Event|📝\s*Note\b|menu\s+below)`)

func TestNoScreenPointsAtAReplyButton(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("globbing handlers: %v", err)
	}
	if len(files) < 5 {
		// Positive control: if the glob silently matched nothing, every
		// assertion below would pass while checking no code at all.
		t.Fatalf("found %d handler files — the scan is not looking at the package", len(files))
	}

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if !strings.Contains(line, `"`) {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if m := replyButtonProse.FindString(line); m != "" {
				t.Errorf("%s:%d points the user at a reply button (%q). "+
					"Give the screen a button instead:\n\t%s",
					f, i+1, m, strings.TrimSpace(line))
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bot && go test ./internal/handlers/ -run TestNoScreenPointsAtAReplyButton -v`
Expected: FAIL, naming at least these, which are the violations the spec listed:
- `tasks.go` — `"📋 <b>No tasks</b>\n\nUse ➕ New Task to create one."`
- `calendar.go` — `"Пусто. Создать — <b>📅 New Event</b>."`
- `handler.go` — any surviving `"Use the menu buttons or /start"`

🔴 If it reports **zero** failures, the scan is broken, not the code — check the glob and the `_test.go` filter before going on.

- [ ] **Step 3: Fix the violations with buttons**

Add to `bot/internal/keyboards/menu.go`:

```go
// TaskListEmpty replaces the sentence "Use ➕ New Task to create one."
func TaskListEmpty() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Новая задача", "new_task"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Меню", "main_menu"),
		),
	)
}
```

In `bot/internal/handlers/tasks.go`, change `handleTasks` to take a `messageID`, render through `editOrSend`, and use the keyboard above:

```go
func (h *Handler) handleTasks(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "TODO")
	if err != nil {
		h.editOrSend(chatID, messageID,
			"⚠️ Не дозвонился до сервера. Попробуй через минуту.", keyboards.HomeInline())
		return
	}
	if len(tasks) == 0 {
		h.editOrSend(chatID, messageID, "📋 <b>Задач нет</b>", keyboards.TaskListEmpty())
		return
	}
	// … existing list rendering, ending in editOrSend instead of sendHTMLWithKeyboard
}
```

In `bot/internal/handlers/calendar.go`, `handleCalendarDay` drops the sentence; the button arrives with `DayActions` in Task 5. For now:

```go
	if len(events) == 0 {
		text += "Пусто."
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd bot && go test ./internal/handlers/ -run TestNoScreenPointsAtAReplyButton -v`
Expected: PASS

- [ ] **Step 5: Prove the test still bites**

Add `h.sendText(chatID, "Use ➕ New Task")` anywhere in `tasks.go`. The test must FAIL and name that file and line. Remove it and confirm PASS.

- [ ] **Step 6: Full verification**

Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add bot/internal/handlers/screens_test.go bot/internal/handlers/tasks.go bot/internal/handlers/calendar.go bot/internal/keyboards/menu.go
git commit -m "test(bot): a screen that offers an action carries a button for it, and a test that keeps it so"
```

---

## Task 5: The calendar day — paging, "Today", and no new message

**Files:**
- Modify: `bot/internal/keyboards/keyboards.go` (`MonthGrid`, `DayView` → `DayActions`)
- Modify: `bot/internal/handlers/calendar.go`
- Test: `bot/internal/handlers/calendar_test.go` (extend), `bot/internal/keyboards/keyboards_test.go` (extend)

**Interfaces:**
- Consumes: `editOrSend` (Task 1).
- Produces: `func shiftDay(iso string, days int) (string, bool)`, `keyboards.DayActions(date string, prev, next string, year, month int) tgbotapi.InlineKeyboardMarkup`.

The three real regressions against v0.2.1, each cited by handler line (`_legacy/…/v0.2.1/apps/bot/src/index.mjs`):

| Capability | v0.2.1 | Today |
|---|---|---|
| Day edits the same message | `index.mjs:801` `editMessageText` | `calendar.go` posts a new one |
| ⬅/➡ day paging | handler `calendar_day_` at `index.mjs:750` | absent |
| "Today" jump from the grid | live | absent |

🔴 **Not regressions, do not "restore" them:** `new_event_<date>`, `day_tasks_<date>`, `day_stats_<date>` appear in `keyboards.mjs:288–297` but **no `bot.action` registers them** — they did nothing in v0.2.1 either. The `➕ Событие на этот день` button below is added under the Task 4 rule, not as a restoration.

- [ ] **Step 1: Write the failing test**

```go
package handlers

import "testing"

func TestShiftDayCrossesMonthAndYear(t *testing.T) {
	cases := []struct {
		in    string
		days  int
		want  string
	}{
		{"2026-08-18", 1, "2026-08-19"},
		{"2026-08-31", 1, "2026-09-01"},
		{"2026-09-01", -1, "2026-08-31"},
		{"2026-12-31", 1, "2027-01-01"},
		{"2027-01-01", -1, "2026-12-31"},
		// 2028 is a leap year: the 29th exists and must not be skipped.
		{"2028-02-28", 1, "2028-02-29"},
		{"2028-02-29", 1, "2028-03-01"},
		// 2026 is not: the 29th does not exist and must not be produced.
		{"2026-02-28", 1, "2026-03-01"},
	}
	for _, c := range cases {
		got, ok := shiftDay(c.in, c.days)
		if !ok {
			t.Errorf("shiftDay(%q, %d) refused a valid date", c.in, c.days)
			continue
		}
		if got != c.want {
			t.Errorf("shiftDay(%q, %d) = %q, want %q", c.in, c.days, got, c.want)
		}
	}
}

func TestShiftDayRefusesGarbage(t *testing.T) {
	// The date arrives from callback_data, which is user-reachable input.
	// Refusing is the only safe answer; guessing puts the user on a wrong day.
	for _, bad := range []string{"", "tomorrow", "2026-13-01", "18-08-2026"} {
		if _, ok := shiftDay(bad, 1); ok {
			t.Errorf("shiftDay accepted %q", bad)
		}
	}
}
```

And in `bot/internal/keyboards/keyboards_test.go`:

```go
func TestDayActionsFitTheCallbackBudget(t *testing.T) {
	kb := DayActions("2026-08-18", "2026-08-17", "2026-08-19", 2026, 8)
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == nil {
				continue
			}
			if n := len(*b.CallbackData); n > 64 {
				t.Errorf("callback_data %q is %d bytes, Telegram allows 64", *b.CallbackData, n)
			}
		}
	}
}

func TestMonthGridOffersAJumpToToday(t *testing.T) {
	kb := MonthGrid(2026, 8, "август", make([]string, 42), make([]string, 42), "2026-08-18")
	var found bool
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData != nil && *b.CallbackData == "cal_day_2026-08-18" {
				found = true
			}
		}
	}
	if !found {
		t.Error("the month grid has no jump to today — paging back from December is the only way home")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd bot && go test ./internal/handlers/ -run TestShiftDay -v && go test ./internal/keyboards/ -run 'TestDayActions|TestMonthGrid' -v`
Expected: FAIL — `undefined: shiftDay`, `undefined: DayActions`, and `MonthGrid` takes five arguments, not six.

- [ ] **Step 3: Write the implementation**

In `bot/internal/handlers/calendar.go`:

```go
// shiftDay moves an ISO date by whole days.
//
// Deliberately date-only arithmetic with time.UTC: the value comes from
// callback_data as "2026-08-18" and goes straight back into it. Doing this in a
// zone with DST would give 23- and 25-hour days, and "yesterday" could land on
// today again on the switchover — a bug that appears twice a year and is
// invisible the rest of the time.
func shiftDay(iso string, days int) (string, bool) {
	d, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return "", false
	}
	return d.AddDate(0, 0, days).Format("2006-01-02"), true
}
```

`handleCalendarDay` gains a `messageID`, renders through `editOrSend`, and gets the new keyboard:

```go
func (h *Handler) handleCalendarDay(chatID int64, messageID int, date string) {
	prev, okPrev := shiftDay(date, -1)
	next, okNext := shiftDay(date, 1)
	if !okPrev || !okNext {
		// Unparseable date from callback_data: say so rather than render a day
		// that is not the one asked for.
		h.editOrSend(chatID, messageID, "Не понял дату.", keyboards.HomeInline())
		return
	}
	// … existing loading and text building, with the empty case now just "Пусто."
	h.editOrSend(chatID, messageID, text,
		keyboards.DayActions(date, prev, next, day.Year(), int(day.Month())))
}
```

In `bot/internal/keyboards/keyboards.go`, replace `DayView` with:

```go
// DayActions is the day screen. Paging first, because that is the regression
// people feel: v0.2.1 let you walk day to day from here (handler calendar_day_,
// index.mjs:750) and this bot made you go back to the grid every time.
func DayActions(date, prev, next string, year, month int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅", "cal_day_"+prev),
			tgbotapi.NewInlineKeyboardButtonData("Сегодня", "cal_today"),
			tgbotapi.NewInlineKeyboardButtonData("➡", "cal_day_"+next),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Событие на этот день", "cal_new_"+date),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« К месяцу",
				"cal_back_"+strconv.Itoa(year)+"_"+strconv.Itoa(month)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Меню", "main_menu"),
		),
	)
}
```

`MonthGrid` takes `todayISO` and puts the jump in its bottom row:

```go
func MonthGrid(year, month int, monthName string, labels, dates []string, todayISO string) tgbotapi.InlineKeyboardMarkup {
	// … unchanged header, weekday and week rows …
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📅 Сегодня", "cal_day_"+todayISO),
		tgbotapi.NewInlineKeyboardButtonData("🏠 Меню", "main_menu"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
```

Route the two new prefixes in `HandleCallback`. 🔴 `cal_today` must be matched **before** any `cal_` prefix branch that could swallow it:

```go
	case data == "cal_today":
		h.handleCalendarDay(chatID, cb.Message.MessageID, time.Now().In(h.location()).Format("2006-01-02"))
	case strings.HasPrefix(data, "cal_new_"):
		h.startNewEventForDay(chatID, strings.TrimPrefix(data, "cal_new_"))
	case strings.HasPrefix(data, "cal_day_"):
		h.handleCalendarDay(chatID, cb.Message.MessageID, strings.TrimPrefix(data, "cal_day_"))
```

`startNewEventForDay` seeds the existing event flow with a day, so the user types only a title and time:

```go
// startNewEventForDay begins the normal new-event flow with the day already
// chosen, so "18:00 Ужин" is enough.
func (h *Handler) startNewEventForDay(chatID int64, date string) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_event"
	us.FlowStep = "title"
	us.FlowData["date"] = date
	h.sendHTML(chatID, "📅 <b>Событие на "+format.Escape(date)+"</b>\n\nНапиши время и название — например «18:00 Ужин».")
}
```

In `handleNewEventFlow`, prepend the seeded date to the line before parsing, so the existing parser resolves it with no new code path:

```go
	if d, ok := us.FlowData["date"].(string); ok && d != "" {
		// parse.Parse understands "14.08"; feeding the chosen day in this form
		// reuses the tested path instead of adding a second way to set a date.
		if t, err := time.Parse("2006-01-02", d); err == nil {
			text = t.Format("02.01") + " " + text
		}
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd bot && go test ./internal/handlers/ -run TestShiftDay -v && go test ./internal/keyboards/ -v`
Expected: PASS

- [ ] **Step 5: Prove the tests can fail**

Change `d.AddDate(0, 0, days)` to `d.AddDate(0, 0, days*2)`. `TestShiftDayCrossesMonthAndYear` must FAIL on the first case. Restore. Then remove the `📅 Сегодня` button from `MonthGrid`; `TestMonthGridOffersAJumpToToday` must FAIL. Restore and confirm both PASS.

- [ ] **Step 6: Full verification**

Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add bot/internal/handlers/calendar.go bot/internal/handlers/calendar_test.go bot/internal/handlers/events.go bot/internal/handlers/handler.go bot/internal/keyboards/keyboards.go bot/internal/keyboards/keyboards_test.go
git commit -m "feat(bot): walk day to day in the calendar, jump to today, and stop posting a new message per day"
```

---

## Task 6: `parse.ParseTask` — one line to a task

**Files:**
- Create: `bot/internal/parse/task.go`
- Create: `bot/internal/parse/task_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:

```go
type TaskResult struct {
    Title            string
    Priority         *int       // nil = not stated
    DueDate          *time.Time // nil = not stated
    EstimatedMinutes *int       // nil = not stated
    Tags             []string   // never nil; empty is []
}
func ParseTask(line string, now time.Time) TaskResult
```

Modelled on `bot/internal/parse/event.go`, which already parses "Ужин завтра 19:00" and is tested. Same shape, same rule: **what is not understood stays in the title**.

🔴 **Pointers, not zero values.** `Priority` must distinguish "not stated" from `0`, because `0` is *Buffer* — a real priority. A plain `int` would turn "no priority given" into "lowest urgency" silently. This is also why `api.CreateTaskReq.Priority` becomes `*int` in Task 7.

- [ ] **Step 1: Write the failing test**

```go
package parse

import (
	"testing"
	"time"
)

func now() time.Time { return time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC) }

func TestParseTaskKeepsAPlainLineWhole(t *testing.T) {
	r := ParseTask("позвонить в банк", now())
	if r.Title != "позвонить в банк" {
		t.Errorf("Title = %q, want the whole line", r.Title)
	}
	if r.Priority != nil || r.DueDate != nil || r.EstimatedMinutes != nil {
		t.Error("a plain line invented a field")
	}
	if r.Tags == nil {
		t.Error("Tags is nil; the project's rule is empty slices, never nil")
	}
}

func TestParseTaskReadsPriorityAndKnowsItIsInverted(t *testing.T) {
	high := ParseTask("позвонить !1", now())
	low := ParseTask("позвонить !5", now())
	if high.Priority == nil || low.Priority == nil {
		t.Fatal("priority not parsed")
	}
	// 🔴 1 = Emergency, 5 = If Possible. Lower number, higher urgency.
	if *high.Priority >= *low.Priority {
		t.Errorf("!1 (%d) is not more urgent than !5 (%d) — the inversion was lost",
			*high.Priority, *low.Priority)
	}
	if *high.Priority != 1 {
		t.Errorf("!1 parsed as %d", *high.Priority)
	}
	buffer := ParseTask("почитать !0", now())
	if buffer.Priority == nil || *buffer.Priority != 0 {
		t.Error("!0 (Buffer) was dropped — a nil here would become a default, not Buffer")
	}
}

func TestParseTaskReadsEstimates(t *testing.T) {
	for in, want := range map[string]int{
		"созвон 30м": 30, "созвон 1ч": 60, "созвон 90м": 90, "созвон 2ч": 120,
	} {
		r := ParseTask(in, now())
		if r.EstimatedMinutes == nil {
			t.Errorf("%q: no estimate parsed", in)
			continue
		}
		if *r.EstimatedMinutes != want {
			t.Errorf("%q: %d minutes, want %d", in, *r.EstimatedMinutes, want)
		}
		if r.Title != "созвон" {
			t.Errorf("%q: title left as %q — the marker was not cut out", in, r.Title)
		}
	}
}

func TestParseTaskReadsDueDates(t *testing.T) {
	r := ParseTask("купить билеты завтра", now())
	if r.DueDate == nil {
		t.Fatal("«завтра» did not produce a due date")
	}
	if got := r.DueDate.Format("2006-01-02"); got != "2026-08-19" {
		t.Errorf("due = %s, want 2026-08-19", got)
	}
	if r.Title != "купить билеты" {
		t.Errorf("title = %q, want the day word removed", r.Title)
	}
}

func TestParseTaskReadsCyrillicTags(t *testing.T) {
	// 🔴 Every tag Denis writes is Cyrillic. An ASCII-only \w+ pattern returns
	// an empty list on every one of them and looks like "he uses no tags".
	r := ParseTask("отчёт #работа #срочное", now())
	if len(r.Tags) != 2 {
		t.Fatalf("Tags = %v, want two Cyrillic tags", r.Tags)
	}
	if r.Tags[0] != "работа" || r.Tags[1] != "срочное" {
		t.Errorf("Tags = %v", r.Tags)
	}
	if r.Title != "отчёт" {
		t.Errorf("title = %q, want the tags removed", r.Title)
	}
}

func TestParseTaskLeavesUnknownMarkersInTheTitle(t *testing.T) {
	// 🔴 Cutting a piece of the title out silently is worse than not parsing:
	// the user loses words they typed and never learns why.
	r := ParseTask("позвонить насчёт !срочно", now())
	if r.Priority != nil {
		t.Error("!срочно was read as a priority")
	}
	if r.Title != "позвонить насчёт !срочно" {
		t.Errorf("title = %q — an unrecognised marker was eaten", r.Title)
	}
}

func TestParseTaskHandlesEmptyInput(t *testing.T) {
	r := ParseTask("   ", now())
	if r.Title != "" {
		t.Errorf("Title = %q, want empty", r.Title)
	}
	if r.Tags == nil {
		t.Error("Tags is nil")
	}
}

func TestParseTaskCombinesEverything(t *testing.T) {
	r := ParseTask("позвонить в банк завтра 30м !1 #дела", now())
	if r.Title != "позвонить в банк" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.Priority == nil || *r.Priority != 1 {
		t.Error("priority lost when combined")
	}
	if r.EstimatedMinutes == nil || *r.EstimatedMinutes != 30 {
		t.Error("estimate lost when combined")
	}
	if r.DueDate == nil {
		t.Error("due date lost when combined")
	}
	if len(r.Tags) != 1 || r.Tags[0] != "дела" {
		t.Errorf("Tags = %v", r.Tags)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bot && go test ./internal/parse/ -run TestParseTask -v`
Expected: FAIL — `undefined: ParseTask`

- [ ] **Step 3: Write the implementation**

Create `bot/internal/parse/task.go`:

```go
package parse

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TaskResult is one parsed task line. Every optional field is a pointer so that
// "not stated" and "stated as zero" stay different answers.
//
// 🔴 Priority is the reason this matters. 0 is Buffer, a real priority a user
// can choose, and 5 is "if possible". A plain int would make an unstated
// priority indistinguishable from Buffer, and the API would store the wrong one
// without anything looking broken.
type TaskResult struct {
	Title            string
	Priority         *int
	DueDate          *time.Time
	EstimatedMinutes *int
	Tags             []string
}

var (
	// !0 … !5, standing alone. The trailing boundary is what stops "!срочно"
	// from being read as a priority and having its word eaten.
	priorityRe = regexp.MustCompile(`(?:^|\s)!([0-5])(?:\s|$)`)
	// 30м, 90м, 1ч, 2ч. Minutes and hours, nothing finer — this is a phone.
	estimateRe = regexp.MustCompile(`(?:^|\s)(\d{1,3})\s*(м|мин|ч|час)(?:\s|$)`)
	// #тег. \p{L} rather than \w: every tag here is Cyrillic, and \w is ASCII.
	tagRe = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)
)

// ParseTask reads a line like "позвонить в банк завтра 30м !1 #дела".
//
// `now` is a parameter rather than a clock read so the behaviour is testable
// and so the caller supplies the user's own timezone — same contract as Parse.
//
// 🔴 Whatever is not recognised stays in the title. Silently dropping a word
// the user typed is worse than not parsing it: they lose text and never learn
// why. Only a marker that matched in full is cut out.
func ParseTask(line string, now time.Time) TaskResult {
	res := TaskResult{Tags: []string{}}
	text := strings.TrimSpace(line)
	if text == "" {
		return res
	}

	if m := tagRe.FindAllStringSubmatch(text, -1); m != nil {
		seen := map[string]bool{}
		for _, g := range m {
			tag := strings.ToLower(g[1])
			if !seen[tag] {
				seen[tag] = true
				res.Tags = append(res.Tags, tag)
			}
		}
		text = tagRe.ReplaceAllString(text, " ")
	}

	if m := priorityRe.FindStringSubmatch(text); m != nil {
		p, _ := strconv.Atoi(m[1])
		res.Priority = &p
		text = strings.Replace(text, m[0], " ", 1)
	}

	if m := estimateRe.FindStringSubmatch(text); m != nil {
		n, _ := strconv.Atoi(m[1])
		if strings.HasPrefix(m[2], "ч") {
			n *= 60
		}
		if n > 0 {
			res.EstimatedMinutes = &n
			text = strings.Replace(text, m[0], " ", 1)
		}
	}

	// The day words and explicit dates are already solved for events; reuse
	// that shape rather than growing a second dialect of the same thing.
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lower := strings.ToLower(text)
	for _, rd := range relativeDays {
		if idx := strings.Index(lower, rd.word); idx >= 0 {
			d := day.AddDate(0, 0, rd.days)
			res.DueDate = &d
			text = text[:idx] + text[idx+len(rd.word):]
			break
		}
	}
	if res.DueDate == nil {
		if m := dateRe.FindStringSubmatch(text); m != nil {
			d, _ := strconv.Atoi(m[1])
			mo, _ := strconv.Atoi(m[2])
			year := now.Year()
			if m[3] != "" {
				year, _ = strconv.Atoi(m[3])
			}
			if mo >= 1 && mo <= 12 && d >= 1 && d <= 31 {
				candidate := time.Date(year, time.Month(mo), d, 0, 0, 0, 0, now.Location())
				if m[3] == "" && candidate.Before(day) {
					candidate = candidate.AddDate(1, 0, 0)
				}
				res.DueDate = &candidate
				text = strings.Replace(text, m[0], " ", 1)
			}
		}
	}

	res.Title = cleanTitle(text)
	return res
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd bot && go test ./internal/parse/ -v`
Expected: PASS

- [ ] **Step 5: Prove the two load-bearing tests can fail**

Change `tagRe` to `#(\w+)`. `TestParseTaskReadsCyrillicTags` must FAIL — this is the only proof that the Unicode class is doing work. Restore.
Then change `priorityRe` to `!([0-5])` (no trailing boundary). `TestParseTaskLeavesUnknownMarkersInTheTitle` must FAIL. Restore and confirm PASS.

- [ ] **Step 6: Full verification**

Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add bot/internal/parse/task.go bot/internal/parse/task_test.go
git commit -m "feat(bot): read a task out of one typed line, and leave what it does not understand in the title"
```

---

## Task 7: The task card, and `✅ Создать` available immediately

**Files:**
- Create: `bot/internal/handlers/newtask.go`
- Create: `bot/internal/keyboards/task.go`
- Modify: `bot/internal/api/client.go` (`CreateTaskReq`, `Task`)
- Modify: `bot/internal/handlers/flows.go` (`handleNewTaskFlow` routes to the card)
- Test: `bot/internal/handlers/newtask_test.go`, `bot/internal/keyboards/task_test.go`

**Interfaces:**
- Consumes: `parse.ParseTask` and `parse.TaskResult` (Task 6); `editOrSend` (Task 1).
- Produces: `func taskCardText(r parse.TaskResult, tz string) string`, `keyboards.TaskCard() tgbotapi.InlineKeyboardMarkup`, and the extended request type:

```go
type CreateTaskReq struct {
    Title            string   `json:"title"`
    Priority         *int     `json:"priority,omitempty"`
    Status           string   `json:"status,omitempty"`
    DueDate          *string  `json:"due_date,omitempty"`
    EstimatedMinutes *int     `json:"estimated_minutes,omitempty"`
    Tags             []string `json:"tags,omitempty"`
}
```

🔴 **`Priority` becomes `*int`.** Today it is `int` with no `omitempty`, so every create sends a number — and skipping the priority step would send `0`, which the API stores as **Buffer**, not as "use the default". This is the change that makes "nothing is required" true rather than merely offered.

- [ ] **Step 1: Write the failing test**

```go
package handlers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

func TestTaskCardShowsOnlyWhatWasParsed(t *testing.T) {
	r := parse.ParseTask("позвонить в банк", time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC))
	got := taskCardText(r, "UTC")
	if !strings.Contains(got, "позвонить в банк") {
		t.Errorf("the title is missing:\n%s", got)
	}
	if strings.Contains(got, "📅") || strings.Contains(got, "⏱") {
		t.Errorf("the card invented a due date or an estimate:\n%s", got)
	}
}

func TestTaskCardShowsEverythingThatWasParsed(t *testing.T) {
	r := parse.ParseTask("позвонить в банк завтра 30м !1", time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC))
	got := taskCardText(r, "UTC")
	for _, want := range []string{"📅", "⏱", "30"} {
		if !strings.Contains(got, want) {
			t.Errorf("card is missing %q:\n%s", want, got)
		}
	}
}

func TestTaskCardEscapesTheTitle(t *testing.T) {
	r := parse.ParseTask("R&D <срочно>", time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC))
	got := taskCardText(r, "UTC")
	if strings.Contains(got, "<срочно>") {
		t.Errorf("an unescaped title reached an HTML message:\n%s", got)
	}
}

func TestUnsetPriorityIsAbsentFromTheWire(t *testing.T) {
	// 🔴 The whole point of *int. 0 is Buffer — a real choice. If an unset
	// priority serialised as 0, "skip" would silently mean "lowest urgency".
	body, err := json.Marshal(api.CreateTaskReq{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "priority") {
		t.Errorf("an unset priority was sent anyway: %s", body)
	}
}

func TestBufferPriorityIsSentAsZero(t *testing.T) {
	zero := 0
	body, err := json.Marshal(api.CreateTaskReq{Title: "x", Priority: &zero})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"priority":0`) {
		t.Errorf("Buffer (0) did not reach the wire: %s", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bot && go test ./internal/handlers/ -run 'TestTaskCard|TestUnsetPriority|TestBufferPriority' -v`
Expected: FAIL — `undefined: taskCardText`, and `CreateTaskReq.Priority` is not a `*int`.

- [ ] **Step 3: Extend the API types**

In `bot/internal/api/client.go`:

```go
// CreateTaskReq mirrors api-go's CreateTaskRequest for the fields the bot uses.
//
// 🔴 Every optional field is a pointer with omitempty, and Priority especially:
// 0 is Buffer, a priority a user can pick, so a plain int cannot express "not
// stated". The API treats an absent priority as its own default; it treats 0 as
// Buffer. Those are different tasks.
type CreateTaskReq struct {
	Title            string   `json:"title"`
	Priority         *int     `json:"priority,omitempty"`
	Status           string   `json:"status,omitempty"`
	DueDate          *string  `json:"due_date,omitempty"` // ISO 8601
	EstimatedMinutes *int     `json:"estimated_minutes,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}
```

Add `Tags []string \`json:"tags"\`` to `api.Task`.

Fix the two existing call sites in `bot/internal/handlers/flows.go` — `handleNoteFlow` and `handlePrioritySelect` — to take the address of a local `int` rather than passing a literal.

- [ ] **Step 4: Write the card**

Create `bot/internal/handlers/newtask.go`:

```go
package handlers

import (
	"fmt"
	"strings"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// taskCardText renders what was understood — and nothing else.
//
// A card that shows "Срок: —" for a task with no due date trains the reader to
// skip the line. Absent fields are absent.
func taskCardText(r parse.TaskResult, tz string) string {
	title := r.Title
	if title == "" {
		title = "(без названия)"
	}
	text := "➕ <b>Новая задача</b>\n"
	if r.Priority != nil {
		text += format.PriorityEmoji(*r.Priority) + " "
	}
	text += fmt.Sprintf("<b>%s</b>\n", format.Escape(title))

	var meta []string
	if r.DueDate != nil {
		meta = append(meta, "📅 "+r.DueDate.Format("02.01"))
	}
	if r.EstimatedMinutes != nil {
		meta = append(meta, "⏱ "+format.Duration(*r.EstimatedMinutes))
	}
	if len(r.Tags) > 0 {
		meta = append(meta, "🏷 "+format.Escape(strings.Join(r.Tags, ", ")))
	}
	if len(meta) > 0 {
		text += strings.Join(meta, " · ") + "\n"
	}
	return text
}
```

Create `bot/internal/keyboards/task.go`:

```go
package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// TaskCard is what a parsed line offers.
//
// 🔴 ✅ Создать is FIRST and always enabled. Denis, 18.08: "all of them
// shouldn't be required to create the task". The wizard is an offer underneath
// it, never a gate in front of it.
func TaskCard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Создать", "nt_save"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Подробнее (по шагам)", "nt_wizard"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "main_menu"),
		),
	)
}
```

Rewrite `handleNewTaskFlow` in `bot/internal/handlers/flows.go` so the title step produces a card instead of jumping straight to a priority question:

```go
	case "title":
		r := parse.ParseTask(text, time.Now().In(h.location()))
		us.FlowData["title"] = r.Title
		if r.Priority != nil {
			us.FlowData["priority"] = *r.Priority
		}
		if r.EstimatedMinutes != nil {
			us.FlowData["minutes"] = *r.EstimatedMinutes
		}
		if r.DueDate != nil {
			us.FlowData["due"] = r.DueDate.Format(time.RFC3339)
		}
		if len(r.Tags) > 0 {
			us.FlowData["tags"] = r.Tags
		}
		us.FlowStep = "card"
		h.sendHTMLWithKeyboard(chatID, taskCardText(r, h.cfg.Timezone), keyboards.TaskCard())
```

Handle `nt_save` in `HandleCallback`, building the request from `FlowData` and creating the task.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd bot && go test ./internal/handlers/ -run 'TestTaskCard|TestUnsetPriority|TestBufferPriority' -v`
Expected: PASS

- [ ] **Step 6: Prove the priority test can fail**

Change `Priority *int` back to `Priority int` with the `omitempty` tag and adjust the call sites. `TestBufferPriorityIsSentAsZero` must FAIL — `omitempty` drops a zero int, so Buffer would vanish from the wire. Restore the pointer and confirm both PASS. Record both failures in the report: together they are the evidence that this field needed a pointer rather than a tag.

- [ ] **Step 7: Full verification**

Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: all PASS

- [ ] **Step 8: Commit**

```bash
git add bot/internal/handlers/newtask.go bot/internal/handlers/newtask_test.go bot/internal/handlers/flows.go bot/internal/keyboards/task.go bot/internal/keyboards/task_test.go bot/internal/api/client.go bot/internal/handlers/handler.go
git commit -m "feat(bot): a task is one line and one tap, with the rest offered rather than demanded"
```

---

## Task 8: The `📝 Подробнее` wizard, every step skippable

**Files:**
- Modify: `bot/internal/handlers/newtask.go`
- Modify: `bot/internal/keyboards/task.go`
- Modify: `bot/internal/handlers/handler.go`
- Test: `bot/internal/handlers/wizard_test.go`

**Interfaces:**
- Consumes: `taskCardText`, `keyboards.TaskCard` (Task 7).
- Produces: `func nextWizardStep(current string, has map[string]bool) string`, `keyboards.WizardPriority()`, `keyboards.WizardDue()`, `keyboards.WizardEstimate()`.

- [ ] **Step 1: Write the failing test**

```go
package handlers

import "testing"

func TestWizardVisitsEveryStepInOrder(t *testing.T) {
	has := map[string]bool{}
	order := []string{"priority", "due", "estimate", "done"}
	cur := "start"
	for _, want := range order {
		cur = nextWizardStep(cur, has)
		if cur != want {
			t.Fatalf("after %q the wizard went to %q, want %q", cur, cur, want)
		}
	}
}

func TestWizardSkipsStepsTheLineAlreadyAnswered(t *testing.T) {
	// 🔴 The line "позвонить завтра 30м !1" answered all three. Asking anyway
	// makes the fast path slower than the wizard, which defeats having one.
	has := map[string]bool{"priority": true, "due": true, "estimate": true}
	if got := nextWizardStep("start", has); got != "done" {
		t.Errorf("wizard asked %q about a line that stated everything", got)
	}
}

func TestWizardStopsAtDone(t *testing.T) {
	// A wizard that walks past its last step loops forever on the user's screen.
	if got := nextWizardStep("done", map[string]bool{}); got != "done" {
		t.Errorf("nextWizardStep(done) = %q, want done", got)
	}
}

// Denis, 18.08: nothing is required. Both escapes must exist at every step —
// "⏭ Пропустить" for this field, "✅ Создать сейчас" for all remaining ones.
// This test lives in the keyboards package, so put it in
// bot/internal/keyboards/task_test.go rather than in wizard_test.go.
func TestEveryWizardKeyboardOffersBothEscapes(t *testing.T) {
	kbs := map[string]tgbotapi.InlineKeyboardMarkup{
		"priority": keyboards.WizardPriority(),
		"due":      keyboards.WizardDue(),
		"estimate": keyboards.WizardEstimate(),
	}
	for name, kb := range kbs {
		var skip, save bool
		for _, row := range kb.InlineKeyboard {
			for _, b := range row {
				if b.CallbackData != nil && *b.CallbackData == "nt_skip" {
					skip = true
				}
				if b.CallbackData != nil && *b.CallbackData == "nt_save" {
					save = true
				}
				if b.CallbackData != nil && len(*b.CallbackData) > 64 {
					t.Errorf("%s: callback_data %q is %d bytes", name, *b.CallbackData, len(*b.CallbackData))
				}
			}
		}
		if !skip {
			t.Errorf("%s step has no ⏭ Пропустить — the field became required", name)
		}
		if !save {
			t.Errorf("%s step has no ✅ Создать сейчас — the wizard became a gate", name)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bot && go test ./internal/handlers/ -run TestWizard -v`
Expected: FAIL — `undefined: nextWizardStep`

- [ ] **Step 3: Write the implementation**

```go
// wizardOrder is the sequence of optional fields, coarsest first: a priority is
// a judgement, a date is a commitment, an estimate is a guess.
var wizardOrder = []string{"priority", "due", "estimate"}

// nextWizardStep returns the next field to ask about, skipping anything the
// typed line already answered, and "done" when nothing is left.
func nextWizardStep(current string, has map[string]bool) string {
	start := 0
	if current != "start" && current != "" {
		for i, f := range wizardOrder {
			if f == current {
				start = i + 1
				break
			}
		}
	}
	if current == "done" {
		return "done"
	}
	for i := start; i < len(wizardOrder); i++ {
		if !has[wizardOrder[i]] {
			return wizardOrder[i]
		}
	}
	return "done"
}
```

Keyboards in `bot/internal/keyboards/task.go` — each carries both escapes:

```go
func WizardPriority() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 1", "nt_p_1"),
			tgbotapi.NewInlineKeyboardButtonData("🟠 2", "nt_p_2"),
			tgbotapi.NewInlineKeyboardButtonData("🟡 3", "nt_p_3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 4", "nt_p_4"),
			tgbotapi.NewInlineKeyboardButtonData("⚪ 5", "nt_p_5"),
			tgbotapi.NewInlineKeyboardButtonData("🔵 0", "nt_p_0"),
		),
		wizardEscapes(),
	)
}

func WizardDue() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня", "nt_d_0"),
			tgbotapi.NewInlineKeyboardButtonData("Завтра", "nt_d_1"),
			tgbotapi.NewInlineKeyboardButtonData("Через неделю", "nt_d_7"),
		),
		wizardEscapes(),
	)
}

func WizardEstimate() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("15м", "nt_e_15"),
			tgbotapi.NewInlineKeyboardButtonData("30м", "nt_e_30"),
			tgbotapi.NewInlineKeyboardButtonData("1ч", "nt_e_60"),
			tgbotapi.NewInlineKeyboardButtonData("2ч", "nt_e_120"),
		),
		wizardEscapes(),
	)
}

// wizardEscapes is the row that makes "nothing is required" true. It is one
// function so a new step cannot be added without it.
func wizardEscapes() []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "nt_skip"),
		tgbotapi.NewInlineKeyboardButtonData("✅ Создать сейчас", "nt_save"),
	)
}
```

Route `nt_wizard`, `nt_skip`, `nt_p_*`, `nt_d_*`, `nt_e_*` in `HandleCallback`; each writes into `FlowData` and advances via `nextWizardStep`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd bot && go test ./internal/handlers/ -run TestWizard -v && go test ./internal/keyboards/ -v`
Expected: PASS

- [ ] **Step 5: Prove the escape test can fail**

Delete the `"✅ Создать сейчас"` button from `wizardEscapes`. `TestEveryWizardKeyboardOffersBothEscapes` must FAIL for all three steps. Restore and confirm PASS.

- [ ] **Step 6: Full verification**

Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add bot/internal/handlers/newtask.go bot/internal/handlers/wizard_test.go bot/internal/keyboards/task.go bot/internal/keyboards/task_test.go bot/internal/handlers/handler.go
git commit -m "feat(bot): a step-by-step task wizard where every step can be skipped and the task saved"
```

---

## Task 9: The task card gains Срок, Оценка, Теги

**Files:**
- Modify: `bot/internal/keyboards/keyboards.go` (`TaskActions`)
- Modify: `bot/internal/handlers/tasks.go`
- Test: `bot/internal/keyboards/keyboards_test.go` (extend)

**Interfaces:**
- Consumes: `keyboards.WizardDue`, `keyboards.WizardEstimate` (Task 8) — the same screens, reused.
- Produces: callback prefixes `task_due_`, `task_est_`, `task_tag_`.

- [ ] **Step 1: Write the failing test**

```go
func TestTaskActionsOfferTheOptionalFields(t *testing.T) {
	id := "11111111-2222-3333-4444-555555555555" // 36 chars, a real UUID length
	kb := TaskActions(id)
	want := map[string]bool{
		"task_sched_" + id: false,
		"task_due_" + id:   false,
		"task_est_" + id:   false,
		"task_tag_" + id:   false,
		"task_done_" + id:  false,
	}
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == nil {
				continue
			}
			if n := len(*b.CallbackData); n > 64 {
				t.Errorf("callback_data %q is %d bytes, Telegram allows 64", *b.CallbackData, n)
			}
			if _, ok := want[*b.CallbackData]; ok {
				want[*b.CallbackData] = true
			}
		}
	}
	for data, found := range want {
		if !found {
			t.Errorf("the task card has no button for %q", data)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bot && go test ./internal/keyboards/ -run TestTaskActions -v`
Expected: FAIL — no `task_due_`, `task_est_` or `task_tag_` buttons.

- [ ] **Step 3: Write the implementation**

```go
// TaskActions is the card for one existing task.
//
// Срок / Оценка / Теги reuse the wizard's own step screens rather than
// duplicating them: the question "when is this due" has one right keyboard, and
// two copies of it drift apart the first time one is edited.
//
// Budget check, not arithmetic in prose: keyboards_test.go asserts every
// callback_data here against Telegram's 64 bytes. task_sched_ + a 36-character
// UUID is the longest at 47.
func TaskActions(taskID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ Запланировать", "task_sched_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Срок", "task_due_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData("⏱ Оценка", "task_est_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData("🏷 Теги", "task_tag_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Done", "task_done_"+taskID),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Delete", "task_delete_"+taskID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад", "top_tasks"),
		),
	)
}
```

Route the three prefixes in `HandleCallback`. Each sends the matching wizard keyboard, and its answer issues `PATCH /api/tasks/{id}` through the existing `api.UpdateTask`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd bot && go test ./internal/keyboards/ -v`
Expected: PASS

- [ ] **Step 5: Prove the budget test can fail**

Change the prefix `"task_sched_"` to a 40-character string. `TestTaskActionsOfferTheOptionalFields` must FAIL on the 64-byte assertion. Restore and confirm PASS.

- [ ] **Step 6: Full verification**

Run: `cd bot && go build ./... && go vet ./... && go test ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add bot/internal/keyboards/keyboards.go bot/internal/keyboards/keyboards_test.go bot/internal/handlers/tasks.go bot/internal/handlers/handler.go
git commit -m "feat(bot): set a due date, an estimate and tags from the task card"
```

---

## Task 10: Deploy the dev bot to nl-2 and check the disk

**Files:** none in the repository.

🔴 **This task exists because CI does not deploy the bot.** `.github/workflows/ci.yml`, job `backend`, builds and tests the bot module and deploys **neither** instance. Both live on nl-2 and are updated by hand. On 19.08 four bot features were written, tested and called done while none of them had reached a bot anyone could press.

- [ ] **Step 1: Confirm the working tree is committed and note the ref**

```bash
cd "E:/Projects/007 - Ventures/V003 - NeuroBoost"
git status --short          # expected: empty
git rev-parse --short HEAD  # note this as <ref>
```

- [ ] **Step 2: Back up what is on the server**

```bash
ssh -i ~/.ssh/ufo_servers root@185.214.10.107 \
  'cd /opt/neuroboost-bot && mv src src.bak-2026-08-18'
```

⚠ Only `/opt/neuroboost-bot` (dev). 🔴 Do **not** touch `/opt/neuroboost-bot-prod`, and do not touch anything else on this host — a live Nivium exit node runs beside it.

- [ ] **Step 3: Ship the source**

```bash
git archive --prefix=src/ <ref>:bot | \
  ssh -i ~/.ssh/ufo_servers root@185.214.10.107 'cd /opt/neuroboost-bot && tar x'
```

- [ ] **Step 4: Rebuild the container**

```bash
ssh -i ~/.ssh/ufo_servers root@185.214.10.107 \
  'cd /opt/neuroboost-bot && docker compose up -d --build bot'
```

⚠ `docker compose restart` does not re-read `.env`. Use `up -d --build`, as above.

- [ ] **Step 5: Verify on the disk, not in the test output**

```bash
ssh -i ~/.ssh/ufo_servers root@185.214.10.107 \
  'ls /opt/neuroboost-bot/src/internal/handlers/'
```

Expected to include `menu.go`, `agenda.go`, `newtask.go`, `screens_test.go`. If any is missing, the deploy did not land — say so and stop; do not report the task done.

```bash
ssh -i ~/.ssh/ufo_servers root@185.214.10.107 \
  'docker ps --format "{{.Names}}\t{{.Status}}" | grep neuroboost-dev-bot'
```

Expected: `Up … (healthy)`.

- [ ] **Step 6: Read the log through the filter**

```bash
ssh -i ~/.ssh/ufo_servers root@185.214.10.107 \
  'docker logs --tail 50 neuroboost-dev-bot 2>&1' | sed -E 's/bot[0-9]+:[A-Za-z0-9_-]+/bot<REDACTED>/g'
```

🔴 **Never without the filter.** The library prints `https://api.telegram.org/bot<TOKEN>/getUpdates`, so an unfiltered read puts the token into the transcript.

- [ ] **Step 7: Hand it to Denis for the manual pass**

Report the deployed ref, the `ls` output, and ask him to walk **@NeuroBoost_dev_bot** against `docs/proverka-bota-2026-08-19.md`.

🔴 **This is the acceptance for this plan.** Every part of it is about what a person sees on a screen; `go test` cannot answer "is it better". Do not mark the plan complete on green tests.

---

## Self-Review

**1. Spec coverage**

| Spec requirement | Task |
|---|---|
| Six reply entrances, everything else inline | 2 |
| `🏠 Меню` shows state, degrades without data | 2 |
| `📅 События` defined as the upcoming list | 3 |
| No screen points at a reply button (4 named violations) | 4 |
| Navigation edits the message it was triggered from | 1, then 2–5 use it |
| Message past the 48-hour edit window falls back to a new one | 1 |
| Day paging ⬅/➡ | 5 |
| "Today" jump from the grid | 5 |
| Day opens without a new message | 1 + 5 |
| `➕ Событие на этот день` (under the Task 4 rule, not as a restoration) | 5 |
| Line parsing: day words, `15м/1ч`, `!1..!5/!0`, `#тег` | 6 |
| Unparsed remainder stays in the title | 6 |
| Priority inversion named in code and asserted | 6 |
| `✅ Создать` available immediately | 7 |
| Wizard with `⏭ Пропустить` and `✅ Создать сейчас` on every step | 8 |
| Card gains Срок / Оценка / Теги | 9 |
| `callback_data` budget tested per keyboard | 2, 5, 8, 9 |
| nl-2 deploy as a numbered task | 10 |

No gaps.

**2. Placeholder scan** — one was found and REMOVED, not annotated: Task 8's fourth test first appeared as an empty loop over an anonymous map, with a note to replace it later. A note next to broken code is not a fix; an executor reading top to bottom pastes the first block. Only the working version remains, and it names the file it belongs in (`bot/internal/keyboards/task_test.go`, not `wizard_test.go` — it exercises the keyboards package). `<ref>` and `src.bak-2026-08-18` in Task 10 are values the executor fills from Step 1, not placeholders.

**3. Type consistency**

- `handleTasks(chatID, messageID)` — signature changed in Task 2, used in Task 4. Task 2 says explicitly to add the parameter and ignore it so that task compiles alone.
- `handleAgenda(chatID, messageID)` — referenced in Task 2's routing, defined in Task 3. Task 2 names this.
- `handleCalendarDay(chatID, messageID, date)` — third parameter added in Task 5; Task 4 does not call it.
- `MonthGrid(year, month, monthName, labels, dates, todayISO)` — sixth parameter added in Task 5; `showMonth` in `calendar.go` is its only caller and must be updated in the same task.
- `keyboards.DayView` is replaced by `keyboards.DayActions` in Task 5; no other caller exists.
- `parse.TaskResult` fields are pointers throughout Tasks 6–8; `api.CreateTaskReq` matches them field for field.
- `wizardEscapes()` returns `[]tgbotapi.InlineKeyboardButton`, which is what `NewInlineKeyboardMarkup` takes as a row.
