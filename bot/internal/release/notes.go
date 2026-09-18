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
}

// notes is newest first. Three entries is enough: «что нового» answers "since I
// last looked", not "since the beginning".
var notes = []Note{
	{
		Version:  "v0.4.11.3",
		Released: "18.09.2026 03:00",
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
