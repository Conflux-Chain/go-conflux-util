package alert

import "sync"

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
	mu          sync.Mutex
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

	old := m.allChannels[ch.Name()]
	m.allChannels[ch.Name()] = ch

	return old
}

func (m *Manager) Del(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.allChannels, name)
}

func (m *Manager) Channel(name string) (Channel, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch, ok := m.allChannels[name]
	return ch, ok
}

func (m *Manager) All(name string) (chs []Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, v := range m.allChannels {
		chs = append(chs, v)
	}

	return chs
}
