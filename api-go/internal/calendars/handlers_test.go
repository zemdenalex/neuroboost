package calendars

import "testing"

// TestValidColor pins the hex-colour rule the handler enforces: a nil pointer
// (field omitted) is always valid — it means "not supplied" — while a
// non-nil pointer must be a strict 6-digit hex colour. An empty string is
// deliberately rejected rather than accepted as "clear the colour".
func TestValidColor(t *testing.T) {
	str := func(s string) *string { return &s }

	cases := []struct {
		name  string
		color *string
		want  bool
	}{
		{"nil is valid (omitted)", nil, true},
		{"lowercase hex", str("#7c3aed"), true},
		{"uppercase hex", str("#7C3AED"), true},
		{"empty string is invalid", str(""), false},
		{"missing hash", str("7c3aed"), false},
		{"too short", str("#7c3ae"), false},
		{"too long", str("#7c3aed1"), false},
		{"non-hex characters", str("red"), false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := validColor(c.color); got != c.want {
				t.Errorf("validColor(%v) = %v, want %v", c.color, got, c.want)
			}
		})
	}
}
