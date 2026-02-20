package scenes

import (
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type SceneHandler func(api *tgbotapi.BotAPI, m *tgbotapi.Message) (done bool, err error)

type Manager struct {
	mu     sync.RWMutex
	active map[int64]string // tgUserId -> sceneName
}

func NewManager() *Manager {
	return &Manager{active: make(map[int64]string)}
}

func (m *Manager) Set(userID int64, scene string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active[userID] = scene
}

func (m *Manager) Clear(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, userID)
}

func (m *Manager) Get(userID int64) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.active[userID]
	return s, ok
}
