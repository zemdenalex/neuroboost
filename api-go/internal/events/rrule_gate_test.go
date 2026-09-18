package events

import (
	"errors"
	"testing"
)

// 🔴 The rules the bot and the web can produce must be refused on WRITE if this
// API cannot expand them.
//
// Before 18.09 an unknown rule was stored happily, and the read path then fell
// back to "include the parent event as-is" (handlers.go:71) — so «каждый год»
// created an event that appeared once, never repeated, and reported nothing.
// The user believed they had a series. Refusing at the door is the only place
// this can be fixed for good: a rule already in the database is a series
// somebody is already waiting for.
func TestUnsupportedRulesAreRefusedNotStored(t *testing.T) {
	refused := []string{
		"FREQ=YEARLY",          // the bot wrote this until v0.4.11.2
		"FREQ=WEEKLY;BYDAY=TU", // and this
		"FREQ=HOURLY",
		"RRULE:FREQ=DAILY", // the iCal prefix, a plausible client mistake
		"every day",
	}
	for _, r := range refused {
		if _, err := parseRRule(r); err == nil {
			t.Errorf("parseRRule(%q) accepted a rule this API cannot expand", r)
		}
	}

	// The positive control: everything the product actually writes must pass,
	// or this gate would reject real repeats instead of broken ones.
	for _, r := range []string{
		"FREQ=DAILY",
		"FREQ=WEEKLY",
		"FREQ=MONTHLY",
		"FREQ=MONTHLY;INTERVAL=12", // how the bot spells «каждый год»
		"FREQ=DAILY;INTERVAL=3",
		"FREQ=WEEKLY;COUNT=10",
		"FREQ=DAILY;UNTIL=2026-12-01",
	} {
		if _, err := parseRRule(r); err != nil {
			t.Errorf("parseRRule(%q) refused a rule the product writes: %v", r, err)
		}
	}
}

// The sentinel exists so the handler can answer 400. If it stops being
// distinguishable, an unsupported rule goes back to reading as a server fault.
func TestInvalidRruleIsItsOwnError(t *testing.T) {
	wrapped := errors.Join(ErrInvalidRrule, errors.New("FREQ=YEARLY"))
	if !errors.Is(wrapped, ErrInvalidRrule) {
		t.Error("ErrInvalidRrule does not survive wrapping; the handler would answer 500")
	}
}
