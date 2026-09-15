package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
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
	Title        string
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
	case !st.D.HasTime && !st.D.AllDay:
		return askTime
	default:
		return askNothing
	}
}

var weekdayRu = [...]string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}

var freqRu = map[string]string{
	"FREQ=DAILY":   "каждый день",
	"FREQ=WEEKLY":  "каждую неделю",
	"FREQ=MONTHLY": "каждый месяц",
	"FREQ=YEARLY":  "каждый год",
}

var colourRu = map[string]string{
	"blue": "синий", "violet": "фиолетовый", "green": "зелёный", "red": "красный",
	"amber": "янтарный", "cyan": "голубой", "pink": "розовый", "slate": "серый",
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
func renderDraft(st draftState, now time.Time) string {
	var b strings.Builder

	// 🔴 Date, then time, then title. Denis, 15.09: «Плохо что сначала
	// показывается навание потом дата потом время, должно быть дата потом
	// время потом название, чтобы было проще сориентироваться».
	//
	// It matters most in a list. Seventeen entries scanned down the left edge
	// answer "when is this" first, which is the question you are actually
	// asking when you read back a timetable; the title is the part you already
	// know, because you just typed it.
	if st.D.HasDay {
		fmt.Fprintf(&b, "🗓 %s, %d %s%s\n",
			weekdayRu[int(st.D.Day.Weekday())], st.D.Day.Day(), monthGenitive(st.D.Day.Month()),
			checkMark(st.D.IsUncertain(parse.FieldDay)))
	} else {
		b.WriteString("⚠ дата не указана — спрошу\n")
	}

	switch {
	case st.D.AllDay:
		b.WriteString("🕐 весь день\n")
	case st.D.HasTime && st.D.HasEnd:
		fmt.Fprintf(&b, "🕐 %s–%s%s\n", offsetHHMM(st.D.Start), offsetHHMM(st.D.End),
			checkMark(st.D.IsUncertain(parse.FieldTime)))
	case st.D.HasTime:
		fmt.Fprintf(&b, "🕐 %s%s\n", offsetHHMM(st.D.Start),
			checkMark(st.D.IsUncertain(parse.FieldTime)))
	default:
		b.WriteString("⚠ время не указано — спрошу\n")
	}

	icon := "📅"
	if st.D.IsTask {
		icon = "✅"
	}
	title := st.Title
	if strings.TrimSpace(title) == "" {
		title = "без названия"
	}
	fmt.Fprintf(&b, "%s <b>%s</b>\n", icon, format.Escape(title))

	switch {
	case st.D.Repeat != "":
		name, ok := freqRu[st.D.Repeat]
		if !ok {
			name = st.D.Repeat
		}
		fmt.Fprintf(&b, "🔁 %s\n", format.Escape(name))
	case st.D.RepeatAsked:
		b.WriteString("⚠ повтор — частота не указана, спрошу\n")
	}

	if st.D.Colour != "" {
		name, ok := colourRu[st.D.Colour]
		if !ok {
			name = st.D.Colour
		}
		fmt.Fprintf(&b, "🎨 %s\n", format.Escape(name))
	}
	if st.CalendarName != "" {
		fmt.Fprintf(&b, "📁 %s\n", format.Escape(st.CalendarName))
	}
	if len(st.D.Tags) > 0 {
		fmt.Fprintf(&b, "🏷 %s\n", format.Escape(strings.Join(st.D.Tags, ", ")))
	}

	// 🔴 Three states, three sentences. Saying nothing for the nil case is
	// correct — it means "my usual reminders", which is what the server will
	// do — but «не напоминать» has to be visible, or an event that will stay
	// silent forever looks identical to one that will not.
	if st.ReminderOffsets != nil {
		if len(*st.ReminderOffsets) == 0 {
			b.WriteString("🔕 не напоминать\n")
		} else {
			parts := make([]string, 0, len(*st.ReminderOffsets))
			for _, m := range *st.ReminderOffsets {
				parts = append(parts, humanOffset(m))
			}
			fmt.Fprintf(&b, "🔔 за %s\n", strings.Join(parts, ", "))
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// checkMark flags a value the parser read from a loose format.
//
// 🔴 Denis, 15.09: «если время написано не строго по формулировке должно
// помечать на уточнение/подтверждение». The bot now reads «13;00», «12» and
// «12:00 -12:50», which is what he asked for — but reading a sloppy format
// silently turns a typo into a fact nobody checked. The mark is the difference
// between a kindness and a guess.
func checkMark(uncertain bool) string {
	if uncertain {
		return " ⚠ проверь"
	}
	return ""
}

// humanOffset turns minutes-before into the words a person would use.
func humanOffset(m int) string {
	switch {
	case m >= 1440 && m%1440 == 0:
		return fmt.Sprintf("%d дн.", m/1440)
	case m >= 60 && m%60 == 0:
		return fmt.Sprintf("%d ч.", m/60)
	default:
		return fmt.Sprintf("%d мин.", m)
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
