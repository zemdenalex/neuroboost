package inbox

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot/models"
)

// Telegram has no «end of batch» marker, and two things arrive as several
// updates that are one thought:
//   - a forward with a comment: the comment is its own message, sent next to
//     the forward (before it, in today's clients);
//   - an album: one update per picture, sharing media_group_id, the caption on
//     only one of them.
// So messages of a chat are held until it has been quiet for Window, and the
// whole batch is then turned into notes by Build.

// Window is how long a chat must be quiet before its batch becomes notes.
const Window = 1500 * time.Millisecond

// Draft is one note about to be saved.
type Draft struct {
	ClientKey string
	Type      string
	Projects  []string
	Text      string
	Source    *Source
	Media     []Media
}

// Build turns one quiet batch into notes:
//   - consecutive forwards from one place (or one album) are one note;
//   - Denis's own text in a batch that has forwards is their comment — tags
//     come from it, never from the forwarded post's own hashtags;
//   - without forwards, every own message (or own album) is its own note.
func Build(chatID int64, msgs []*models.Message, maxBytes int64) []Draft {
	type unit struct {
		msgs    []*models.Message
		forward bool
	}
	var units []unit
	for _, m := range msgs {
		fwd := m.ForwardOrigin != nil
		if n := len(units); n > 0 {
			last := &units[n-1]
			prev := last.msgs[len(last.msgs)-1]
			sameAlbum := m.MediaGroupID != "" && m.MediaGroupID == prev.MediaGroupID
			sameOrigin := fwd && last.forward && originKey(Origin(m.ForwardOrigin)) == originKey(Origin(prev.ForwardOrigin))
			if sameAlbum || sameOrigin {
				last.msgs = append(last.msgs, m)
				continue
			}
		}
		units = append(units, unit{msgs: []*models.Message{m}, forward: fwd})
	}

	hasForward := false
	for _, u := range units {
		if u.forward {
			hasForward = true
		}
	}

	var comments []string
	var drafts []Draft
	for _, u := range units {
		text, media := joinContent(u.msgs, maxBytes)
		if hasForward && !u.forward && len(media) == 0 {
			comments = append(comments, text)
			continue
		}
		d := Draft{
			ClientKey: fmt.Sprintf("chat:%d:%d", chatID, u.msgs[0].ID),
			Text:      text,
			Media:     media,
		}
		if u.forward {
			d.Source = Origin(u.msgs[0].ForwardOrigin)
		} else {
			d.Type, d.Projects = Tags(text)
		}
		drafts = append(drafts, d)
	}
	if comment := strings.TrimSpace(strings.Join(comments, "\n")); comment != "" {
		typ, projects := Tags(comment)
		for i := range drafts {
			if drafts[i].Source == nil {
				continue
			}
			drafts[i].Type, drafts[i].Projects = typ, projects
			drafts[i].Text = strings.TrimSpace(drafts[i].Text + "\n\nКомментарий: " + comment)
		}
	}
	return drafts
}

func joinContent(msgs []*models.Message, maxBytes int64) (string, []Media) {
	var parts []string
	var media []Media
	for _, m := range msgs {
		if c := strings.TrimSpace(Content(m)); c != "" {
			parts = append(parts, c)
		}
		media = append(media, MediaOf(m, maxBytes)...)
	}
	return strings.Join(parts, "\n\n---\n\n"), media
}

// Collector holds each chat's messages until the chat is quiet, then hands
// the batch's drafts to Flush.
type Collector struct {
	Window   time.Duration
	MaxBytes int64
	Flush    func(chatID int64, drafts []Draft)
	// AfterFunc is time.AfterFunc; tests replace it to fire by hand.
	AfterFunc func(time.Duration, func()) interface{ Stop() bool }

	mu      sync.Mutex
	pending map[int64][]*models.Message
	timers  map[int64]interface{ Stop() bool }
	// gen numbers each arming per chat. A timer whose Stop() lost the race
	// still runs; it takes the batch only if no newer message re-armed the chat.
	gen map[int64]uint64
}

func (c *Collector) Add(chatID int64, m *models.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending == nil {
		c.pending = map[int64][]*models.Message{}
		c.timers = map[int64]interface{ Stop() bool }{}
		c.gen = map[int64]uint64{}
	}
	c.pending[chatID] = append(c.pending[chatID], m)
	if t := c.timers[chatID]; t != nil {
		t.Stop()
	}
	after := c.AfterFunc
	if after == nil {
		after = func(d time.Duration, f func()) interface{ Stop() bool } { return time.AfterFunc(d, f) }
	}
	c.gen[chatID]++
	g := c.gen[chatID]
	c.timers[chatID] = after(c.Window, func() { c.fire(chatID, g) })
}

func (c *Collector) fire(chatID int64, g uint64) {
	// A timer callback is its own goroutine; a panic here would end the
	// process (gotcha 9).
	defer func() {
		if r := recover(); r != nil {
			log.Printf("inbox: flush panic: %v", r)
		}
	}()
	c.mu.Lock()
	if c.gen[chatID] != g {
		c.mu.Unlock()
		return
	}
	msgs := c.pending[chatID]
	delete(c.pending, chatID)
	delete(c.timers, chatID)
	c.mu.Unlock()
	if len(msgs) == 0 {
		return
	}
	c.Flush(chatID, Build(chatID, msgs, c.MaxBytes))
}
