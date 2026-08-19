package handlers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

func TestTaskCardShowsOnlyWhatWasParsed(t *testing.T) {
	r := parse.ParseTask("позвонить в банк", time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC))
	got := taskCardText(r, "UTC")
	if !strings.Contains(got, "позвонить в банк") {
		t.Errorf("the title is missing:\n%s", got)
	}
	if strings.Contains(got, "📅") || strings.Contains(got, "⏱") {
		t.Errorf("the card invented a due date or an estimate:\n%s", got)
	}
}

func TestTaskCardShowsEverythingThatWasParsed(t *testing.T) {
	r := parse.ParseTask("позвонить в банк завтра 30м !1", time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC))
	got := taskCardText(r, "UTC")
	for _, want := range []string{"📅", "⏱", "30"} {
		if !strings.Contains(got, want) {
			t.Errorf("card is missing %q:\n%s", want, got)
		}
	}
}

func TestTaskCardEscapesTheTitle(t *testing.T) {
	r := parse.ParseTask("R&D <срочно>", time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC))
	got := taskCardText(r, "UTC")
	if strings.Contains(got, "<срочно>") {
		t.Errorf("an unescaped title reached an HTML message:\n%s", got)
	}
}

func TestUnsetPriorityIsAbsentFromTheWire(t *testing.T) {
	// 🔴 The whole point of *int. 0 is Buffer — a real choice. If an unset
	// priority serialised as 0, "skip" would silently mean "lowest urgency".
	body, err := json.Marshal(api.CreateTaskReq{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "priority") {
		t.Errorf("an unset priority was sent anyway: %s", body)
	}
}

func TestBufferPriorityIsSentAsZero(t *testing.T) {
	zero := 0
	body, err := json.Marshal(api.CreateTaskReq{Title: "x", Priority: &zero})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"priority":0`) {
		t.Errorf("Buffer (0) did not reach the wire: %s", body)
	}
}
