package calendars

import "testing"

func TestAccessibleIDsKeepsOnlyActiveMemberships(t *testing.T) {
	// 🔴 An invitee sees nothing until they accept. If this rule breaks,
	// a person would see someone else's calendar they never accepted into.
	got := AccessibleIDs([]Membership{
		{CalendarID: "a", Status: StatusActive},
		{CalendarID: "b", Status: StatusInvited},
		{CalendarID: "c", Status: StatusActive},
	})
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("got %v, want [a c]", got)
	}
}

func TestAccessibleIDsOnEmptyInputIsEmptyNotNil(t *testing.T) {
	// An empty slice ends up in `calendar_id = ANY($1)`. nil there would mean
	// "no condition", i.e. returning rows that belong to someone else.
	got := AccessibleIDs(nil)
	if got == nil {
		t.Fatal("AccessibleIDs(nil) returned nil; must be an empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestAccessibleIDsIgnoresUnknownStatus(t *testing.T) {
	// An unrecognised status is treated as "no access", not "has access".
	got := AccessibleIDs([]Membership{{CalendarID: "a", Status: "revoked"}})
	if len(got) != 0 {
		t.Fatalf("got %v, want empty for an unrecognised status", got)
	}
}
