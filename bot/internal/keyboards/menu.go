package keyboards

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// MainMenu is the reply keyboard, and it holds ENTRANCES ONLY.
//
// Denis, 18.08: six buttons — menu, calendar, events, tasks, create, settings —
// and "each of keyboard buttons open inline menus". The previous keyboard mixed
// entrances with actions (Note, New Task, New Event, Planning, Stats), and that
// is the mechanical reason inline screens kept saying "Use ➕ New Task" instead
// of offering a button: the action lived somewhere a screen could only point at.
func MainMenu(lang i18n.Lang) tgbotapi.ReplyKeyboardMarkup {
	rows := make([]([]tgbotapi.KeyboardButton), 0, len(menuRows))
	for _, row := range menuRows {
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(row[0].label(lang)),
			tgbotapi.NewKeyboardButton(row[1].label(lang)),
		))
	}
	return tgbotapi.NewReplyKeyboard(rows...)
}

// Screen names. A press of a reply button opens one of these, and — since
// 15.09 — abandons whatever flow was running first.
const (
	ScreenMenu     = "menu"
	ScreenCalendar = "calendar"
	ScreenAgenda   = "agenda"
	ScreenTasks    = "tasks"
	ScreenCreate   = "create"
	ScreenSettings = "settings"
)

type menuEntrance struct {
	RU     string
	EN     string
	Screen string
	// What the entrance is, one line, for the menu tour (MenuTour). Declared
	// here so a seventh button cannot arrive without saying what it is.
	WhatRU string
	WhatEN string
}

func (e menuEntrance) label(lang i18n.Lang) string { return i18n.T(lang, e.RU, e.EN) }

// menuRows is the single declaration of the reply keyboard: MainMenu builds
// the buttons from it and MenuScreen answers from it.
//
// 🔴 Derived, not retyped. The guard in package handlers that interrupts a
// running flow asks MenuScreen whether a message is a button press. If that
// set were a second hand-kept list, a seventh button would one day be added to
// the keyboard and guarded by nobody — and the symptom would be Denis's: the
// label becomes the title of the event he was creating.
var menuRows = [3][2]menuEntrance{
	{
		{"🏠 Меню", "🏠 Menu", ScreenMenu, "главный экран, отсюда всё остальное", "the home screen, everything else starts here"},
		{"🗓 Календарь", "🗓 Calendar", ScreenCalendar, "месяц сеткой, день по нажатию", "the month as a grid, a day on tap"},
	},
	{
		{"📅 События", "📅 Events", ScreenAgenda, "ближайшие события списком", "upcoming events as a list"},
		{"📋 Задачи", "📋 Tasks", ScreenTasks, "список задач: отметить, запланировать, изменить", "your tasks: tick, schedule, edit"},
	},
	{
		{"➕ Создать", "➕ Create", ScreenCreate, "задача, событие или заметка кнопками", "a task, an event or a note, with buttons"},
		{"⚙️ Настройки", "⚙️ Settings", ScreenSettings, "язык, часовой пояс, напоминания, задачи дня", "language, time zone, reminders, day tasks"},
	},
}

// MenuScreen reports which screen a reply-keyboard label opens.
//
// 🔴 It answers for BOTH languages, always, and that is not laziness — it is the
// only correct answer. A reply keyboard lives on the phone, not on the server:
// after switching the language, the OLD keyboard stays on screen until the next
// one is sent. A press of it must still work, or the language switch breaks
// every entrance until the user finds a way to make the bot resend the keyboard.
//
// Matching is exact. A message that merely CONTAINS a label is not a press —
// «напомнить про 📋 Задачи» is a note someone typed, not a button.
func MenuScreen(text string) (string, bool) {
	for _, row := range menuRows {
		for _, e := range row {
			if e.RU == text || e.EN == text {
				return e.Screen, true
			}
		}
	}
	return "", false
}

// HomeInline is the home screen, and it must reach EVERYWHERE the reply
// keyboard reaches.
//
// 🔴 Denis, 23.08: «странно что в меню обычном нету тех же кнопок или невозможно
// добраться до кнопок, которые есть в клавиатурном меню». Календарь, События and
// Задачи lived only on the reply keyboard — and a reply keyboard can be
// collapsed, at which point three of the six entrances were unreachable.
//
// The rule this encodes, held by TestEveryReplyEntranceHasAnInlineWayIn: every
// reply-keyboard button has an inline button leading to the same screen. 🏠 Меню
// is the exception because it IS this screen.
// HomeInlineFor is the home keyboard with or without «📌 Задачи дня»: switched
// off, day tasks leave no button behind (spec 2026-09-22 §11).
func HomeInlineFor(lang i18n.Lang, dayTasks bool) tgbotapi.InlineKeyboardMarkup {
	last := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⚙️ Настройки", "⚙️ Settings"), "settings_menu"))
	if dayTasks {
		last = tgbotapi.NewInlineKeyboardRow(
			// «Задачи дня» (spec 2026-09-22 §8): «today» resolves in the user's zone.
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📌 Задачи дня", "📌 Day tasks"), "dt_d_today"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⚙️ Настройки", "⚙️ Settings"), "settings_menu"))
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗓 Календарь", "🗓 Calendar"), "cal_open"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📅 События", "📅 Events"), "agenda_open"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📋 Задачи", "📋 Tasks"), "top_tasks"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "➕ Создать", "➕ Create"), "create_menu"),
		),
		// 🗂 Планирование is hidden until tasks can be placed into free slots
		// (Denis, 17.09): the first outside user could not tell what it was for,
		// and Denis's own answer was «not ready yet». The «planning» callback
		// still answers, so a button on an old message keeps working.
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🎯 Сегодня", "🎯 Today"), "today_focus"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📊 Статистика", "📊 Stats"), "stats"),
		),
		last,
	)
}

// CreateMenu is what ➕ Создать offers. 📝 Заметка still writes a task today;
// the notes entity is a separate spec
// (specs/2026-08-18-notes-as-their-own-entity-design.md) and this button moves
// to it unchanged when that lands.
func CreateMenu(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📋 Задача", "📋 Task"), "new_task"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📅 Событие", "📅 Event"), "new_event"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📝 Заметка", "📝 Note"), "new_note"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🏠 Меню", "🏠 Menu"), "main_menu"),
		),
	)
}

// TaskListEmpty replaces the sentence "Use ➕ New Task to create one."
func TaskListEmpty(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "➕ Новая задача", "➕ New task"), "new_task"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🏠 Меню", "🏠 Menu"), "main_menu"),
		),
	)
}

// RestartNewEvent replaces the sentence "Начни заново: 📅 New Event" — the
// flow's own state was lost (a stale button press after a bot restart), and
// the recovery is a button that starts a fresh new-event flow, not prose
// telling the user which reply button to go press.
func RestartNewEvent(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📅 Новое событие", "📅 New event"), "new_event"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🏠 Меню", "🏠 Menu"), "main_menu"),
		),
	)
}

// AgendaActions sits under 📅 События. The screen offers what can be done from
// it rather than naming a reply button.
func AgendaActions(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "➕ Событие", "➕ Event"), "new_event"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Изменить", "✏️ Edit"), "event_pick"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗓 Календарь", "🗓 Calendar"), "cal_open"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🏠 Меню", "🏠 Menu"), "main_menu"),
		),
	)
}

// Menu tour callbacks (N1, pass 3): MenuTourOpen shows the tour, and each of
// its buttons is MenuTourPrefix + the screen it opens.
const (
	MenuTourOpen   = "menu_tour"
	MenuTourPrefix = "mo_"
)

// MenuTour explains the reply keyboard: a line per entrance, and a button per
// entrance that opens it, built from menuRows so it cannot drift from the menu.
func MenuTour(lang i18n.Lang) (string, tgbotapi.InlineKeyboardMarkup) {
	var b strings.Builder
	b.WriteString(i18n.T(lang, "<b>Что в меню</b>\n", "<b>What is in the menu</b>\n"))
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, row := range menuRows {
		var buttons []tgbotapi.InlineKeyboardButton
		for _, e := range row {
			fmt.Fprintf(&b, "\n%s: %s", e.label(lang), i18n.T(lang, e.WhatRU, e.WhatEN))
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(e.label(lang), MenuTourPrefix+e.Screen))
		}
		rows = append(rows, buttons)
	}
	b.WriteString(i18n.T(lang, "\n\nНажми, чтобы открыть.", "\n\nTap one to open it."))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Назад", "« Back"), "help_x")))
	return b.String(), tgbotapi.NewInlineKeyboardMarkup(rows...)
}
