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
