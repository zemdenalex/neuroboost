package tasks

import "testing"

func TestValidPriority(t *testing.T) {
	for _, p := range []int{0, 1, 2, 3, 4, 5} {
		if !validPriority(p) {
			t.Errorf("validPriority(%d) = false, want true", p)
		}
	}
	for _, p := range []int{-1, 6, 99, -100} {
		if validPriority(p) {
			t.Errorf("validPriority(%d) = true, want false", p)
		}
	}
}

func TestValidStatus(t *testing.T) {
	for _, s := range []TaskStatus{StatusTodo, StatusInProgress, StatusScheduled, StatusDone, StatusCancelled} {
		if !validStatus(s) {
			t.Errorf("validStatus(%q) = false, want true", s)
		}
	}
	for _, s := range []TaskStatus{"", "GARBAGE", "todo", "Done"} {
		if validStatus(s) {
			t.Errorf("validStatus(%q) = true, want false", s)
		}
	}
}

func TestValidCategory(t *testing.T) {
	for _, c := range []TaskCategory{CategoryEmergency, CategoryAsap, CategoryMustToday, CategoryDeadlineSoon, CategoryIfPossible, CategoryBuffer} {
		if !validCategory(c) {
			t.Errorf("validCategory(%q) = false, want true", c)
		}
	}
	for _, c := range []TaskCategory{"", "NOPE", "emergency"} {
		if validCategory(c) {
			t.Errorf("validCategory(%q) = true, want false", c)
		}
	}
}

func TestValidateTaskMutation(t *testing.T) {
	bad := TaskStatus("BAD")
	badPri := 7
	badCat := TaskCategory("BAD")
	okStatus := StatusDone
	okPri := 2
	okCat := CategoryAsap

	// All nil → valid (nothing to validate).
	if code, _ := validateTaskMutation(nil, nil, nil); code != "" {
		t.Errorf("all-nil returned code %q, want empty", code)
	}

	// All valid → no error.
	if code, _ := validateTaskMutation(&okStatus, &okPri, &okCat); code != "" {
		t.Errorf("valid fields returned code %q, want empty", code)
	}

	// Each bad field flagged with the right code.
	if code, _ := validateTaskMutation(&bad, nil, nil); code != "INVALID_STATUS" {
		t.Errorf("bad status code = %q, want INVALID_STATUS", code)
	}
	if code, _ := validateTaskMutation(nil, &badPri, nil); code != "INVALID_PRIORITY" {
		t.Errorf("bad priority code = %q, want INVALID_PRIORITY", code)
	}
	if code, _ := validateTaskMutation(nil, nil, &badCat); code != "INVALID_CATEGORY" {
		t.Errorf("bad category code = %q, want INVALID_CATEGORY", code)
	}
}
