package state

import (
	"sync"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

type UserState struct {
	ChatID    int64
	AuthToken string
	// AuthExpiresAt is Unix *seconds*, matching the API's expires_at. Compare
	// it against time.Now().Unix() and nothing else — the web client's
	// long-standing bug was comparing this same field to a millisecond clock.
	AuthExpiresAt int64
	CurrentFlow   string
	FlowStep      string
	FlowData      map[string]any

	// Lang is the interface language, cached for this chat.
	//
	// 🔴 Cached, because every rendered screen needs it and reading it from
	// the API per message would add an HTTP round trip to every keypress —
	// parseIntoDraft already makes three. LangKnown is what tells a cached
	// answer from an unset one, since "" is a valid zero value and would
	// otherwise mean "re-read on every message".
	Lang      string
	LangKnown bool
	// LangAt is when Lang was read or set. After a few minutes it is read
	// again: the language can change in the web (one language per person,
	// Denis 26.09), and a process-long cache kept the chat on the old one
	// until the next manual redeploy.
	LangAt time.Time

	// TZ is the user's IANA timezone, cached on the same terms as Lang.
	TZ      string
	TZKnown bool

	// PersonalCalendar is the name of the calendar a new item lands in when
	// none was named, cached on the same terms as Lang and TZ.
	//
	// 🔴 Cached because every card prints it. Denis, 18.09: a card saying
	// «Календарь: нет» for an event that will land in «Личный» is not merely
	// unhelpful, it teaches that the field does not work.
	PersonalCalendar      string
	PersonalCalendarKnown bool

	// QuickTaskID and QuickRaw are the last task a typed line became, and
	// that line. «📅 → Событие» / «📝 → Заметка» under it re-read the LINE,
	// not the saved title: «купить молоко завтра» lost «завтра» to the due
	// date, and an event needs it back. Kept outside FlowData on purpose —
	// saving clears the flow, and these must outlive it.
	QuickTaskID string
	QuickRaw    string

	// Calendars is the last list read from the API, with the moment it was
	// read. Reused for a few seconds so one button press costs one request
	// instead of three (see calendarListTTL).
	Calendars   []api.CalendarDetail
	CalendarsAt time.Time

	// PriorityStyle is settings.bot.priority_style, cached like Lang: every
	// task list draws it (spec 21.09 §B). "" means «not chosen» — circles.
	PriorityStyle      string
	PriorityStyleKnown bool

	// DayTasksOn caches settings.day_tasks_enabled (spec 2026-09-22 §11): the
	// home keyboard draws it on every screen, so it is not read each time.
	DayTasksOn    bool
	DayTasksKnown bool
	// DayTasksAt is when DayTasksOn was read: the web switches it too.
	DayTasksAt time.Time
	// CalendarCell caches settings.bot.calendar_cell (spec §6): what a month
	// cell shows, "both" / "colour" / "bar". Read on every month page.
	CalendarCell      string
	CalendarCellKnown bool
	// DayPrefsFailedAt is when the last settings read failed: for a short
	// while after it the default is used without asking again (review I3).
	DayPrefsFailedAt time.Time
	// DayTasksAskDone: the one-time day-tasks question needs no more reads.
	DayTasksAskDone bool

	// Onboarded caches bot.onboarded once it is known to be true. False means
	// "not known yet", never "known false" — that one is always re-read, so a
	// user who finishes onboarding on the web or another device is not asked
	// again by a stale cache.
	Onboarded bool
}

type Store struct {
	mu    sync.RWMutex
	users map[int64]*UserState
}

func NewStore() *Store {
	return &Store{users: make(map[int64]*UserState)}
}

func (s *Store) Get(chatID int64) *UserState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[chatID]
}

func (s *Store) GetOrCreate(chatID int64) *UserState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[chatID]; ok {
		return u
	}
	u := &UserState{ChatID: chatID, FlowData: make(map[string]any)}
	s.users[chatID] = u
	return u
}

// SetAuth stores the session obtained for this chat.
//
// 🔴 The mutex protects the MAP, not the structs in it. GetOrCreate hands out a
// pointer and callers write through it without any lock — handlers/events.go
// sets CurrentFlow, FlowStep and FlowData directly, and handler.go reads
// AuthToken the same way. Taking the lock here does not make those safe.
//
// This is sound today for one reason only: there is exactly one writer. The
// update loop in cmd/main.go handles updates sequentially in a single
// goroutine, and the notifier never receives the store — see the signature of
// notifier.Start, which takes no *Store.
//
// 🔴 So the first `go h.HandleMessage(…)` added for parallelism turns this into
// a real data race without a single line changing in this file. If that day
// comes, close the fields behind methods (SetFlow, SetFlowData, Snapshot)
// rather than adding more locking here.
//
// The previous comment claimed the mutex made concurrent observation safe. It
// did not, and a comment that promises a guarantee the code lacks is worse than
// none — this repository has already lost a session to one.
func (s *Store) SetAuth(chatID int64, token string, expiresAt int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[chatID]
	if !ok {
		u = &UserState{ChatID: chatID, FlowData: make(map[string]any)}
		s.users[chatID] = u
	}
	u.AuthToken = token
	u.AuthExpiresAt = expiresAt
}

func (s *Store) ClearFlow(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[chatID]; ok {
		u.CurrentFlow = ""
		u.FlowStep = ""
		u.FlowData = make(map[string]any)
	}
}

// SetLang records the chat's interface language and when it was known.
func (s *UserState) SetLang(lang string) {
	s.Lang, s.LangKnown, s.LangAt = lang, true, time.Now()
}
