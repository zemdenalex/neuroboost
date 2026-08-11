package notifier

import "testing"

// A real reminder id: 36 characters, which is what makes the 64-byte cap tight.
const sampleID = "e2b1c0de-1111-2222-3333-444455556666"

func TestEncodedCallbackFitsTelegramLimit(t *testing.T) {
	// 🔴 The reason the wire codes are single letters. Telegram rejects a button
	// whose callback_data exceeds 64 bytes, and it does so when the message is
	// SENT — so an over-long payload would break delivery of the notification
	// itself, not just the button.
	for _, code := range []string{codeAck, codeSnooze, codeDone} {
		data := EncodeCallback(code, sampleID)
		if !FitsCallbackLimit(data) {
			t.Errorf("callback_data %q is %d bytes, over the %d limit", data, len(data), CallbackDataLimit)
		}
	}
}

func TestParseCallbackRoundTripsEachAction(t *testing.T) {
	cases := map[string]string{codeAck: ActionAck, codeSnooze: ActionSnooze, codeDone: ActionDone}
	for code, want := range cases {
		got, ok := ParseCallback(EncodeCallback(code, sampleID))
		if !ok {
			t.Fatalf("code %q did not parse", code)
		}
		if got.Action != want {
			t.Errorf("code %q → action %q, want %q", code, got.Action, want)
		}
		if got.ReminderID != sampleID {
			t.Errorf("code %q → id %q, want %q", code, got.ReminderID, sampleID)
		}
	}
}

func TestSnoozeCarriesItsDefaultMinutes(t *testing.T) {
	cb, ok := ParseCallback(EncodeCallback(codeSnooze, sampleID))
	if !ok {
		t.Fatal("snooze did not parse")
	}
	if cb.Minutes != SnoozeMinutes {
		t.Errorf("minutes = %d, want %d", cb.Minutes, SnoozeMinutes)
	}
}

func TestParseCallbackIgnoresEverythingElse(t *testing.T) {
	// The bot already routes task_done_*, priority_*, main_menu and friends.
	// Claiming one of those would break existing commands.
	for _, data := range []string{
		"main_menu", "task_done_123", "priority_1", "", "nb:", "nb:x:" + sampleID, "nb:s:",
	} {
		if _, ok := ParseCallback(data); ok {
			t.Errorf("claimed a callback that is not ours: %q", data)
		}
	}
}

func TestKeyboardPerSourceKind(t *testing.T) {
	if kb := Keyboard("TASK", sampleID); kb == nil || len(kb.InlineKeyboard[0]) != 2 {
		t.Error("a task reminder should offer done and snooze")
	}
	if kb := Keyboard("EVENT", sampleID); kb == nil || len(kb.InlineKeyboard[0]) != 2 {
		t.Error("an event reminder should offer ack and snooze")
	}
	// A digest summarises several items, so there is no single thing to
	// complete or postpone.
	if kb := Keyboard("DIGEST", sampleID); kb != nil {
		t.Error("a digest should carry no buttons")
	}
}

func TestKeyboardIsCaseInsensitiveOnSourceKind(t *testing.T) {
	// source_kind arrives from the API as a string; a casing change there must
	// not silently turn a task's buttons into an event's.
	lower := Keyboard("task", sampleID)
	if lower == nil {
		t.Fatal("lowercase task produced no keyboard")
	}
	if lower.InlineKeyboard[0][0].Text != Keyboard("TASK", sampleID).InlineKeyboard[0][0].Text {
		t.Error("casing changed which buttons a task gets")
	}
}

func TestParseMinutesFallsBackToTheDefault(t *testing.T) {
	// Zero would mean "remind me immediately", which is an infinite loop.
	for _, raw := range []string{"", "abc", "0", "-5"} {
		if got := ParseMinutes(raw); got != SnoozeMinutes {
			t.Errorf("ParseMinutes(%q) = %d, want %d", raw, got, SnoozeMinutes)
		}
	}
	if got := ParseMinutes("30"); got != 30 {
		t.Errorf("ParseMinutes(\"30\") = %d, want 30", got)
	}
}
