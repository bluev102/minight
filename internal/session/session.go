package session

import (
	"os"
	"sync"
)

const DefaultSessionID = "default"

type State struct {
	ID               string
	CWD              string
	Env              map[string]string
	LastCommand      string
	BackgroundPIDs   []int
	EnvKeyCount      int
	LastReturnCode   int
	LastHadFailure   bool
}

type SessionInfo struct {
	ID             string
	CWD            string
	EnvKeyCount    int
	LastCommand    string
	BackgroundJobs int
	LastReturnCode int
	LastHadFailure bool
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

	s := m.getOrCreateLocked(id)
	return cloneState(s)
}

func (m *Manager) getOrCreateLocked(id string) *State {
	s, ok := m.sessions[id]
	if !ok {
		s = &State{
			ID:  id,
			CWD: m.homeDir,
			Env: make(map[string]string),
		}
		m.sessions[id] = s
	}
	return s
}

func (m *Manager) Update(id string, cwd string, env map[string]string, meta UpdateMeta) State {
	id = normalizeID(id)

	m.mu.Lock()
	defer m.mu.Unlock()

	s := m.getOrCreateLocked(id)

	if cwd != "" {
		s.CWD = cwd
	}
	if env != nil {
		s.Env = cloneEnv(env)
	}
	if meta.LastCommand != "" {
		s.LastCommand = meta.LastCommand
	}
	if len(meta.BackgroundPIDs) > 0 {
		s.BackgroundPIDs = mergePIDs(s.BackgroundPIDs, meta.BackgroundPIDs)
	}
	s.EnvKeyCount = len(s.Env)
	s.LastReturnCode = meta.ReturnCode
	s.LastHadFailure = meta.HadFailure

	return cloneState(s)
}

type UpdateMeta struct {
	LastCommand    string
	BackgroundPIDs []int
	ReturnCode     int
	HadFailure     bool
}

func (m *Manager) List() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, sessionInfoFromState(s))
	}
	return out
}

func (m *Manager) Info(id string) SessionInfo {
	id = normalizeID(id)

	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[id]
	if !ok {
		return SessionInfo{ID: id, CWD: m.homeDir}
	}
	return sessionInfoFromState(s)
}

func sessionInfoFromState(s *State) SessionInfo {
	return SessionInfo{
		ID:             s.ID,
		CWD:            s.CWD,
		EnvKeyCount:    len(s.Env),
		LastCommand:    s.LastCommand,
		BackgroundJobs: len(s.BackgroundPIDs),
		LastReturnCode: s.LastReturnCode,
		LastHadFailure: s.LastHadFailure,
	}
}

func (m *Manager) BackgroundPIDs(id string) []int {
	id = normalizeID(id)

	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[id]
	if !ok {
		return nil
	}
	return append([]int(nil), s.BackgroundPIDs...)
}

func (m *Manager) Kill(id string) []int {
	id = normalizeID(id)

	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		return nil
	}
	pids := append([]int(nil), s.BackgroundPIDs...)
	delete(m.sessions, id)
	return pids
}

func mergePIDs(existing, incoming []int) []int {
	seen := make(map[int]struct{}, len(existing))
	out := append([]int(nil), existing...)
	for _, pid := range existing {
		seen[pid] = struct{}{}
	}
	for _, pid := range incoming {
		if pid <= 0 {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		out = append(out, pid)
	}
	return out
}

func cloneState(s *State) State {
	return State{
		ID:             s.ID,
		CWD:            s.CWD,
		Env:            cloneEnv(s.Env),
		LastCommand:    s.LastCommand,
		BackgroundPIDs: append([]int(nil), s.BackgroundPIDs...),
		EnvKeyCount:    len(s.Env),
		LastReturnCode: s.LastReturnCode,
		LastHadFailure: s.LastHadFailure,
	}
}

func cloneEnv(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
