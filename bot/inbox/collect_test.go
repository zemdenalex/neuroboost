package inbox

import (
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
)

func channelFwd(id int, text, chanName string, post int) *models.Message {
	return &models.Message{ID: id, Text: text, ForwardOrigin: &models.MessageOrigin{
		Type: models.MessageOriginTypeChannel, MessageOriginChannel: &models.MessageOriginChannel{
			MessageID: post, Chat: models.Chat{ID: -1001, Title: chanName, Username: strings.ToLower(chanName)}}}}
}

func own(id int, text string) *models.Message { return &models.Message{ID: id, Text: text} }

// Denis forwards a channel post and types «#idea #avocar» with it. The post's
// own hashtags (#ai #news) must not become his tags, and the comment must not
// become a second note.
func TestForwardWithCommentIsOneNoteTaggedByTheComment(t *testing.T) {
	ds := Build(5, []*models.Message{
		own(1, "#idea #avocar надо попробовать"),
		channelFwd(2, "New model released #ai #news", "Chan", 77),
	}, 1<<20)
	if len(ds) != 1 {
		t.Fatalf("got %d notes, want 1: %+v", len(ds), ds)
	}
	d := ds[0]
	if d.Type != "idea" || strings.Join(d.Projects, ",") != "avocar" {
		t.Errorf("tags = %q %v, want idea [avocar]", d.Type, d.Projects)
	}
	if d.Source == nil || d.Source.URL != "https://t.me/chan/77" {
		t.Errorf("source = %+v", d.Source)
	}
	if !strings.Contains(d.Text, "New model released") || !strings.Contains(d.Text, "Комментарий: #idea #avocar") {
		t.Errorf("text = %q", d.Text)
	}
	if d.ClientKey != "chat:5:2" {
		t.Errorf("key = %q", d.ClientKey)
	}
}

func TestAForwardAloneTakesNoTagsFromThePost(t *testing.T) {
	ds := Build(5, []*models.Message{channelFwd(2, "#idea in someone else's words", "Chan", 1)}, 1<<20)
	if len(ds) != 1 || ds[0].Type != "" || len(ds[0].Projects) != 0 {
		t.Errorf("got %+v", ds)
	}
}

func TestAlbumIsOneNoteWithItsCaption(t *testing.T) {
	p := func(id int, cap string) *models.Message {
		return &models.Message{ID: id, MediaGroupID: "g1", Caption: cap,
			Photo: []models.PhotoSize{{FileID: "f", FileUniqueID: string(rune('a' + id)), FileSize: 100}}}
	}
	ds := Build(5, []*models.Message{p(1, ""), p(2, "#reflection доска после созвона"), p(3, "")}, 1<<20)
	if len(ds) != 1 {
		t.Fatalf("got %d notes, want 1", len(ds))
	}
	if len(ds[0].Media) != 3 || ds[0].Type != "reflection" {
		t.Errorf("got %+v", ds[0])
	}
}

func TestSeparateThoughtsStaySeparate(t *testing.T) {
	ds := Build(5, []*models.Message{own(1, "#idea one"), own(2, "#note two")}, 1<<20)
	if len(ds) != 2 || ds[0].Type != "idea" || ds[1].Type != "note" {
		t.Errorf("got %+v", ds)
	}
}

func TestPostsFromTwoChannelsAreTwoNotesSharingTheComment(t *testing.T) {
	ds := Build(5, []*models.Message{
		own(1, "#link"),
		channelFwd(2, "a", "One", 1), channelFwd(3, "b", "One", 2),
		channelFwd(4, "c", "Two", 9),
	}, 1<<20)
	if len(ds) != 2 {
		t.Fatalf("got %d notes, want 2", len(ds))
	}
	if !strings.Contains(ds[0].Text, "a\n\n---\n\nb") || ds[0].Type != "link" || ds[1].Type != "link" {
		t.Errorf("got %+v", ds)
	}
}

type fakeTimer struct{ stopped bool }

func (f *fakeTimer) Stop() bool { f.stopped = true; return true }

func TestCollectorWaitsForQuietThenFlushesOnce(t *testing.T) {
	var fire []func()
	var got [][]Draft
	c := &Collector{Window: time.Second, MaxBytes: 1 << 20,
		Flush: func(_ int64, ds []Draft) { got = append(got, ds) },
		AfterFunc: func(_ time.Duration, f func()) interface{ Stop() bool } {
			fire = append(fire, f)
			return &fakeTimer{}
		}}
	c.Add(5, own(1, "#idea #graph"))
	c.Add(5, channelFwd(2, "post", "Chan", 3))
	if len(got) != 0 {
		t.Fatal("flushed before the chat was quiet")
	}
	fire[len(fire)-1]()
	if len(got) != 1 || len(got[0]) != 1 || got[0][0].Type != "idea" {
		t.Fatalf("got %+v", got)
	}
	fire[0]() // the superseded timer, if it fires late, finds nothing
	if len(got) != 1 {
		t.Errorf("flushed twice: %+v", got)
	}
}

// A timer that fires after a newer message re-armed the chat must not take the
// batch: it would cut the window short and split a forward from its comment.
func TestAStaleTimerDoesNotFlushTheNewerBatch(t *testing.T) {
	var fire []func()
	var got [][]Draft
	c := &Collector{Window: time.Second, MaxBytes: 1 << 20,
		Flush: func(_ int64, ds []Draft) { got = append(got, ds) },
		AfterFunc: func(_ time.Duration, f func()) interface{ Stop() bool } {
			fire = append(fire, f)
			return &fakeTimer{}
		}}
	c.Add(5, own(1, "#idea #graph"))
	c.Add(5, channelFwd(2, "post", "Chan", 3))
	fire[0]() // the first timer's Stop() lost the race and it runs anyway
	if len(got) != 0 {
		t.Fatalf("stale timer flushed: %+v", got)
	}
	fire[1]()
	if len(got) != 1 || len(got[0]) != 1 || got[0][0].Type != "idea" {
		t.Fatalf("got %+v", got)
	}
}
