package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/minight/minight-terminal/internal/config"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	t.Setenv("MAX_TIMEOUT_SECONDS", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load(): %v", err)
	}
	return cfg
}

func connectTestServer(t *testing.T) *mcp.ClientSession {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	server := BuildMCPServer(testConfig(t))
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func callToolJSON(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%q) error = %v", name, err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("CallTool(%q) content len = %d", name, len(res.Content))
	}
	text := res.Content[0].(*mcp.TextContent).Text
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", text, err)
	}
	return payload
}

func TestRunCommandPWD(t *testing.T) {
	session := connectTestServer(t)
	home, _ := os.UserHomeDir()

	payload := callToolJSON(t, session, "run_command", map[string]any{
		"command":    "pwd",
		"session_id": "handler-pwd",
		"cwd":        home,
	})
	if payload["return_code"].(float64) != 0 {
		t.Fatalf("return_code = %v", payload["return_code"])
	}
	if strings.TrimSpace(payload["stdout"].(string)) != home {
		t.Fatalf("stdout = %q, want %q", payload["stdout"], home)
	}
}

func TestGetSession(t *testing.T) {
	session := connectTestServer(t)
	home, _ := os.UserHomeDir()

	payload := callToolJSON(t, session, "get_session", map[string]any{
		"session_id": "default",
	})
	if payload["session_id"].(string) != "default" {
		t.Fatalf("session_id = %v", payload["session_id"])
	}
	if payload["current_cwd"].(string) != home {
		t.Fatalf("current_cwd = %v, want %q", payload["current_cwd"], home)
	}
}

func TestKillSession(t *testing.T) {
	session := connectTestServer(t)
	tmp := t.TempDir()

	callToolJSON(t, session, "run_command", map[string]any{
		"command":    "cd " + tmp,
		"session_id": "kill-me",
	})
	killPayload := callToolJSON(t, session, "kill_session", map[string]any{
		"session_id": "kill-me",
	})
	if killPayload["reset"].(bool) != true {
		t.Fatalf("reset = %v", killPayload["reset"])
	}

	sessionPayload := callToolJSON(t, session, "get_session", map[string]any{
		"session_id": "kill-me",
	})
	home, _ := os.UserHomeDir()
	if sessionPayload["current_cwd"].(string) != home {
		t.Fatalf("current_cwd after kill = %v, want %q", sessionPayload["current_cwd"], home)
	}
}

func TestListSessionsTool(t *testing.T) {
	session := connectTestServer(t)
	tmp := t.TempDir()

	callToolJSON(t, session, "run_command", map[string]any{
		"command":    "cd " + tmp,
		"session_id": "listed",
	})
	payload := callToolJSON(t, session, "list_sessions", map[string]any{})
	sessions, ok := payload["sessions"].([]any)
	if !ok || len(sessions) == 0 {
		t.Fatalf("sessions = %#v", payload["sessions"])
	}
}

func TestRunCommandRejectsEmptyCommand(t *testing.T) {
	session := connectTestServer(t)
	payload := callToolJSON(t, session, "run_command", map[string]any{
		"command": "",
	})
	if payload["error"].(string) != "command is required" {
		t.Fatalf("error = %v", payload["error"])
	}
}
