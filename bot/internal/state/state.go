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

	// Calendars is the last list read from the API, with the moment it was
	// read. Reused for a few seconds so one button press costs one request
	// instead of three (see calendarListTTL).
	Calendars   []api.CalendarDetail
	CalendarsAt time.Time

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
