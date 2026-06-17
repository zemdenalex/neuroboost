package export

import (
	"testing"
	"time"
)

func evAt(start, end time.Time) EventRow { return EventRow{ID: "e", Title: "t", StartsAt: start, EndsAt: end} }

func TestValidateImportEvent(t *testing.T) {
	base := time.Date(2026, 6, 17, 9, 0, 0, 0, time.UTC)

	if msg := validateImportEvent(evAt(base, base.Add(time.Hour))); msg != "" {
		t.Errorf("valid event rejected: %q", msg)
	}
	if msg := validateImportEvent(evAt(base, base)); msg == "" {
		t.Error("zero-length event (end == start) should be rejected")
	}
	if msg := validateImportEvent(evAt(base, base.Add(-time.Hour))); msg == "" {
		t.Error("inverted event (end < start) should be rejected")
	}
}

func cat(s string) *string { return &s }

func TestValidateImportTask(t *testing.T) {
	valid := []TaskRow{
		{ID: "a", Title: "x", Status: "TODO", Priority: 3},
		{ID: "b", Title: "y", Status: "DONE", Priority: 0, Category: cat("ASAP")},
		{ID: "c", Title: "z", Status: "SCHEDULED", Priority: 5, Category: cat("BUFFER")},
	}
	for i, tk := range valid {
		if msg := validateImportTask(tk); msg != "" {
			t.Errorf("valid task %d rejected: %q", i, msg)
		}
	}

	invalid := []struct {
		name string
		tk   TaskRow
	}{
		{"bad status", TaskRow{ID: "a", Title: "x", Status: "WIP", Priority: 3}},
		{"empty status", TaskRow{ID: "a", Title: "x", Status: "", Priority: 3}},
		{"priority too high", TaskRow{ID: "a", Title: "x", Status: "TODO", Priority: 6}},
		{"priority negative", TaskRow{ID: "a", Title: "x", Status: "TODO", Priority: -1}},
		{"bad category", TaskRow{ID: "a", Title: "x", Status: "TODO", Priority: 3, Category: cat("URGENT")}},
	}
	for _, c := range invalid {
		if msg := validateImportTask(c.tk); msg == "" {
			t.Errorf("%s: expected rejection, got none", c.name)
		}
	}
}
