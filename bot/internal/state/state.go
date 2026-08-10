package state

import "sync"

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

// SetAuth stores the session obtained for this chat. Callers hold a pointer
// from GetOrCreate, but the write goes through the mutex so a notifier tick or
// a second update in flight cannot observe a half-written session.
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
