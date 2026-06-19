package session

import (
	"os"
	"sync"
)

const DefaultSessionID = "default"

type State struct {
	ID  string
	CWD string
	Env map[string]string
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*State
	homeDir  string
}

func NewManager() *Manager {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	return &Manager{
		sessions: make(map[string]*State),
		homeDir:  home,
	}
}

func normalizeID(id string) string {
	if id == "" {
		return DefaultSessionID
	}
	return id
}

func (m *Manager) Get(id string) State {
	id = normalizeID(id)

	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		s = &State{
			ID:  id,
			CWD: m.homeDir,
			Env: make(map[string]string),
		}
		m.sessions[id] = s
	}
	return cloneState(s)
}

func (m *Manager) Update(id string, cwd string, env map[string]string) State {
	id = normalizeID(id)

	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		s = &State{
			ID:  id,
			CWD: m.homeDir,
			Env: make(map[string]string),
		}
		m.sessions[id] = s
	}

	if cwd != "" {
		s.CWD = cwd
	}
	if env != nil {
		s.Env = cloneEnv(env)
	}
	return cloneState(s)
}

func (m *Manager) Kill(id string) {
	id = normalizeID(id)

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

func cloneState(s *State) State {
	return State{
		ID:  s.ID,
		CWD: s.CWD,
		Env: cloneEnv(s.Env),
	}
}

func cloneEnv(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
