package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// draftState is everything the confirmation card shows and everything creation
// needs, in one value.
//
// 🔴 One struct, not a handful of keys in FlowData. The task wizard keeps its
// fields as separate map entries (`title`, `priority`, `minutes`, `due`) and
// that has already cost one defect: a save from a wizard step read keys that
// were written under different names and silently did nothing.
type draftState struct {
	// EventID is empty for a draft being created and set for one being
	// edited. It is the ONLY thing separating a POST from a PATCH — and the
	// card, the edit menu and every field screen are shared between the two,
	// on purpose: two editors would be two places to forget a field.
	EventID string

	Title string

	// Description is the event's note. Optional and free text: nothing parses
	// it, the way nothing parses a tag typed on the tags step.
	Description string

	D            parse.Draft
	CalendarID   string
	CalendarName string

	// ReminderOffsets distinguishes three states, and all three are real:
	// nil    — not stated, so the API applies the user's default preset;
	// &[]    — stated as none, so the event stays silent forever;
	// &[n]   — stated.
	// A plain []int would collapse the first two, and the collapse is the
	// defect this product hit on 12.08.
	ReminderOffsets *[]int
}

// Question names a thing the card cannot create without. The empty string
// means nothing is missing.
const (
	askNothing = ""
	askTitle   = "ask:title"
	askFreq    = "ask:freq"
	askDate    = "ask:date"
	askSpan    = "ask:span"
	askTime    = "ask:time"
)

// nextQuestion reports what still has to be answered before an event can be
// created, in the order the user should be asked.
//
// 🔴 A bare «повтор» is a question, not a default. Denis, 15.09: «поскольу
// повтор я не написал частоту, тоже должен уточнить». Creating a one-off
// because no frequency was given would be a silent answer to a question he
// explicitly asked to be asked — and on a repeating event, a wrong guess is
// wrong once a week forever.
func nextQuestion(st draftState) string {
	switch {
	case strings.TrimSpace(st.Title) == "":
		return askTitle
	case st.D.RepeatAsked && st.D.Repeat == "":
		return askFreq
	case !st.D.HasDay:
		return askDate
	case !st.D.EndDay.IsZero() && st.D.EndDay.Before(st.D.Day):
		// 🔴 Denis, 17.09: ⚠ was shown and ✅ created it anyway; then the plain
		// day question offered three buttons and no way to fix a SPAN.
		return askSpan
	case !st.D.HasTime && !st.D.AllDay:
		return askTime
	default:
		return askNothing
	}
}

// weekdayName names the day on the card. Sunday first, because that is where
// Go's time.Weekday starts — the ISO week begins on Monday everywhere else in
// this product, and this array is indexed by the standard library, not by us.
func weekdayName(lang i18n.Lang, w time.Weekday) string {
	ru := [...]string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}
	en := [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	return i18n.T(lang, ru[int(w)], en[int(w)])
}

// repeatName says the whole rule in words: the period, then the end.
// «раз в 3 дня · 10 раз», «каждый год · до 01.12».
func repeatName(lang i18n.Lang, d parse.Draft) string {
	name := freqName(lang, d.Repeat)
	switch {
	case d.RepeatCount > 0:
		name += " · " + fmt.Sprintf(i18n.T(lang, "%d раз", "%d times"), d.RepeatCount)
	case !d.RepeatUntil.IsZero():
		name += " · " + i18n.T(lang, "до ", "until ") + d.RepeatUntil.Format("02.01.2006")
	}
	return name
}

// freqName turns an RRULE into words. The RRULE itself is never translated —
// it is what the server stores.
func freqName(lang i18n.Lang, rrule string) string {
	if n, unit, ok := intervalOf(rrule); ok {
		return intervalName(lang, n, unit)
	}
	switch rrule {
	case "FREQ=DAILY":
		return i18n.T(lang, "каждый день", "every day")
	case "FREQ=WEEKLY":
		return i18n.T(lang, "каждую неделю", "every week")
	case "FREQ=MONTHLY":
		return i18n.T(lang, "каждый месяц", "every month")
	case "FREQ=YEARLY", "FREQ=MONTHLY;INTERVAL=12":
		return i18n.T(lang, "каждый год", "every year")
	}
	return rrule
}

// intervalOf reads «FREQ=X;INTERVAL=N» with N ≥ 2. A year is stored as twelve
// months (parse/repeat.go), so multiples of twelve months are said as years.
func intervalOf(rrule string) (int, string, bool) {
	parts := strings.Split(rrule, ";")
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "FREQ=") || !strings.HasPrefix(parts[1], "INTERVAL=") {
		return 0, "", false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(parts[1], "INTERVAL="))
	if err != nil || n < 2 {
		return 0, "", false
	}
	freq := strings.TrimPrefix(parts[0], "FREQ=")
	if freq == "MONTHLY" && n%12 == 0 {
		if n == 12 {
			return 0, "", false // «каждый год», said by freqName
		}
		return n / 12, "YEARLY", true
	}
	return n, freq, true
}

// intervalName: «раз в 3 дня» / «every 3 days». Each language formats its own
// sentence — Russian needs the plural form, English does not take one.
func intervalName(lang i18n.Lang, n int, freq string) string {
	switch freq {
	case "DAILY":
		return i18n.T(lang, fmt.Sprintf("раз в %d %s", n, ruPlural(n, "день", "дня", "дней")), fmt.Sprintf("every %d days", n))
	case "WEEKLY":
		return i18n.T(lang, fmt.Sprintf("раз в %d %s", n, ruPlural(n, "неделю", "недели", "недель")), fmt.Sprintf("every %d weeks", n))
	case "MONTHLY":
		return i18n.T(lang, fmt.Sprintf("раз в %d %s", n, ruPlural(n, "месяц", "месяца", "месяцев")), fmt.Sprintf("every %d months", n))
	default:
		return i18n.T(lang, fmt.Sprintf("раз в %d %s", n, ruPlural(n, "год", "года", "лет")), fmt.Sprintf("every %d years", n))
	}
}

// ruPlural picks the Russian form for a count: 1 день, 3 дня, 5 дней.
func ruPlural(n int, one, few, many string) string {
	switch {
	case n%10 == 1 && n%100 != 11:
		return one
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
		return few
	default:
		return many
	}
}

// colourName turns a palette name into words. The palette name is what the web
// paints from, so it is stored and never translated.
func colourName(lang i18n.Lang, palette string) string {
	switch palette {
	case "blue":
		return i18n.T(lang, "синий", "blue")
	case "violet":
		return i18n.T(lang, "фиолетовый", "violet")
	case "green":
		return i18n.T(lang, "зелёный", "green")
	case "red":
		return i18n.T(lang, "красный", "red")
	case "amber":
		return i18n.T(lang, "янтарный", "amber")
	case "cyan":
		return i18n.T(lang, "голубой", "cyan")
	case "pink":
		return i18n.T(lang, "розовый", "pink")
	case "slate":
		return i18n.T(lang, "серый", "slate")
	}
	return palette
}

// renderDraft writes the card.
//
// It is a pure function of the state and the clock so it can be tested without
// a Telegram server — the same reason the parser takes `now` as a parameter.
//
// 🔴 Missing values are printed as questions, not omitted. A card that simply
// leaves out the date reads as "no date needed"; «⚠ дата не указана — спрошу»
// reads as what it is. This product has already shipped one defect whose whole
// shape was "absent looks the same as empty".
func renderDraft(lang i18n.Lang, st draftState, now time.Time) string {
	var b strings.Builder

	// 🔴 Date, then time, then title. Denis, 15.09: «Плохо что сначала
	// показывается навание потом дата потом время, должно быть дата потом
	// время потом название, чтобы было проще сориентироваться».
	//
	// It matters most in a list. Seventeen entries scanned down the left edge
	// answer "when is this" first, which is the question you are actually
	// asking when you read back a timetable; the title is the part you already
	// know, because you just typed it.
	if st.D.HasDay && !st.D.EndDay.IsZero() {
		// «🗓 14.09 – 29.09 · 16 дней» — a span says both ends and its length.
		days := int(st.D.EndDay.Sub(st.D.Day).Round(24*time.Hour)/(24*time.Hour)) + 1
		fmt.Fprintf(&b, "🗓 %s – %s · %s%s\n",
			st.D.Day.Format("02.01"), st.D.EndDay.Format("02.01"),
			i18n.T(lang, fmt.Sprintf("%d %s", days, ruPlural(days, "день", "дня", "дней")), fmt.Sprintf("%d days", days)),
			checkMark(lang, st.D.IsUncertain(parse.FieldDay)))
	} else if st.D.HasDay {
		fmt.Fprintf(&b, "🗓 %s, %d %s%s\n",
			weekdayName(lang, st.D.Day.Weekday()), st.D.Day.Day(), monthGenitive(lang, st.D.Day.Month()),
			checkMark(lang, st.D.IsUncertain(parse.FieldDay)))
	} else {
		b.WriteString(i18n.T(lang, "⚠ дата не указана — спрошу\n", "⚠ no date — I will ask\n"))
	}

	switch {
	case st.D.AllDay:
		b.WriteString(i18n.T(lang, "🕐 весь день\n", "🕐 all day\n"))
	case st.D.HasTime && st.D.HasEnd:
		fmt.Fprintf(&b, "🕐 %s–%s%s\n", offsetHHMM(st.D.Start), offsetHHMM(st.D.End),
			checkMark(lang, st.D.IsUncertain(parse.FieldTime)))
	case st.D.HasTime:
		fmt.Fprintf(&b, "🕐 %s%s\n", offsetHHMM(st.D.Start),
			checkMark(lang, st.D.IsUncertain(parse.FieldTime)))
	default:
		b.WriteString(i18n.T(lang, "⚠ время не указано — спрошу\n", "⚠ no time — I will ask\n"))
	}

	// 💬 for the name of the thing. Denis, 18.09: «надо поменять иконки, у
	// названия например 💬». The KIND (event or task) is said by the header
	// line and by the «✅ и задача» line below, so the title no longer has to
	// carry two meanings in one glyph.
	icon := "💬"
	title := st.Title
	if strings.TrimSpace(title) == "" {
		title = i18n.T(lang, "без названия", "untitled")
	}
	fmt.Fprintf(&b, "%s <b>%s</b>\n", icon, format.Escape(title))
	if st.D.IsTask {
		// 🔴 Denis, 17.09: «карточка одна, но теперь непонятно, что ещё и
		// задача была создана». The ✅ icon alone did not say it.
		b.WriteString(i18n.T(lang,
			"✅ и задача — её можно отметить выполненной\n",
			"✅ plus a task — it can be ticked off\n"))
	}

	// 🔴 From here down every characteristic is printed, filled or not. Denis,
	// 18.09: «в базовых карточках должна быть вся информация, просто говоря
	// "нет" для характеристик без значения, чтобы люди знали, что они
	// существуют».
	//
	// The old card printed a field only when it had a value, which meant the
	// bot taught its own feature set by accident: whatever you had never typed,
	// you had no way to learn about. It is also why the difference between a
	// task and an event stayed unclear — the card never showed the fields the
	// two differ in.
	switch {
	case st.D.Repeat != "":
		b.WriteString(fieldLine("🔁", i18n.T(lang, "Повтор:", "Repeat:"), format.Escape(repeatName(lang, st.D))))
	case st.D.RepeatAsked:
		// Stronger than «нет»: a question that is about to be asked is not an
		// absence, and printing «нет» here would answer it silently.
		b.WriteString(i18n.T(lang,
			"⚠ повтор — частота не указана, спрошу\n",
			"⚠ repeats — no frequency given, I will ask\n"))
	default:
		b.WriteString(fieldLine("🔁", i18n.T(lang, "Повтор:", "Repeat:"), noneWord(lang)))
	}

	b.WriteString(fieldLine("📁", i18n.T(lang, "Календарь:", "Calendar:"),
		orNone(lang, format.Escape(st.CalendarName))))

	// 🔴 Three states, three sentences, and none of them is «нет». Saying
	// nothing for the nil case used to be correct-but-invisible: it means "my
	// usual reminders", which is what the server will do — while «не
	// напоминать» means silence forever. Collapsing either into «нет» would
	// merge two different futures into one word.
	switch {
	case st.ReminderOffsets == nil:
		b.WriteString(fieldLine("🔔", i18n.T(lang, "Напоминания:", "Reminders:"),
			i18n.T(lang, "как обычно", "as usual")))
	case len(*st.ReminderOffsets) == 0:
		b.WriteString(fieldLine("🔕", i18n.T(lang, "Напоминания:", "Reminders:"),
			i18n.T(lang, "без напоминаний", "none")))
	default:
		parts := make([]string, 0, len(*st.ReminderOffsets))
		for _, m := range *st.ReminderOffsets {
			parts = append(parts, humanOffset(lang, m))
		}
		joined := strings.Join(parts, ", ")
		b.WriteString(fieldLine("🔔", i18n.T(lang, "Напоминания:", "Reminders:"),
			i18n.T(lang, "за "+joined, joined+" before")))
	}

	b.WriteString(fieldLine("🏷", i18n.T(lang, "Теги:", "Tags:"),
		orNone(lang, format.Escape(strings.Join(st.D.Tags, ", ")))))
	b.WriteString(fieldLine("🎨", i18n.T(lang, "Цвет:", "Colour:"),
		orNone(lang, format.Escape(colourName(lang, st.D.Colour)))))
	b.WriteString(fieldLine("📄", i18n.T(lang, "Описание:", "Description:"),
		orNone(lang, format.Escape(shorten(st.Description, descriptionOnCard)))))

	return strings.TrimRight(b.String(), "\n")
}

// fieldLine prints one characteristic of a card, ALWAYS.
//
// No column padding: Telegram renders HTML in a proportional font, so spaces
// would align nothing and only look ragged on a narrow phone.
func fieldLine(icon, label, value string) string {
	return fmt.Sprintf("%s %s %s\n", icon, label, value)
}

// noneWord is the word an empty characteristic is printed with.
func noneWord(lang i18n.Lang) string {
	return i18n.T(lang, "нет", "none")
}

// orNone renders an empty value as a word rather than as a blank.
//
// 🔴 A blank after a label reads as a bug in the bot; «нет» reads as an answer.
// The difference matters because the whole point of printing empty fields is to
// teach that the field exists.
func orNone(lang i18n.Lang, value string) string {
	if strings.TrimSpace(value) == "" {
		return noneWord(lang)
	}
	return value
}

// descriptionOnCard is how much of a note the card shows.
//
// 🔴 The card is a summary, not the document. A description pasted out of a
// letter would push the buttons off a phone screen — and a confirmation card
// whose confirm button is below the fold has stopped being one.
const descriptionOnCard = 60

// shorten flattens a description to ONE line, cuts it at a word boundary, and
// says it cut with «…».
//
// 🔴 Three separate requirements, each learned the hard way on 18.09:
//
//   - one line — the first version counted runes of the whole string, so four
//     short lines came to 55 characters and were printed in full, turning a
//     nine-line card into a thirteen-line one;
//   - a word boundary — «обязательно если есть возможность чтобы оно не
//     обрывалось на середине слова»;
//   - the ellipsis — without it a cut description is indistinguishable from a
//     complete one, and someone edits the half they were shown believing it is
//     all of it.
func shorten(s string, limit int) string {
	flat := strings.Join(strings.Fields(s), " ")
	r := []rune(flat)
	if len(r) <= limit {
		return flat
	}
	cut := string(r[:limit])
	// Back up to the last space, unless that would leave almost nothing — a
	// single very long word has no boundary to find, and half of it is more
	// useful than none of it.
	if i := strings.LastIndex(cut, " "); i > limit/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,;:—-") + "…"
}

// checkMark flags a value the parser read from a loose format.
//
// 🔴 Denis, 15.09: «если время написано не строго по формулировке должно
// помечать на уточнение/подтверждение». The bot now reads «13;00», «12» and
// «12:00 -12:50», which is what he asked for — but reading a sloppy format
// silently turns a typo into a fact nobody checked. The mark is the difference
// between a kindness and a guess.
func checkMark(lang i18n.Lang, uncertain bool) string {
	if uncertain {
		return i18n.T(lang, " ⚠ проверь", " ⚠ check this")
	}
	return ""
}

// humanOffset turns minutes-before into the words a person would use.
func humanOffset(lang i18n.Lang, m int) string {
	switch {
	case m >= 1440 && m%1440 == 0:
		return fmt.Sprintf(i18n.T(lang, "%d дн.", "%dd"), m/1440)
	case m >= 60 && m%60 == 0:
		return fmt.Sprintf(i18n.T(lang, "%d ч.", "%dh"), m/60)
	default:
		return fmt.Sprintf(i18n.T(lang, "%d мин.", "%dm"), m)
	}
}

// offsetHHMM prints an offset from midnight as a clock time. An offset past
// 24 hours belongs to an end that crossed midnight and wraps back round.
func offsetHHMM(d time.Duration) string {
	d %= 24 * time.Hour
	return fmt.Sprintf("%02d:%02d", int(d/time.Hour), int(d%time.Hour/time.Minute))
}

// splitTags reads the raw text of the «изменить → теги» step.
//
// 🔴 No recogniser runs on it. Denis, 15.09: «если теги, то там через запятую
// или как-то тоже даже если тег это полдень то это тег, а уже не время». So
// «полдень» typed here is the tag «полдень», and the fact that the word also
// names a time is irrelevant on this branch.
func splitTags(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	seen := map[string]bool{}
	tags := make([]string, 0, len(fields))
	for _, f := range fields {
		tag := strings.ToLower(strings.Trim(strings.TrimSpace(f), "#"))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	return tags
}
