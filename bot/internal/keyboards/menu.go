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
			tgbotapi.NewKeyboardButton(menuRows[0][0].Label),
			tgbotapi.NewKeyboardButton(menuRows[0][1].Label),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(menuRows[1][0].Label),
			tgbotapi.NewKeyboardButton(menuRows[1][1].Label),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(menuRows[2][0].Label),
			tgbotapi.NewKeyboardButton(menuRows[2][1].Label),
		),
	)
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
	Label  string
	Screen string
}

// menuRows is the single declaration of the reply keyboard: MainMenu builds
// the buttons from it and MenuScreen answers from it.
//
// 🔴 Derived, not retyped. The guard in package handlers that interrupts a
// running flow asks MenuScreen whether a message is a button press. If that
// set were a second hand-kept list, a seventh button would one day be added to
// the keyboard and guarded by nobody — and the symptom would be Denis's: the
// label becomes the title of the event he was creating.
var menuRows = [3][2]menuEntrance{
	{{"🏠 Меню", ScreenMenu}, {"🗓 Календарь", ScreenCalendar}},
	{{"📅 События", ScreenAgenda}, {"📋 Задачи", ScreenTasks}},
	{{"➕ Создать", ScreenCreate}, {"⚙️ Настройки", ScreenSettings}},
}

// MenuScreen reports which screen a reply-keyboard label opens.
//
// Matching is exact. A message that merely CONTAINS a label is not a press —
// «напомнить про 📋 Задачи» is a note someone typed, not a button.
func MenuScreen(text string) (string, bool) {
	for _, row := range menuRows {
		for _, e := range row {
			if e.Label == text {
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
func HomeInline() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗓 Календарь", "cal_open"),
			tgbotapi.NewInlineKeyboardButtonData("📅 События", "agenda_open"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Задачи", "top_tasks"),
			tgbotapi.NewInlineKeyboardButtonData("➕ Создать", "create_menu"),
		),
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

// RestartNewEvent replaces the sentence "Начни заново: 📅 New Event" — the
// flow's own state was lost (a stale button press after a bot restart), and
// the recovery is a button that starts a fresh new-event flow, not prose
// telling the user which reply button to go press.
func RestartNewEvent() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Новое событие", "new_event"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Меню", "main_menu"),
		),
	)
}

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
