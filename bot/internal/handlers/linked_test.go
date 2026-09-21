package handlers

import (
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

// Denis 21.09: a scheduled task stays in the list, marked with its time.
func TestAScheduledTaskIsStillOpen(t *testing.T) {
	got := openTasks([]api.Task{
		{ID: "a", Status: "TODO"}, {ID: "b", Status: "SCHEDULED"},
		{ID: "c", Status: "DONE"}, {ID: "d", Status: "CANCELLED"},
	})
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Errorf("open = %+v, want a and b", got)
	}
}
