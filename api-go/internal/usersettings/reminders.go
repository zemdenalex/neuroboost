// Package usersettings parses the user's settings JSONB blob.
//
// It is a leaf package on purpose: events, tasks and reminders all need to
// read reminder defaults, and reminders already imports events, so putting
// this anywhere else would create an import cycle.
package usersettings

import (
	"encoding/json"
	"strconv"
	"strings"
)

// ParseHHMM turns "08:00" into minutes since local midnight. ok=false for
// anything unparseable — the settings blob is user-writable.
func ParseHHMM(v string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(v), ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, false
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// Reminders is the `reminders` section of the "user".settings JSONB blob.
// No migration is needed for any of this — settings has been JSONB since
// migration 000005, the same trick P1 used for quick-task defaults.
type Reminders struct {
	Presets             map[string][]int
	DefaultEventPreset  string
	DefaultTaskPreset   string
	DigestAt            string
	DigestEnabled       bool
	QuietHoursRespected bool
	QuietHoursStart     string
	QuietHoursEnd       string
}

// DefaultSettings is what a user who has never opened the settings page gets.
func DefaultReminders() Reminders {
	return Reminders{
		Presets: map[string][]int{
			"важное":  {43200, 10080, 4320, 1440, 60},
			"обычное": {1440, 60},
			"без":     {},
		},
		DefaultEventPreset:  "обычное",
		DefaultTaskPreset:   "обычное",
		DigestAt:            "08:00",
		DigestEnabled:       true,
		QuietHoursRespected: true,
	}
}

// rawSettings mirrors the JSON shape with pointer fields, so "absent" is
// distinguishable from "present but zero" — an explicit false must not be
// overwritten by a default of true.
type rawSettings struct {
	Reminders *struct {
		Presets             map[string][]int `json:"presets"`
		DefaultEventPreset  *string          `json:"default_event_preset"`
		DefaultTaskPreset   *string          `json:"default_task_preset"`
		DigestAt            *string          `json:"digest_at"`
		DigestEnabled       *bool            `json:"digest_enabled"`
		QuietHoursRespected *bool            `json:"quiet_hours_respected"`
	} `json:"reminders"`
	QuietHoursStart *string `json:"quiet_hours_start"`
	QuietHoursEnd   *string `json:"quiet_hours_end"`
}

// ParseSettings never fails. The blob is user-writable and a scan that gave up
// on one malformed profile would stop reminders for that user silently.
func ParseReminders(raw []byte) Reminders {
	s := DefaultReminders()
	if len(raw) == 0 {
		return s
	}
	var parsed rawSettings
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Try field-by-field rather than giving up: a single bad value
		// (digest_at as a number) must not discard the user's presets.
		var loose map[string]json.RawMessage
		if json.Unmarshal(raw, &loose) != nil {
			return s
		}
		applyLooseTop(&s, loose)
		if rem, ok := loose["reminders"]; ok {
			applyLooseReminders(&s, rem)
		}
		return s
	}
	if parsed.QuietHoursStart != nil {
		s.QuietHoursStart = *parsed.QuietHoursStart
	}
	if parsed.QuietHoursEnd != nil {
		s.QuietHoursEnd = *parsed.QuietHoursEnd
	}
	r := parsed.Reminders
	if r == nil {
		return s
	}
	if len(r.Presets) > 0 {
		s.Presets = r.Presets
	}
	if r.DefaultEventPreset != nil {
		s.DefaultEventPreset = *r.DefaultEventPreset
	}
	if r.DefaultTaskPreset != nil {
		s.DefaultTaskPreset = *r.DefaultTaskPreset
	}
	if r.DigestAt != nil {
		if _, ok := ParseHHMM(*r.DigestAt); ok {
			s.DigestAt = *r.DigestAt
		}
	}
	if r.DigestEnabled != nil {
		s.DigestEnabled = *r.DigestEnabled
	}
	if r.QuietHoursRespected != nil {
		s.QuietHoursRespected = *r.QuietHoursRespected
	}
	return s
}

// applyLooseTop salvages the top-level quiet-hours fields when the strict
// unmarshal failed somewhere inside `reminders`.
func applyLooseTop(s *Reminders, fields map[string]json.RawMessage) {
	if v, ok := fields["quiet_hours_start"]; ok {
		var at string
		if json.Unmarshal(v, &at) == nil {
			s.QuietHoursStart = at
		}
	}
	if v, ok := fields["quiet_hours_end"]; ok {
		var at string
		if json.Unmarshal(v, &at) == nil {
			s.QuietHoursEnd = at
		}
	}
}

// applyLooseReminders salvages individually-valid fields from a reminders
// object whose strict unmarshal failed.
func applyLooseReminders(s *Reminders, rem json.RawMessage) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(rem, &fields) != nil {
		return
	}
	if v, ok := fields["presets"]; ok {
		var presets map[string][]int
		if json.Unmarshal(v, &presets) == nil && len(presets) > 0 {
			s.Presets = presets
		}
	}
	if v, ok := fields["default_event_preset"]; ok {
		var name string
		if json.Unmarshal(v, &name) == nil {
			s.DefaultEventPreset = name
		}
	}
	if v, ok := fields["default_task_preset"]; ok {
		var name string
		if json.Unmarshal(v, &name) == nil {
			s.DefaultTaskPreset = name
		}
	}
	if v, ok := fields["digest_at"]; ok {
		var at string
		if json.Unmarshal(v, &at) == nil {
			if _, valid := ParseHHMM(at); valid {
				s.DigestAt = at
			}
		}
	}
	if v, ok := fields["digest_enabled"]; ok {
		var enabled bool
		if json.Unmarshal(v, &enabled) == nil {
			s.DigestEnabled = enabled
		}
	}
	if v, ok := fields["quiet_hours_respected"]; ok {
		var respected bool
		if json.Unmarshal(v, &respected) == nil {
			s.QuietHoursRespected = respected
		}
	}
}

// OffsetsForPreset resolves a preset name to its offsets. An unknown name
// yields an empty slice, never nil, so callers can range over it and so
// "unknown preset" behaves exactly like the "без" preset.
func OffsetsForPreset(s Reminders, name string) []int {
	if offsets, ok := s.Presets[name]; ok && offsets != nil {
		return offsets
	}
	return []int{}
}
