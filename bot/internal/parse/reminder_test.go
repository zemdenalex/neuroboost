package parse

import "testing"

func TestReminderOffsetInWords(t *testing.T) {
	cases := map[string]int{
		"оркестр напомнить за 15м":   15,
		"оркестр напомнить 15м":      15,
		"оркестр напомнить за 30мин": 30,
		"оркестр напомнить за 1ч":    60,
		"оркестр напомнить за 2часа": 120,
		"оркестр напомнить за 1день": 1440,
		"orchestra remind in 10min":  10,
	}
	for line, want := range cases {
		toks := Tokenize(line)
		var d Draft
		if !RecogniseReminderOffset(toks, &d) {
			t.Errorf("%q: not recognised", line)
			continue
		}
		if d.ReminderOffsets == nil || len(*d.ReminderOffsets) != 1 || (*d.ReminderOffsets)[0] != want {
			t.Errorf("%q: offsets = %v, want [%d]", line, d.ReminderOffsets, want)
		}
		title := Title(toks)
		if title != "оркестр" && title != "orchestra" {
			t.Errorf("%q: Title = %q", line, title)
		}
	}
}

// 🔴 nil is not the same as an empty list. Absent means "apply my preset";
// empty means "stay silent forever". The parser must leave nil alone when
// nothing was said — this product already shipped the collapse of those two.
func TestNoReminderWordLeavesTheFieldAbsent(t *testing.T) {
	p := ParseLine("оркестр завтра 14:00", tuesday15())
	if p.Draft.ReminderOffsets != nil {
		t.Errorf("ReminderOffsets = %v, want nil (absent)", *p.Draft.ReminderOffsets)
	}
}

// «напомнить» with nothing usable after it is not a reminder — and the word
// stays in the title rather than vanishing.
func TestBareRemindWordIsLeftAlone(t *testing.T) {
	for _, line := range []string{"напомнить позвонить", "напомнить"} {
		toks := Tokenize(line)
		var d Draft
		if RecogniseReminderOffset(toks, &d) {
			t.Errorf("%q: recognised an offset that is not there", line)
		}
		if got := Title(toks); got != line {
			t.Errorf("%q: title changed to %q", line, got)
		}
	}
}

func TestReminderPresetByName(t *testing.T) {
	toks := Tokenize("оркестр важное")
	var d Draft
	presets := map[string][]int{"важное": {60, 10}, "без": {}}
	if !RecogniseReminderPreset(toks, presets, &d) {
		t.Fatal("not recognised")
	}
	if d.ReminderOffsets == nil || len(*d.ReminderOffsets) != 2 {
		t.Fatalf("offsets = %v, want two entries", d.ReminderOffsets)
	}
	if got := Title(toks); got != "оркестр" {
		t.Errorf("Title = %q, want %q", got, "оркестр")
	}
}

// ⚠ A preset naming an empty list is a real answer — silence — and must be
// told apart from no preset at all.
func TestEmptyPresetMeansSilenceNotAbsence(t *testing.T) {
	toks := Tokenize("оркестр без")
	var d Draft
	if !RecogniseReminderPreset(toks, map[string][]int{"без": {}}, &d) {
		t.Fatal("not recognised")
	}
	if d.ReminderOffsets == nil {
		t.Fatal("ReminderOffsets is nil — that means «apply my preset», not «silent»")
	}
	if len(*d.ReminderOffsets) != 0 {
		t.Errorf("offsets = %v, want an empty list", *d.ReminderOffsets)
	}
}

func TestUnknownPresetNameIsLeftAlone(t *testing.T) {
	toks := Tokenize("оркестр важное")
	var d Draft
	if RecogniseReminderPreset(toks, map[string][]int{"срочное": {5}}, &d) {
		t.Error("matched a preset that does not exist")
	}
	if got := Title(toks); got != "оркестр важное" {
		t.Errorf("Title = %q, want the line unchanged", got)
	}
}
