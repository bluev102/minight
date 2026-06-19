package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/minight/minight-terminal/internal/config"
	"github.com/minight/minight-terminal/internal/runner"
	"github.com/minight/minight-terminal/internal/session"
)

type RunCommandInput struct {
	Command   string `json:"command"`
	SessionID string `json:"session_id,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
	CWD       string `json:"cwd,omitempty"`
	Verbose   bool   `json:"verbose,omitempty"`
}

type SessionInput struct {
	SessionID string `json:"session_id,omitempty"`
}

type JSONResult struct {
	Stdout             string `json:"stdout"`
	Stderr             string `json:"stderr"`
	ReturnCode         int    `json:"return_code"`
	TimedOut           bool   `json:"timed_out"`
	CurrentCWD         string `json:"current_cwd"`
	Truncated          bool   `json:"truncated"`
	DurationMS         int64  `json:"duration_ms,omitempty"`
	StdoutOmittedBytes int    `json:"stdout_omitted_bytes,omitempty"`
	StderrOmittedBytes int    `json:"stderr_omitted_bytes,omitempty"`
	SessionID          string `json:"session_id,omitempty"`
	EnvChangedCount    int    `json:"env_changed_count,omitempty"`
	Error              string `json:"error,omitempty"`
}

type GetSessionResult struct {
	SessionID  string `json:"session_id"`
	CurrentCWD string `json:"current_cwd"`
}

type KillSessionResult struct {
	SessionID string `json:"session_id"`
	Reset     bool   `json:"reset"`
}

type Server struct {
	cfg     config.Config
	session *session.Manager
	runner  *runner.Runner
}

func New(cfg config.Config) *Server {
	sessions := session.NewManager()
	return &Server{
		cfg:     cfg,
		session: sessions,
		runner:  runner.New(cfg, sessions),
	}
}

func (s *Server) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "run_command",
		Description: "Execute a shell command with session-aware cwd and environment persistence.",
	}, s.handleRunCommand)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_session",
		Description: "Return the current working directory for a session.",
	}, s.handleGetSession)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "kill_session",
		Description: "Reset a session to its initial state.",
	}, s.handleKillSession)
}

func (s *Server) handleRunCommand(ctx context.Context, req *mcp.CallToolRequest, input RunCommandInput) (*mcp.CallToolResult, any, error) {
	if input.Command == "" {
		return jsonToolResult(JSONResult{
			ReturnCode: 1,
			Error:      "command is required",
		}, true)
	}

	timeout := s.cfg.DefaultTimeout
	if input.Timeout > 0 {
		timeout = time.Duration(input.Timeout) * time.Second
	}

	resp := s.runner.Run(ctx, runner.Request{
		Command:   input.Command,
		SessionID: input.SessionID,
		Timeout:   timeout,
		CWD:       input.CWD,
		Verbose:   input.Verbose,
	})

	isError := resp.Error != "" || resp.ReturnCode != 0 || resp.TimedOut
	return jsonToolResult(toJSONResult(resp), isError)
}

func (s *Server) handleGetSession(_ context.Context, _ *mcp.CallToolRequest, input SessionInput) (*mcp.CallToolResult, any, error) {
	state := s.session.Get(input.SessionID)
	return jsonToolResult(GetSessionResult{
		SessionID:  state.ID,
		CurrentCWD: state.CWD,
	}, false)
}

func (s *Server) handleKillSession(_ context.Context, _ *mcp.CallToolRequest, input SessionInput) (*mcp.CallToolResult, any, error) {
	id := input.SessionID
	if id == "" {
		id = session.DefaultSessionID
	}
	s.session.Kill(id)
	return jsonToolResult(KillSessionResult{
		SessionID: id,
		Reset:     true,
	}, false)
}

func toJSONResult(resp runner.Response) JSONResult {
	return JSONResult{
		Stdout:             resp.Stdout,
		Stderr:             resp.Stderr,
		ReturnCode:         resp.ReturnCode,
		TimedOut:           resp.TimedOut,
		CurrentCWD:         resp.CurrentCWD,
		Truncated:          resp.Truncated,
		DurationMS:         resp.DurationMS,
		StdoutOmittedBytes: resp.StdoutOmittedBytes,
		StderrOmittedBytes: resp.StderrOmittedBytes,
		SessionID:          resp.SessionID,
		EnvChangedCount:    resp.EnvChangedCount,
		Error:              resp.Error,
	}
}

func jsonToolResult(payload any, isError bool) (*mcp.CallToolResult, any, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal tool result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(raw)},
		},
		IsError: isError,
	}, payload, nil
}

func BuildMCPServer(cfg config.Config) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "minight-terminal",
		Version: "0.1.0",
	}, nil)
	New(cfg).Register(server)
	return server
}
