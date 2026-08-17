package notifier

import (
	"strings"
	"testing"
)

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

// An invitation gets yes/no, and no snooze.
//
// 🔴 Snooze is deliberately absent: it re-sends the same notification later,
// and an invitation is a question, not a reminder. A third button offering
// neither answer would make the message read as a chore to postpone.
func TestKeyboardForInvite(t *testing.T) {
	kb := Keyboard("INVITE", "11111111-2222-3333-4444-555555555555")
	if kb == nil {
		t.Fatal("an invitation must carry buttons — answering it in the chat is the point")
	}

	var codes []string
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData != nil {
				codes = append(codes, *b.CallbackData)
			}
		}
	}
	if len(codes) != 2 {
		t.Fatalf("want accept and decline, got %d buttons", len(codes))
	}

	for _, data := range codes {
		cb, ok := ParseCallback(data)
		if !ok {
			t.Fatalf("the button we just built does not parse: %q", data)
		}
		if cb.Action != ActionAccept && cb.Action != ActionDecline {
			t.Errorf("unexpected action on an invitation: %q", cb.Action)
		}
		if cb.Action == ActionSnooze {
			t.Error("an invitation must not offer snooze")
		}
	}
}

// KeyboardFits asks the real buttons rather than a representative one.
//
// The old check sampled EncodeCallback(codeSnooze, id), which was right only
// because every action code happens to be one byte — and the INVITE keyboard
// has no snooze button at all.
func TestKeyboardFitsChecksEveryButton(t *testing.T) {
	ok := Keyboard("INVITE", "11111111-2222-3333-4444-555555555555")
	if !KeyboardFits(ok) {
		t.Error("a normal UUID must fit inside the callback limit")
	}

	// The negative control: something long enough to be refused. Without it,
	// a KeyboardFits that always returned true would pass the line above.
	tooLong := Keyboard("INVITE", strings.Repeat("x", CallbackDataLimit+10))
	if KeyboardFits(tooLong) {
		t.Error("an oversized payload must be refused — Telegram rejects the whole message")
	}

	if KeyboardFits(nil) {
		t.Error("no keyboard is not a keyboard that fits")
	}
}

// Every button any keyboard can produce must have something to say back.
//
// 🔴 This is the test that would have caught the 17.08 silence. The INVITE
// keyboard shipped with Accept and Decline buttons; the handler's switch had
// cases for snooze, done and ack only. The API returned 200, the membership
// changed, and the chat said nothing — so it looked broken and got pressed
// seven times.
//
// It walks the REAL keyboards for every source kind and parses the REAL
// callback data, rather than a list of actions written beside it: a list would
// have been written from the same mistaken idea of which actions exist.
func TestEveryButtonHasAReply(t *testing.T) {
	seen := 0
	for _, kind := range KnownSourceKinds {
		kb := Keyboard(kind, sampleID)
		if kb == nil {
			continue // a digest has no buttons, deliberately
		}
		for _, row := range kb.InlineKeyboard {
			for _, b := range row {
				if b.CallbackData == nil {
					continue
				}
				cb, ok := ParseCallback(*b.CallbackData)
				if !ok {
					t.Fatalf("%s: button data does not parse: %q", kind, *b.CallbackData)
				}
				if ActionReply(cb.Action) == "" {
					t.Errorf("%s: pressing %q says nothing back — a silent success reads as a failure",
						kind, cb.Action)
				}
				seen++
			}
		}
	}
	// The floor: if Keyboard ever stops returning buttons, the loop above
	// asserts nothing and passes. That is the shape of the original defect.
	if seen < 6 {
		t.Fatalf("only %d buttons examined — the sweep is no longer covering the keyboards", seen)
	}
}
