package auth

import "time"

// validTimezone reports whether tz is a valid IANA timezone name (e.g.
// "Europe/Berlin"). It is backed by the system tz database — tzdata is installed
// in the runtime image — so any real zone is accepted instead of being limited to
// a hand-maintained allowlist that drifts out of sync with the UI's options.
func validTimezone(tz string) bool {
	if tz == "" {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}
