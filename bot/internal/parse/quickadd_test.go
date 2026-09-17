package parse

import (
	"testing"
	"time"
)

// Denis, 17.09 18:16 — the three inputs quick add is built for, verbatim.
// «Wednesday 16.09 10:00» is the reference clock.
func quickNow() time.Time { return time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC) }

func TestQuickAddDentist(t *testing.T) {
	p := ParseLine("завтра в 15 стоматолог напомни за час", quickNow())
	if p.Title != "стоматолог" {
		t.Errorf("Title = %q, want %q", p.Title, "стоматолог")
	}
	if !p.Draft.HasDay || p.Draft.Day.Day() != 17 {
		t.Errorf("day = %s, want 17 September", p.Draft.Day.Format("2 Jan"))
	}
	if !p.Draft.HasTime || p.Draft.Start != 15*time.Hour {
		t.Errorf("start = %v (hasTime %v), want 15:00", p.Draft.Start, p.Draft.HasTime)
	}
	if p.Draft.ReminderOffsets == nil || len(*p.Draft.ReminderOffsets) != 1 || (*p.Draft.ReminderOffsets)[0] != 60 {
		t.Errorf("reminders = %v, want [60]", p.Draft.ReminderOffsets)
	}
}

func TestQuickAddTwoEventsInOneLine(t *testing.T) {
	line := "tuesday morning office work wednesday midnight movie"
	if !LooksLikeList(line, quickNow()) {
		t.Fatalf("one line with two days and two titles must be offered as a list")
	}
	got := ParseEventList(line, quickNow())
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(got), got)
	}
	if got[0].Title != "office work" || got[0].Draft.Day.Day() != 22 || got[0].Draft.Start != 9*time.Hour {
		t.Errorf("entry 0 = %q %s %v, want «office work» 22 Sep 09:00", got[0].Title, got[0].Draft.Day.Format("2 Jan"), got[0].Draft.Start)
	}
	if got[1].Title != "movie" || got[1].Draft.Day.Day() != 23 || !got[1].Draft.HasTime || got[1].Draft.Start != 0 {
		t.Errorf("entry 1 = %q %s %v, want «movie» 23 Sep 00:00", got[1].Title, got[1].Draft.Day.Format("2 Jan"), got[1].Draft.Start)
	}
}

func TestQuickAddTaskWordIsATask(t *testing.T) {
	line := "задача на завтра доделать сайт"
	if !IsTaskLine(line) {
		t.Fatalf("«задача» must route straight to a task")
	}
	r := ParseTask(line, quickNow())
	if r.Title != "доделать сайт" {
		t.Errorf("Title = %q, want %q", r.Title, "доделать сайт")
	}
	if r.DueDate == nil || r.DueDate.Day() != 17 {
		t.Errorf("due = %v, want 17 September", r.DueDate)
	}
}

// A day word inside a line that is ONE event must not split it.
func TestOneEventWithOneDayIsNotAList(t *testing.T) {
	for _, line := range []string{"Ужин завтра 19:00", "созвон в среду 10:00", "отжаться 12 раз"} {
		if LooksLikeList(line, quickNow()) {
			t.Errorf("%q offered as a list", line)
		}
	}
	if IsTaskLine("купить задачник") {
		t.Errorf("«задачник» is not the word «задача»")
	}
}

// A span between two days is ONE entry — that is the multi-day event, not two.
func TestARangeOfDaysIsNotSplit(t *testing.T) {
	if LooksLikeList("trip from monday to friday in paris", quickNow()) {
		t.Errorf("«from monday to friday in paris» offered as two entries — the second piece has a title, so only the range word can stop the cut")
	}
}

// 🔴 The substring defect, in the task parser: «завтрак» contains «завтра».
func TestTaskBreakfastIsNotDueTomorrow(t *testing.T) {
	r := ParseTask("купить завтрак", quickNow())
	if r.DueDate != nil {
		t.Errorf("«завтрак» set a due date %v", r.DueDate)
	}
	if r.Title != "купить завтрак" {
		t.Errorf("Title = %q, want %q", r.Title, "купить завтрак")
	}
}

// «в 2 этапа» is not two o'clock — but it reads as one, so the card must say so.
func TestNumberAfterVIsALooseTime(t *testing.T) {
	p := ParseLine("встреча в 2 этапа", quickNow())
	if p.Draft.HasTime && !p.Draft.IsUncertain(FieldTime) {
		t.Errorf("«в 2» read as a time without ⚠")
	}
}
