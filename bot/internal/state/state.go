package state

import "sync"

type UserState struct {
	ChatID      int64
	AuthToken   string
	CurrentFlow string
	FlowStep    string
	FlowData    map[string]interface{}
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
	u := &UserState{ChatID: chatID, FlowData: make(map[string]interface{})}
	s.users[chatID] = u
	return u
}

func (s *Store) ClearFlow(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[chatID]; ok {
		u.CurrentFlow = ""
		u.FlowStep = ""
		u.FlowData = make(map[string]interface{})
	}
}
