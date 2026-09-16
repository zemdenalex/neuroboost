package parse

import "testing"

// 🔴 Denis's own sentence, item 1: «среда 14:00-15:00 оркестр повтор» must be
// titled «оркестр» — «но повтор/повторять и тд не должно идти в название» —
// and must ask for a frequency, because none was given.
func TestRepeatWordIsClaimedAndAsksForFrequency(t *testing.T) {
	for _, word := range []string{"повтор", "повторять", "повторяется", "repeat", "recurring"} {
		toks := Tokenize("оркестр " + word)
		var d Draft
		if !recogniseRepeat(toks, &d) {
			t.Errorf("%q: not recognised", word)
			continue
		}
		if !d.RepeatAsked {
			t.Errorf("%q: RepeatAsked is false — the card would create a one-off silently", word)
		}
		if d.Repeat != "" {
			t.Errorf("%q: Repeat = %q, want empty (no frequency was given)", word, d.Repeat)
		}
		if got := Title(toks); got != "оркестр" {
			t.Errorf("%q: Title = %q, want %q", word, got, "оркестр")
		}
	}
}

func TestNamedFrequenciesNeedNoQuestion(t *testing.T) {
	cases := map[string]string{
		"ежедневно":     "FREQ=DAILY",
		"каждый день":   "FREQ=DAILY",
		"daily":         "FREQ=DAILY",
		"еженедельно":   "FREQ=WEEKLY",
		"каждую неделю": "FREQ=WEEKLY",
		"weekly":        "FREQ=WEEKLY",
		"ежемесячно":    "FREQ=MONTHLY",
		"каждый месяц":  "FREQ=MONTHLY",
		"ежегодно":      "FREQ=YEARLY",
	}
	for line, want := range cases {
		toks := Tokenize("оркестр " + line)
		var d Draft
		if !recogniseRepeat(toks, &d) {
			t.Errorf("%q: not recognised", line)
			continue
		}
		if d.Repeat != want {
			t.Errorf("%q: Repeat = %q, want %q", line, d.Repeat, want)
		}
		if d.RepeatAsked {
			t.Errorf("%q: asks for a frequency it was given", line)
		}
		if got := Title(toks); got != "оркестр" {
			t.Errorf("%q: Title = %q, want %q", line, got, "оркестр")
		}
	}
}

// 🔴 «каждый вторник» is two facts at once: the rule repeats weekly on
// Tuesdays, AND the first one is the coming Tuesday. So the repeat recogniser
// claims only «каждый» and deliberately leaves the weekday for the weekday
// recogniser — which is also why repeat runs before it.
func TestEveryWeekdayLeavesTheDayForTheDayRecogniser(t *testing.T) {
	toks := Tokenize("оркестр каждый вторник")
	var d Draft
	if !recogniseRepeat(toks, &d) {
		t.Fatal("not recognised")
	}
	if d.Repeat != "FREQ=WEEKLY;BYDAY=TU" {
		t.Errorf("Repeat = %q, want %q", d.Repeat, "FREQ=WEEKLY;BYDAY=TU")
	}
	if !recogniseWeekday(toks, tuesday15(), &d) {
		t.Fatal("the weekday was swallowed by the repeat recogniser")
	}
	if d.Day.Day() != 15 {
		t.Errorf("first occurrence = %d September, want 15", d.Day.Day())
	}
	if got := Title(toks); got != "оркестр" {
		t.Errorf("Title = %q, want %q", got, "оркестр")
	}
}

func TestAllDayClaimsBothWordsAndDropsTheTime(t *testing.T) {
	for _, line := range []string{"анализы весь день", "анализы целый день", "checkup all day", "checkup allday"} {
		toks := Tokenize(line)
		var d Draft
		d.HasTime, d.Start = true, hhmm(10, 0)
		if !recogniseAllDay(toks, &d) {
			t.Errorf("%q: not recognised", line)
			continue
		}
		if !d.AllDay {
			t.Errorf("%q: AllDay is false", line)
		}
		if d.HasTime {
			t.Errorf("%q: a time survived an all-day event", line)
		}
		title := Title(toks)
		if title != "анализы" && title != "checkup" {
			t.Errorf("%q: Title = %q", line, title)
		}
	}
}

// «день» on its own is a noun, not a keyword. Claiming it would retitle
// «хороший день» to «хороший».
func TestBareDayWordIsNotAllDay(t *testing.T) {
	toks := Tokenize("хороший день")
	var d Draft
	if recogniseAllDay(toks, &d) {
		t.Error("«хороший день» was read as an all-day marker")
	}
	if got := Title(toks); got != "хороший день" {
		t.Errorf("Title = %q, want the line unchanged", got)
	}
}

func TestKindWordMakesItATask(t *testing.T) {
	for _, word := range []string{"задача", "задачу", "task"} {
		toks := Tokenize("отжаться " + word)
		var d Draft
		if !recogniseKind(toks, &d) {
			t.Errorf("%q: not recognised", word)
			continue
		}
		if !d.IsTask {
			t.Errorf("%q: IsTask is false", word)
		}
		if got := Title(toks); got != "отжаться" {
			t.Errorf("%q: Title = %q, want %q", word, got, "отжаться")
		}
	}
}

// Colours resolve to palette NAMES, not to words. The web paints from
// web/src/lib/calendar/palette.ts and understands nothing else — a colour
// stored as «синий» would be saved, shown as set, and never painted.
func TestColourWordsResolveToPaletteNames(t *testing.T) {
	cases := map[string]string{
		"синий":      "blue",
		"синее":      "blue",
		"синяя":      "blue",
		"красный":    "red",
		"зелёное":    "green",
		"зеленая":    "green",
		"жёлтый":     "amber",
		"оранжевый":  "amber",
		"розовая":    "pink",
		"фиолетовый": "violet",
		"серый":      "slate",
		"голубой":    "cyan",
		"blue":       "blue",
		"red":        "red",
		"purple":     "violet",
		"grey":       "slate",
	}
	for word, want := range cases {
		toks := Tokenize("оркестр " + word)
		var d Draft
		if !recogniseColour(toks, &d) {
			t.Errorf("%q: not recognised", word)
			continue
		}
		if d.Colour != want {
			t.Errorf("%q: Colour = %q, want %q", word, d.Colour, want)
		}
		if got := Title(toks); got != "оркестр" {
			t.Errorf("%q: Title = %q, want %q", word, got, "оркестр")
		}
	}
}

// 🔴 A colour stem must not eat an ordinary word that begins with it. «серый»
// is a colour; «серьёзно», «сердце» and «середина» are not, and prefix
// matching would have taken all four.
func TestColourStemsDoNotEatOrdinaryWords(t *testing.T) {
	for _, line := range []string{"серьёзно", "сердце", "середина", "синица", "краснодар"} {
		toks := Tokenize(line)
		var d Draft
		if recogniseColour(toks, &d) {
			t.Errorf("%q: read as the colour %q", line, d.Colour)
		}
		if got := Title(toks); got != line {
			t.Errorf("%q: title changed to %q", line, got)
		}
	}
}

func TestTagsAreClaimedAndLowercased(t *testing.T) {
	toks := Tokenize("Оркестр #Музыка #дела #музыка")
	var d Draft
	if !recogniseTags(toks, &d) {
		t.Fatal("not recognised")
	}
	if len(d.Tags) != 2 || d.Tags[0] != "музыка" || d.Tags[1] != "дела" {
		t.Errorf("Tags = %v, want [музыка дела]", d.Tags)
	}
	if got := Title(toks); got != "Оркестр" {
		t.Errorf("Title = %q, want %q", got, "Оркестр")
	}
}
