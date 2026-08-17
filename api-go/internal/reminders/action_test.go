package reminders

import (
	"errors"
	"testing"
)

func validReq(mutate func(*ActionRequest)) ActionRequest {
	req := ActionRequest{TgID: 495598685, ReminderID: "e2b1c0de-0000-0000-0000-000000000001", Action: ActionAck}
	if mutate != nil {
		mutate(&req)
	}
	return req
}

func TestValidateActionAcceptsEachKnownAction(t *testing.T) {
	for _, action := range []string{ActionAck, ActionSnooze, ActionDone} {
		if _, err := ValidateAction(validReq(func(r *ActionRequest) { r.Action = action })); err != nil {
			t.Errorf("%s rejected: %v", action, err)
		}
	}
}

func TestValidateActionRejectsUnknownAction(t *testing.T) {
	// An unrecognised action must not fall through to a default that does
	// something: the caller is a bot forwarding user input.
	_, err := ValidateAction(validReq(func(r *ActionRequest) { r.Action = "delete_everything" }))
	if !errors.Is(err, ErrInvalidAction) {
		t.Errorf("unknown action accepted, err = %v", err)
	}
}

func TestValidateActionRequiresIdentity(t *testing.T) {
	if _, err := ValidateAction(validReq(func(r *ActionRequest) { r.TgID = 0 })); !errors.Is(err, ErrInvalidAction) {
		t.Error("accepted a request with no tg_id")
	}
	if _, err := ValidateAction(validReq(func(r *ActionRequest) { r.ReminderID = "" })); !errors.Is(err, ErrInvalidAction) {
		t.Error("accepted a request with no reminder id")
	}
}

func TestSnoozeDefaultsToTenMinutes(t *testing.T) {
	req, err := ValidateAction(validReq(func(r *ActionRequest) { r.Action = ActionSnooze }))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Minutes != DefaultSnoozeMinutes {
		t.Errorf("minutes = %d, want %d", req.Minutes, DefaultSnoozeMinutes)
	}
}

func TestSnoozeClampsInsteadOfRejecting(t *testing.T) {
	// A button that silently does nothing is worse than one that does something
	// slightly different, so out-of-range clamps rather than 400s.
	req, err := ValidateAction(validReq(func(r *ActionRequest) {
		r.Action = ActionSnooze
		r.Minutes = 40_000
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Minutes != MaxSnoozeMinutes {
		t.Errorf("minutes = %d, want clamp to %d", req.Minutes, MaxSnoozeMinutes)
	}

	negative, err := ValidateAction(validReq(func(r *ActionRequest) {
		r.Action = ActionSnooze
		r.Minutes = -30
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if negative.Minutes != DefaultSnoozeMinutes {
		t.Errorf("negative minutes = %d, want default %d", negative.Minutes, DefaultSnoozeMinutes)
	}
}

func TestNonSnoozeActionsCarryNoMinutes(t *testing.T) {
	// Otherwise a stray minutes value rides along into a code path that would
	// have to decide what to do with it.
	req, err := ValidateAction(validReq(func(r *ActionRequest) {
		r.Action = ActionDone
		r.Minutes = 45
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Minutes != 0 {
		t.Errorf("minutes = %d, want 0 for %s", req.Minutes, ActionDone)
	}
}
