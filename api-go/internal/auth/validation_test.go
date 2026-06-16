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

	invalid := []string{"", "Not/AZone", "garbage", "Mars/Phobos", "Moscow"}
	for _, tz := range invalid {
		if validTimezone(tz) {
			t.Errorf("validTimezone(%q) = true, want false", tz)
		}
	}
}
