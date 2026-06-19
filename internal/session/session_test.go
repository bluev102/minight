package session

import (
	"os"
	"testing"
)

func TestGetCreatesDefaultSession(t *testing.T) {
	m := NewManager()
	home, _ := os.UserHomeDir()

	state := m.Get("")
	if state.ID != DefaultSessionID {
		t.Fatalf("ID = %q, want %q", state.ID, DefaultSessionID)
	}
	if state.CWD != home {
		t.Fatalf("CWD = %q, want %q", state.CWD, home)
	}
	if state.Env == nil {
		t.Fatal("Env should not be nil")
	}
}

func TestGetAutoCreatesUnknownSession(t *testing.T) {
	m := NewManager()
	state := m.Get("project-a")
	if state.ID != "project-a" {
		t.Fatalf("ID = %q, want project-a", state.ID)
	}
}

func TestUpdateCWDAndEnv(t *testing.T) {
	m := NewManager()
	updated := m.Update("default", "/tmp", map[string]string{"FOO": "bar"}, UpdateMeta{
		LastCommand: "export FOO=bar",
		ReturnCode:  0,
	})

	if updated.CWD != "/tmp" {
		t.Fatalf("CWD = %q, want /tmp", updated.CWD)
	}
	if updated.Env["FOO"] != "bar" {
		t.Fatalf("Env[FOO] = %q, want bar", updated.Env["FOO"])
	}
	if updated.LastCommand != "export FOO=bar" {
		t.Fatalf("LastCommand = %q", updated.LastCommand)
	}

	got := m.Get("default")
	if got.CWD != "/tmp" {
		t.Fatalf("Get().CWD = %q, want /tmp", got.CWD)
	}
}

func TestKillRemovesSession(t *testing.T) {
	m := NewManager()
	home, _ := os.UserHomeDir()

	m.Update("default", "/tmp", map[string]string{"FOO": "bar"}, UpdateMeta{})
	m.Kill("default")

	state := m.Get("default")
	if state.CWD != home {
		t.Fatalf("after kill CWD = %q, want home %q", state.CWD, home)
	}
	if _, ok := state.Env["FOO"]; ok {
		t.Fatal("env should be reset after kill")
	}
}

func TestEnvMapCopyIsolation(t *testing.T) {
	m := NewManager()
	env := map[string]string{"FOO": "bar"}
	m.Update("default", "", env, UpdateMeta{})
	env["FOO"] = "changed"

	got := m.Get("default")
	if got.Env["FOO"] != "bar" {
		t.Fatalf("Env[FOO] = %q, want bar", got.Env["FOO"])
	}
}

func TestListAndInfo(t *testing.T) {
	m := NewManager()
	m.Update("alpha", "/tmp", map[string]string{"A": "1"}, UpdateMeta{LastCommand: "cd /tmp"})
	m.Update("beta", "/var", nil, UpdateMeta{})

	sessions := m.List()
	if len(sessions) != 2 {
		t.Fatalf("List() len = %d, want 2", len(sessions))
	}

	info := m.Info("alpha")
	if info.CWD != "/tmp" || info.EnvKeyCount != 1 || info.LastCommand != "cd /tmp" {
		t.Fatalf("Info() = %+v", info)
	}
}

func TestMergeBackgroundPIDs(t *testing.T) {
	m := NewManager()
	m.Update("bg", "/tmp", nil, UpdateMeta{BackgroundPIDs: []int{100, 101}})
	m.Update("bg", "/tmp", nil, UpdateMeta{BackgroundPIDs: []int{101, 102}})

	got := m.Get("bg")
	if len(got.BackgroundPIDs) != 3 {
		t.Fatalf("BackgroundPIDs = %v, want 3 unique pids", got.BackgroundPIDs)
	}
}
