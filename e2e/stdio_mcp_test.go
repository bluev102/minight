package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func connectServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	root := projectRoot(t)
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "MAX_TIMEOUT_SECONDS=60")

	client := mcp.NewClient(&mcp.Implementation{Name: "e2e-client", Version: "0.1.0"}, nil)
	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func callToolJSON(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%q) error = %v", name, err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("invalid JSON from %q: %v (%q)", name, err, text)
	}
	return payload
}

func TestToolDiscovery(t *testing.T) {
	session := connectServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	names := make(map[string]struct{})
	for _, tool := range tools.Tools {
		names[tool.Name] = struct{}{}
	}
	for _, expected := range []string{"run_command", "get_session", "kill_session", "list_sessions"} {
		if _, ok := names[expected]; !ok {
			t.Fatalf("missing tool %q, got %#v", expected, names)
		}
	}
}

func TestRunCommandPWD(t *testing.T) {
	session := connectServer(t)
	home, _ := os.UserHomeDir()

	payload := callToolJSON(t, session, "run_command", map[string]any{
		"command":    "pwd",
		"session_id": "e2e-pwd",
		"cwd":        home,
	})
	if payload["return_code"].(float64) != 0 {
		t.Fatalf("return_code = %v", payload["return_code"])
	}
	if strings.TrimSpace(payload["stdout"].(string)) != home {
		t.Fatalf("stdout = %q, want %q", payload["stdout"], home)
	}
}

func TestSessionCWDAndEnvPersistence(t *testing.T) {
	session := connectServer(t)
	tmp := t.TempDir()

	first := callToolJSON(t, session, "run_command", map[string]any{
		"command":    "cd " + tmp + " && export MINIGHT_TEST=ok",
		"session_id": "e2e-session",
		"timeout":    10,
	})
	if first["return_code"].(float64) != 0 {
		t.Fatalf("first return_code = %v", first["return_code"])
	}

	second := callToolJSON(t, session, "run_command", map[string]any{
		"command":    "pwd && printenv MINIGHT_TEST",
		"session_id": "e2e-session",
		"timeout":    10,
	})
	if strings.TrimSpace(strings.Split(second["stdout"].(string), "\n")[0]) != tmp {
		t.Fatalf("cwd not persisted: %v", second["stdout"])
	}
	if !strings.Contains(second["stdout"].(string), "ok") {
		t.Fatalf("env not persisted: %v", second["stdout"])
	}
}

func TestRunCommandTimeout(t *testing.T) {
	session := connectServer(t)
	payload := callToolJSON(t, session, "run_command", map[string]any{
		"command":    "sleep 60",
		"session_id": "e2e-timeout",
		"timeout":    1,
	})
	if payload["timed_out"].(bool) != true {
		t.Fatalf("timed_out = %v", payload["timed_out"])
	}
}

func TestListSessions(t *testing.T) {
	session := connectServer(t)
	tmp := t.TempDir()

	callToolJSON(t, session, "run_command", map[string]any{
		"command":    "cd " + tmp,
		"session_id": "list-me",
	})

	payload := callToolJSON(t, session, "list_sessions", map[string]any{})
	sessions, ok := payload["sessions"].([]any)
	if !ok || len(sessions) == 0 {
		t.Fatalf("sessions = %#v", payload["sessions"])
	}
}

func TestKillSessionResetsCWD(t *testing.T) {
	session := connectServer(t)
	home, _ := os.UserHomeDir()
	tmp := t.TempDir()

	callToolJSON(t, session, "run_command", map[string]any{
		"command":    "cd " + tmp,
		"session_id": "e2e-kill",
	})
	callToolJSON(t, session, "kill_session", map[string]any{
		"session_id": "e2e-kill",
	})

	payload := callToolJSON(t, session, "get_session", map[string]any{
		"session_id": "e2e-kill",
	})
	if payload["current_cwd"].(string) != home {
		t.Fatalf("current_cwd = %v, want %q", payload["current_cwd"], home)
	}
}
