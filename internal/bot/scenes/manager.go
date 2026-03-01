package scenes

import (
	"sync"

	tele "gopkg.in/telebot.v3"
)

type Scene interface {
	Start(c tele.Context) error
	Handle(c tele.Context) (done bool, err error)
}

type Manager struct {
	mu     sync.RWMutex
	active map[int64]Scene // tgUserId -> sceneName
}

func NewManager() *Manager {
	return &Manager{active: make(map[int64]Scene)}
}

func (m *Manager) Set(userID int64, scene Scene) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active[userID] = scene
}

func (m *Manager) Clear(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, userID)
}

func (m *Manager) Get(userID int64) (Scene, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.active[userID]
	return s, ok
}
