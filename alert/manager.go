package alert

import (
	"strings"
	"sync"
)

var (
	stdMgr *Manager
	oncer  sync.Once
)

func DefaultManager() *Manager {
	oncer.Do(func() {
		stdMgr = NewManager()
	})
	return stdMgr
}

type Manager struct {
	mu sync.Mutex
	// allChannels is a map that holds all the channels. The key is the channel ID and
	// the value is the channel itself. The channel ID is treated as case-insensitive.
	allChannels map[string]Channel
}

func NewManager() *Manager {
	return &Manager{
		allChannels: make(map[string]Channel),
	}
}

func (m *Manager) Add(ch Channel) Channel {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := strings.ToLower(ch.Name())
	old := m.allChannels[name]
	m.allChannels[name] = ch

	return old
}

func (m *Manager) Del(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.allChannels, strings.ToLower(name))
}

func (m *Manager) Channel(name string) (Channel, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch, ok := m.allChannels[strings.ToLower(name)]
	return ch, ok
}

func (m *Manager) All() (chs []Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, v := range m.allChannels {
		chs = append(chs, v)
	}

	return chs
}
