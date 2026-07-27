package usersettings

import "testing"

func TestParseSettingsDefaultsOnEmptyBlob(t *testing.T) {
	s := ParseReminders(nil)
	if s.DigestAt != "08:00" {
		t.Errorf("DigestAt = %q, want 08:00", s.DigestAt)
	}
	if !s.QuietHoursRespected {
		t.Error("QuietHoursRespected should default true")
	}
	if got := OffsetsForPreset(s, s.DefaultEventPreset); len(got) != 2 {
		t.Errorf("default event preset = %v, want the 2-entry обычное preset", got)
	}
}

func TestParseSettingsReadsAFullBlob(t *testing.T) {
	raw := []byte(`{
		"quiet_hours_start": "23:00",
		"quiet_hours_end": "06:30",
		"reminders": {
			"presets": {"мой": [30]},
			"default_event_preset": "мой",
			"default_task_preset": "мой",
			"digest_at": "09:15",
			"digest_enabled": false,
			"quiet_hours_respected": false
		}
	}`)
	s := ParseReminders(raw)
	if s.DigestAt != "09:15" {
		t.Errorf("DigestAt = %q", s.DigestAt)
	}
	// false must survive: a pointer field is why "absent" and "explicitly
	// false" do not collapse into the same thing.
	if s.DigestEnabled {
		t.Error("digest_enabled=false was overwritten by the default")
	}
	if s.QuietHoursRespected {
		t.Error("quiet_hours_respected=false was overwritten by the default")
	}
	if s.QuietHoursStart != "23:00" || s.QuietHoursEnd != "06:30" {
		t.Errorf("quiet hours = %q..%q", s.QuietHoursStart, s.QuietHoursEnd)
	}
	if got := OffsetsForPreset(s, s.DefaultTaskPreset); len(got) != 1 || got[0] != 30 {
		t.Errorf("task preset = %v", got)
	}
}

func TestParseSettingsGarbageFallsBackPerField(t *testing.T) {
	// The settings blob is user-writable. A broken digest_at must not take
	// the presets down with it.
	s := ParseReminders([]byte(`{"reminders":{"digest_at":12345,"presets":{"важное":[60]}}}`))
	if s.DigestAt != "08:00" {
		t.Errorf("bad digest_at should fall back, got %q", s.DigestAt)
	}
	if got := OffsetsForPreset(s, "важное"); len(got) != 1 || got[0] != 60 {
		t.Errorf("preset lost: %v", got)
	}
}

func TestParseSettingsRejectsUnparseableDigestTime(t *testing.T) {
	s := ParseReminders([]byte(`{"reminders":{"digest_at":"25:99"}}`))
	if s.DigestAt != "08:00" {
		t.Errorf("out-of-range digest_at should fall back, got %q", s.DigestAt)
	}
}

func TestParseSettingsMalformedJSONIsNotFatal(t *testing.T) {
	s := ParseReminders([]byte(`{not json at all`))
	if s.DigestAt != "08:00" {
		t.Errorf("malformed JSON should yield defaults, got %q", s.DigestAt)
	}
}

func TestOffsetsForUnknownPresetIsEmptyNotNil(t *testing.T) {
	// The scan ranges over this; nil would work in Go but an explicit empty
	// slice keeps the "no reminders" case identical to the "без" preset.
	got := OffsetsForPreset(ParseReminders(nil), "does-not-exist")
	if got == nil || len(got) != 0 {
		t.Errorf("unknown preset = %v, want empty non-nil", got)
	}
}
