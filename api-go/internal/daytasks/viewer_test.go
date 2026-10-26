package daytasks

import (
	"testing"
	"time"

	"neuroboost/api-go/internal/calendars"
)

// Audit M6 (25.09), Denis 26.10: a viewer of a shared calendar may take a
// shared task for a day although only the owner can close it; the owner's
// close counts in the viewer's day. The viewer has to be told, or a tick that
// cannot work looks broken: owner_closes says so per item.
func TestAViewerTakesASharedTaskTheOwnerCloses(t *testing.T) {
	d, ctx, owner := dayDB(t)
	_, _, viewer := dayDB(t)
	today := Today(time.Now(), ny)

	shared, err := calendars.Create(ctx, owner, "Дом", nil)
	if err != nil {
		t.Fatalf("create shared calendar: %v", err)
	}
	if _, err := d.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, $3, $4)`,
		shared.ID, viewer, calendars.RoleViewer, calendars.StatusActive); err != nil {
		t.Fatalf("seed viewer membership: %v", err)
	}
	sharedTask := newTask(t, owner, map[string]any{"title": "полить цветы", "calendar_id": shared.ID})
	ownTask := newTask(t, viewer, map[string]any{"title": "своё дело"})

	if err := Confirm(ctx, viewer, today, []string{sharedTask, ownTask}); err != nil {
		t.Fatalf("a viewer may take a shared task for the day: %v", err)
	}

	items := func() map[string]Item {
		days, err := List(ctx, viewer, today, today)
		if err != nil || len(days) != 1 {
			t.Fatalf("list: %v %+v", err, days)
		}
		out := map[string]Item{}
		for _, it := range days[0].Items {
			out[it.TaskID] = it
		}
		return out
	}

	got := items()
	if !got[sharedTask].OwnerCloses {
		t.Errorf("the viewer cannot close the shared task, owner_closes must say so: %+v", got[sharedTask])
	}
	if got[ownTask].OwnerCloses {
		t.Errorf("the viewer's own task is theirs to close: %+v", got[ownTask])
	}

	closeTask(t, owner, sharedTask)
	if !items()[sharedTask].Done {
		t.Errorf("the owner closed it: it counts in the viewer's day")
	}
}
