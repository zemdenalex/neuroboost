package handlers

import (
	"strings"
	"testing"
)

func TestLoadBarFillsInProportion(t *testing.T) {
	cases := []struct {
		scheduled, available float64
		wantFilled           int
		wantPercent          string
	}{
		{0, 40, 0, "0%"},
		{20, 40, 5, "50%"},
		{40, 40, 10, "100%"},
		{4, 40, 1, "10%"},
	}
	for _, c := range cases {
		got := loadBar(c.scheduled, c.available)
		if filled := strings.Count(got, "▰"); filled != c.wantFilled {
			t.Errorf("%v/%v: %d filled cells, want %d (%q)", c.scheduled, c.available, filled, c.wantFilled, got)
		}
		if !strings.Contains(got, c.wantPercent) {
			t.Errorf("%v/%v: %q does not contain %s", c.scheduled, c.available, got, c.wantPercent)
		}
		if total := strings.Count(got, "▰") + strings.Count(got, "▱"); total != 10 {
			t.Errorf("%v/%v: %d cells, want 10 (%q)", c.scheduled, c.available, total, got)
		}
	}
}

func TestLoadBarSurvivesZeroAvailableHours(t *testing.T) {
	// 🔴 Reachable, not hypothetical: a user who unticks every working day has
	// zero available hours, and ParseWorkWeek keeps that on purpose. Dividing by
	// it would print "NaN%" into the chat, or +Inf cells into a loop.
	got := loadBar(12, 0)
	if strings.Contains(got, "NaN") || strings.Contains(got, "Inf") {
		t.Errorf("got %q", got)
	}
	if total := strings.Count(got, "▰") + strings.Count(got, "▱"); total != 10 {
		t.Errorf("%q does not draw ten cells", got)
	}
	if !strings.Contains(got, "не заданы") {
		t.Errorf("%q does not say why the bar is empty", got)
	}
}

func TestLoadBarClampsAnOverbookedWeek(t *testing.T) {
	// Being over budget is the state this screen exists to reveal, so it must
	// render rather than overflow: 60 of 40 hours is eleven cells if nothing
	// clamps, and the percentage must still be the true one.
	got := loadBar(60, 40)
	if filled := strings.Count(got, "▰"); filled != 10 {
		t.Errorf("%d filled cells, want 10", filled)
	}
	if total := strings.Count(got, "▰") + strings.Count(got, "▱"); total != 10 {
		t.Errorf("%q draws %d cells", got, total)
	}
	if !strings.Contains(got, "150%") {
		t.Errorf("%q hides how far over the week is", got)
	}
}

func TestFormatHoursDropsAPointlessDecimal(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want string
	}{
		{40, "40ч"},
		{0, "0ч"},
		{6.5, "6.5ч"},
		{25.5, "25.5ч"},
	} {
		if got := formatHours(c.in); got != c.want {
			t.Errorf("formatHours(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPlanningTextSaysWhatToDoNext(t *testing.T) {
	// The screen's whole point is that the tasks below are actionable. An empty
	// week and a full one need different sentences, and the singular case needs
	// a third — "Задач без времени: 1" reads as a bug report.
	empty := planningText(0, 40, 0)
	if !strings.Contains(empty, "нет") {
		t.Errorf("empty week: %q", empty)
	}
	if strings.Contains(empty, "Нажми") {
		t.Errorf("empty week invites a tap on nothing: %q", empty)
	}

	one := planningText(8, 40, 1)
	if !strings.Contains(one, "Одна задача") {
		t.Errorf("singular: %q", one)
	}

	many := planningText(8, 40, 6)
	if !strings.Contains(many, "6") {
		t.Errorf("plural does not say how many: %q", many)
	}
}

func TestTruncateLabelCountsCharactersNotBytes(t *testing.T) {
	// 🔴 Every title here is Russian, where a character is two bytes in UTF-8. A
	// byte-based cut would show half the intended text and could split a
	// character down the middle, which Telegram renders as �.
	long := strings.Repeat("я", 60)
	got := truncateLabel(long)
	r := []rune(got)
	if len(r) > 32 {
		t.Errorf("truncated to %d runes, want at most 32", len(r))
	}
	if len(r) < 20 {
		t.Errorf("truncated to %d runes — this is a byte-based cut", len(r))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("%q does not show that it was cut", got)
	}
	if strings.Contains(got, "�") {
		t.Errorf("%q contains a replacement character — a rune was split", got)
	}

	short := "Обед"
	if truncateLabel(short) != short {
		t.Errorf("a short label was altered: %q", truncateLabel(short))
	}
}
