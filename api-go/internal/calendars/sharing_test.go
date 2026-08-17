package calendars

import "testing"

func ptr(s string) *string { return &s }

// Every column feeding DisplayLabel is nullable, and a real account fills an
// arbitrary subset of them: a Telegram-only user has no email, an email user has
// no tg_first_name. The chain therefore has to survive every gap, including all
// of them at once.
func TestDisplayLabelPrefersTheShortestNameAvailable(t *testing.T) {
	cases := []struct {
		name                                      string
		displayName, tgFirstName, tgUsername, mail *string
		want                                      string
	}{
		{"display name wins", ptr("Настя"), ptr("Anastasia"), ptr("nastya"), ptr("a@b.com"), "Настя"},
		{"telegram first name is next", nil, ptr("Anastasia"), ptr("nastya"), ptr("a@b.com"), "Anastasia"},
		{"then the username", nil, nil, ptr("nastya"), ptr("a@b.com"), "nastya"},
		{"email falls back to its local part", nil, nil, nil, ptr("zemdenalex@gmail.com"), "zemdenalex"},
		{"an account with nothing still has a label", nil, nil, nil, nil, "Участник"},

		// Empty strings are not absence to the database but are absence to a
		// reader. Without TrimSpace the label would render as blank space and
		// look like a rendering bug rather than a missing name.
		{"blank display name is skipped", ptr("   "), ptr("Anastasia"), nil, nil, "Anastasia"},
		{"blank everything reaches the constant", ptr(""), ptr(" "), ptr(""), ptr("  "), "Участник"},

		// A malformed address must not produce "" by cutting at a '@' that is
		// not there, nor an empty local part from a leading '@'.
		{"address without an at sign is used whole", nil, nil, nil, ptr("weird"), "weird"},
		{"leading at sign does not yield an empty label", nil, nil, nil, ptr("@host"), "@host"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DisplayLabel(c.displayName, c.tgFirstName, c.tgUsername, c.mail); got != c.want {
				t.Fatalf("DisplayLabel = %q, want %q", got, c.want)
			}
		})
	}
}

// The label lands in a calendar block next to the title. The whole reason it is
// not the full email is width, so assert the property rather than trusting the
// one example above to keep testing it.
func TestDisplayLabelNeverReturnsAnEmailAddress(t *testing.T) {
	got := DisplayLabel(nil, nil, nil, ptr("zemdenalex@gmail.com"))
	if len(got) >= len("zemdenalex@gmail.com") {
		t.Fatalf("label %q is no shorter than the address it came from", got)
	}
	for _, r := range got {
		if r == '@' {
			t.Fatalf("label %q still carries the address domain", got)
		}
	}
}
