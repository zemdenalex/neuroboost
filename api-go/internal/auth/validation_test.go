package auth

import "testing"

func TestValidTimezone(t *testing.T) {
	valid := []string{
		"Europe/Moscow",
		"Europe/Berlin", // offered by the UI but rejected by the old allowlist
		"Asia/Shanghai", // offered by the UI but rejected by the old allowlist
		"America/New_York",
		"Asia/Kolkata",
		"Australia/Sydney",
		"UTC",
	}
	for _, tz := range valid {
		if !validTimezone(tz) {
			t.Errorf("validTimezone(%q) = false, want true", tz)
		}
	}

	// "Local" is Go's name for the server's own zone: time.LoadLocation takes
	// it, Postgres answers «time zone "Local" not recognized» (checked 25.09 on
	// postgres:16). Stored, it made every task list 500 for that user forever
	// (audit 25.09, M1).
	invalid := []string{"", "Not/AZone", "garbage", "Mars/Phobos", "Moscow", "Local"}
	for _, tz := range invalid {
		if validTimezone(tz) {
			t.Errorf("validTimezone(%q) = true, want false", tz)
		}
	}
}
