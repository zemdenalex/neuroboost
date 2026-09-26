// Package lineparse is POST /api/parse: one typed line, read exactly as the bot
// reads it.
//
// Denis, 26.09 (gap list row 1, «A: one parser, API endpoint»): a line typed on
// the web — «стоматолог завтра 15:00 напомни за час», «!1 30м #дом» — must mean
// what it means in the chat. The parser is the bot's own (bot/parse, pulled in
// by a go.mod replace), and so is the order it is applied in: nothing here
// reads a word; this package only fetches what the parser cannot know (the
// user's zone, calendars, own words, reminder presets) and shapes the answer.
package lineparse

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/zemdenalex/neuroboost-bot/parse"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

var db *database.DB

// InitDB follows the package-level-pool pattern the other packages use.
func InitDB(d *database.DB) { db = d }

// now is the clock, a variable so tests can fix «завтра».
var now = time.Now

// maxTextRunes caps the line. A quick-add line is a sentence; anything longer
// is a paste that belongs in the full editor, not in a parser run per keystroke.
const maxTextRunes = 1000

// Register puts the route on a router that already requires a JWT. main.go
// calls it inside the protected group, and so do the tests.
func Register(r chi.Router) {
	r.Post("/api/parse", Handler)
}

// Kind says what the web should do with the line.
type Kind string

const (
	// KindTask — no question needed: saved as a task at once, as the bot's
	// quick save does (a line with no clock time, not a list).
	KindTask Kind = "task"
	// KindEvent — a line with a clock time and nothing missing. The bot shows
	// its card here; the web shows a confirmation.
	KindEvent Kind = "event"
	// KindAsk — the bot would ask something first (Missing names what, or the
	// line is a list, or it names several dates). The web has no such dialog
	// yet and keeps its old behaviour for these.
	KindAsk Kind = "ask"
)

type request struct {
	Text     string  `json:"text"`
	Timezone *string `json:"timezone,omitempty"`
}

// Answer is the parsed draft. Every optional field is a pointer: «not stated»
// and «stated as zero» are different answers (priority 0 is Buffer; an empty
// reminder list means silent forever, a null one means the user's preset).
type Answer struct {
	Kind  Kind   `json:"kind"`
	Title string `json:"title"`
	// Timezone is the zone the line was read in, so the page prints wall
	// times in it rather than in the browser's.
	Timezone string `json:"timezone"`
	// Missing is what the bot's card would ask first ("title", "freq",
	// "date", "span", "time"), or "list" / "dates" for a line the bot asks
	// about as a whole. Empty unless Kind is "ask".
	Missing string `json:"missing,omitempty"`

	Tags  []string `json:"tags"`
	Rrule *string  `json:"rrule"`

	// Task fields.
	Priority         *int    `json:"priority"`
	DueDate          *string `json:"due_date"`
	EstimatedMinutes *int    `json:"estimated_minutes"`

	// Event fields, ready to send to POST /api/events as they are.
	StartsAt        *string `json:"starts_at"`
	EndsAt          *string `json:"ends_at"`
	AllDay          bool    `json:"all_day"`
	Color           *string `json:"color"`
	CalendarID      *string `json:"calendar_id"`
	CalendarName    *string `json:"calendar_name"`
	ReminderOffsets *[]int  `json:"reminder_offsets"`
	// IsTask: the word «задача» was used. The bot then creates a task and an
	// event bound to it by task_id; the web does the same.
	IsTask bool `json:"is_task"`
	// Uncertain names fields read from a loose format («13;00», a bare «12»).
	Uncertain []string `json:"uncertain"`
}

// Handler handles POST /api/parse.
func Handler(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		util.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "text is required")
		return
	}
	if utf8.RuneCountInString(text) > maxTextRunes {
		util.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "text is too long")
		return
	}

	ctx := r.Context()
	zone := userZone(ctx, userID)
	if req.Timezone != nil && strings.TrimSpace(*req.Timezone) != "" {
		zone = strings.TrimSpace(*req.Timezone)
		if _, err := time.LoadLocation(zone); err != nil {
			util.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "unknown timezone")
			return
		}
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		// A zone stored on the user that this build cannot load: the bot falls
		// back the same way rather than refusing to read the line.
		zone, loc = "UTC", time.UTC
	}

	util.RespondJSON(w, http.StatusOK, Read(text, now().In(loc), zone, vocabulary(ctx, userID)))
}

// Read is the whole decision, as a pure function of the line, the clock and
// the user's vocabulary. It mirrors the bot's handleQuickAdd: a plain task
// first, then the event card, and a question for everything else.
func Read(text string, at time.Time, zone string, v parse.Vocabulary) Answer {
	a := Answer{Timezone: zone, Tags: []string{}, Uncertain: []string{}}

	if r, ok := parse.PlainTask(text, at); ok {
		a.Kind, a.Title, a.Priority, a.EstimatedMinutes = KindTask, r.Title, r.Priority, r.EstimatedMinutes
		if r.DueDate != nil {
			a.DueDate = stamp(*r.DueDate)
		}
		if len(r.Tags) > 0 {
			a.Tags = r.Tags
		}
		if r.Rrule != "" {
			a.Rrule = &r.Rrule
		}
		return a
	}

	u := parse.Understand(text, at, v)
	d := u.Draft
	a.Title, a.AllDay, a.IsTask = u.Title, d.AllDay, d.IsTask
	if len(d.Tags) > 0 {
		a.Tags = d.Tags
	}
	if rule := d.RRule(); rule != "" {
		a.Rrule = &rule
	}
	if d.Colour != "" {
		a.Color = &d.Colour
	}
	if u.CalendarID != "" {
		a.CalendarID, a.CalendarName = &u.CalendarID, &u.CalendarName
	}
	a.ReminderOffsets = d.ReminderOffsets
	for _, f := range d.Uncertain {
		a.Uncertain = append(a.Uncertain, parse.FieldName(f))
	}

	switch {
	case parse.LooksLikeList(text, at):
		a.Kind, a.Missing = KindAsk, "list"
	case len(d.MoreDays) > 0:
		a.Kind, a.Missing = KindAsk, "dates"
	case parse.Missing(u.Title, d) != "":
		a.Kind, a.Missing = KindAsk, parse.Missing(u.Title, d)
	default:
		a.Kind = KindEvent
		start, end := parse.Bounds(d)
		a.StartsAt, a.EndsAt = stamp(start), stamp(end)
	}
	return a
}

func stamp(t time.Time) *string {
	s := t.UTC().Format(time.RFC3339)
	return &s
}

// userZone is the zone stored on the user — the one the bot reads through
// /api/auth/me — with the same fallback the rest of the API uses.
func userZone(ctx context.Context, userID string) string {
	tz := "Europe/Moscow"
	if db == nil {
		return tz
	}
	_ = db.Pool.QueryRow(ctx,
		`SELECT COALESCE(NULLIF(timezone, ''), 'Europe/Moscow') FROM "user" WHERE id = $1`, userID).Scan(&tz)
	return tz
}

// vocabulary is what the bot fetches before reading a line: calendars, own
// words, reminder presets. A read that fails leaves that part empty, exactly
// as in the bot — the line is still read, only without those words.
func vocabulary(ctx context.Context, userID string) parse.Vocabulary {
	var v parse.Vocabulary
	if cals, err := calendars.ListFor(ctx, userID); err == nil {
		for _, c := range cals {
			v.Calendars = append(v.Calendars, parse.Calendar{ID: c.ID, Name: c.Name})
		}
	}
	if db == nil {
		return v
	}
	var raw []byte
	if err := db.Pool.QueryRow(ctx,
		`SELECT COALESCE(settings, '{}') FROM "user" WHERE id = $1`, userID).Scan(&raw); err != nil {
		return v
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		return v
	}
	v.Keywords = parse.KeywordsFromSettings(settings)
	v.Presets = parse.PresetsFromSettings(settings)
	return v
}
