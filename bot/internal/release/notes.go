// Package release carries what «Что нового» shows.
//
// 🔴 The text lives in the bot's own source, not in docs/relizy/. The bot runs
// from a copied src/ directory on nl-2 with no repository beside it (gotcha
// 19 — neither bot is deployed by CI), so a file that has to be remembered
// separately at deploy time is a file that will be forgotten. Then the one
// screen whose whole job is to say what changed says nothing, and nobody
// notices, because an empty screen looks like a quiet release.
package release

// Note is one release, in both languages.
type Note struct {
	Version string
	// Released is when this version reached the bot people actually use, in the
	// bot's own words. Denis, 18.09: «можно кроме версии, еще и дату и время
	// релиза писать» — a version number alone does not answer "is this the one
	// I already saw".
	Released string
	RU       string
	EN       string
	// OfferPriority puts «🔘 Выбрать символ приоритета» under this release's
	// broadcast. Once, for the release that made the symbol a choice.
	OfferPriority bool
}

// notes is newest first. A handful of entries is enough: «что нового» answers
// "since I last looked", not "since the beginning".
//
// ⚠ Each entry says WHERE it actually is. A version listed here that has not
// reached production reads as «you have this» to somebody who does not — which
// is precisely the confusion of 18.09, when a night of work sat on dev and the
// report said «выкачено».
var notes = []Note{
	{
		Version:  "v0.4.11.4",
		Released: "22.09.2026 · на dev, в проде ещё нет",
		// The priority symbol is a choice from this release on; people past
		// onboarding are offered it under this release's broadcast (spec 21.09 §B2).
		OfferPriority: true,
		RU: "• 🔁 Повторяющиеся задачи: создать словом или кнопкой, отметить день, отложить, выключить повтор\n" +
			"• 📅 Задачу — в календарь, событие — в задачу: связать или перенести, и заранее видно, что чем станет\n" +
			"• 📊 Статистика — экран с кнопками: неделя, месяц, год, всё время; занятость по часам, задачи, рефлексии\n" +
			"• 🗓 В месячном календаре видно, насколько заполнен каждый день\n" +
			"• 🔔 «Долбить»: напоминание повторяется, пока не ответишь\n" +
			"• 🔘 Символ приоритета на выбор — кружки, точки или тире\n" +
			"• ℹ️ «Что это?» на новых экранах\n" +
			"• 🔗 Аккаунт на сайте прямо из бота: ссылка для входа или код привязки",
		EN: "• 🔁 Repeating tasks: create by word or button, tick a day, postpone, switch the repeat off\n" +
			"• 📅 A task onto the calendar, an event into a task: link or move, and see first what becomes what\n" +
			"• 📊 Statistics is a screen with buttons: week, month, year, all time; busy hours, tasks, reflections\n" +
			"• 🗓 The month calendar shows how full each day is\n" +
			"• 🔔 «Nag»: a reminder repeats until you answer it\n" +
			"• 🔘 Choose the priority symbol — circles, dots or dashes\n" +
			"• ℹ️ «What is this?» on the new screens\n" +
			"• 🔗 Your website account from inside the bot: a sign-in link or a linking code",
	},
	{
		Version:  "v0.4.11.3",
		Released: "21.09.2026 · в проде",
		RU: "• Карточка называет все поля — повтор, календарь, теги, цвет — и пишет «нет» там, где пусто\n" +
			"• Календари прямо в боте: создать, переименовать, цвет, участники, выйти\n" +
			"• Пригласить можно ссылкой — она работает и для тех, у кого нет email\n" +
			"• Напоминание откладывается на 10 минут или на час\n" +
			"• Появились «сообщить об ошибке» и «предложить улучшение»",
		EN: "• Cards name every field — repeat, calendar, tags, colour — and say «none» where empty\n" +
			"• Calendars inside the bot: create, rename, recolour, members, leave\n" +
			"• Invite by link — it works for people with no email\n" +
			"• Snooze a reminder by ten minutes or an hour\n" +
			"• Report a bug and suggest a feature",
	},
	{
		Version:  "v0.4.11.2",
		Released: "17.09.2026 23:40",
		RU: "• Первая настройка: язык и часовой пояс за три шага\n" +
			"• Можно просто написать боту — он поймёт, что создать\n" +
			"• Свой период повтора, события на несколько дней, напоминания галочками",
		EN: "• First-run setup: language and time zone in three steps\n" +
			"• Just write to the bot — it works out what to create\n" +
			"• Custom repeat intervals, multi-day events, reminders as checkboxes",
	},
	{
		Version:  "v0.4.11.1",
		Released: "16.09.2026 21:10",
		RU: "• Событие строкой обычным языком и карточка подтверждения\n" +
			"• Списки дел одним сообщением\n" +
			"• Язык интерфейса: русский и английский",
		EN: "• Create an event by writing a plain sentence, with a confirmation card\n" +
			"• Whole to-do lists in one message\n" +
			"• Interface language: Russian and English",
	},
}

// Latest is the newest release.
func Latest() Note { return notes[0] }

// All returns every release, newest first.
func All() []Note { return notes }
