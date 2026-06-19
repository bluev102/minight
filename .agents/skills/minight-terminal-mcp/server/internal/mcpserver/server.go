package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/minight/minight-terminal/internal/backend"
	"github.com/minight/minight-terminal/internal/config"
	"github.com/minight/minight-terminal/internal/runner"
	"github.com/minight/minight-terminal/internal/session"
)

type RunCommandInput struct {
	Command        string `json:"command"`
	SessionID      string `json:"session_id,omitempty"`
	Timeout        int    `json:"timeout,omitempty"`
	CWD            string `json:"cwd,omitempty"`
	Verbose        bool   `json:"verbose,omitempty"`
	FailOnAnyError bool   `json:"fail_on_any_error,omitempty"`
	Pipefail       bool   `json:"pipefail,omitempty"`
	StripCRLF      *bool  `json:"strip_crlf,omitempty"`
}

type SessionInput struct {
	SessionID               string `json:"session_id,omitempty"`
	TerminateBackgroundJobs bool   `json:"terminate_background_jobs,omitempty"`
}

type JSONResult struct {
	Stdout              string   `json:"stdout"`
	Stderr              string   `json:"stderr"`
	ReturnCode          int      `json:"return_code"`
	TimedOut            bool     `json:"timed_out"`
	CurrentCWD          string   `json:"current_cwd"`
	Truncated           bool     `json:"truncated"`
	DurationMS          int64    `json:"duration_ms,omitempty"`
	StdoutOmittedBytes  int      `json:"stdout_omitted_bytes,omitempty"`
	StderrOmittedBytes  int      `json:"stderr_omitted_bytes,omitempty"`
	StdoutTotalBytes    int      `json:"stdout_total_bytes,omitempty"`
	StderrTotalBytes    int      `json:"stderr_total_bytes,omitempty"`
	SessionID           string   `json:"session_id,omitempty"`
	EnvChangedCount     int      `json:"env_changed_count,omitempty"`
	HadFailure          bool     `json:"had_failure,omitempty"`
	CWDPersisted        bool     `json:"cwd_persisted,omitempty"`
	EnvironmentWarnings []string `json:"environment_warnings,omitempty"`
	SuggestedTimeout    int      `json:"suggested_timeout,omitempty"`
	Error               string   `json:"error,omitempty"`
}

type GetSessionResult struct {
	SessionID      string `json:"session_id"`
	CurrentCWD     string `json:"current_cwd"`
	EnvKeyCount    int    `json:"env_key_count,omitempty"`
	LastCommand    string `json:"last_command,omitempty"`
	BackgroundJobs int    `json:"background_jobs,omitempty"`
	LastReturnCode int    `json:"last_return_code,omitempty"`
	LastHadFailure bool   `json:"last_had_failure,omitempty"`
}

type ListSessionsResult struct {
	Sessions []session.SessionInfo `json:"sessions"`
}

type KillSessionResult struct {
	SessionID              string `json:"session_id"`
	Reset                  bool   `json:"reset"`
	BackgroundJobsKilled   int    `json:"background_jobs_killed,omitempty"`
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
		Name: "run_command",
		Description: "Execute a shell command with session-aware cwd and environment persistence. " +
			"Missing session_id defaults to \"default\". timeout defaults to DEFAULT_TIMEOUT_SECONDS (or 30s); " +
			"0 uses the configured default. return_code is the shell exit code of the full command string; " +
			"use fail_on_any_error or had_failure for earlier failures in ; chains.",
	}, s.handleRunCommand)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_session",
		Description: "Return session metadata including current cwd. Missing session_id defaults to \"default\". " +
			"Environment values are not returned; only env_key_count is exposed.",
	}, s.handleGetSession)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_sessions",
		Description: "List active sessions with cwd and safe metadata. Environment values are not returned.",
	}, s.handleListSessions)
	mcp.AddTool(server, &mcp.Tool{
		Name: "kill_session",
		Description: "Reset a session to its initial state. Missing session_id defaults to \"default\". " +
			"Set terminate_background_jobs=true to kill tracked background PIDs from prior commands.",
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
		Command:        input.Command,
		SessionID:      input.SessionID,
		Timeout:        timeout,
		CWD:            input.CWD,
		Verbose:        input.Verbose,
		FailOnAnyError: input.FailOnAnyError,
		Pipefail:       input.Pipefail,
		StripCRLF:      input.StripCRLF,
	})

	isError := resp.Error != "" || resp.ReturnCode != 0 || resp.TimedOut
	return jsonToolResult(toJSONResult(resp), isError)
}

func (s *Server) handleGetSession(_ context.Context, _ *mcp.CallToolRequest, input SessionInput) (*mcp.CallToolResult, any, error) {
	info := s.session.Info(input.SessionID)
	return jsonToolResult(GetSessionResult{
		SessionID:      info.ID,
		CurrentCWD:     info.CWD,
		EnvKeyCount:    info.EnvKeyCount,
		LastCommand:    info.LastCommand,
		BackgroundJobs: info.BackgroundJobs,
		LastReturnCode: info.LastReturnCode,
		LastHadFailure: info.LastHadFailure,
	}, false)
}

func (s *Server) handleListSessions(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	return jsonToolResult(ListSessionsResult{Sessions: s.session.List()}, false)
}

func (s *Server) handleKillSession(_ context.Context, _ *mcp.CallToolRequest, input SessionInput) (*mcp.CallToolResult, any, error) {
	id := input.SessionID
	if id == "" {
		id = session.DefaultSessionID
	}

	var killed int
	if input.TerminateBackgroundJobs {
		pids := s.session.BackgroundPIDs(id)
		backend.KillBackgroundPIDs(pids)
		killed = len(pids)
	}

	s.session.Kill(id)

	return jsonToolResult(KillSessionResult{
		SessionID:            id,
		Reset:                true,
		BackgroundJobsKilled: killed,
	}, false)
}

func toJSONResult(resp runner.Response) JSONResult {
	return JSONResult{
		Stdout:              resp.Stdout,
		Stderr:              resp.Stderr,
		ReturnCode:          resp.ReturnCode,
		TimedOut:            resp.TimedOut,
		CurrentCWD:          resp.CurrentCWD,
		Truncated:           resp.Truncated,
		DurationMS:          resp.DurationMS,
		StdoutOmittedBytes:  resp.StdoutOmittedBytes,
		StderrOmittedBytes:  resp.StderrOmittedBytes,
		StdoutTotalBytes:    resp.StdoutTotalBytes,
		StderrTotalBytes:    resp.StderrTotalBytes,
		SessionID:           resp.SessionID,
		EnvChangedCount:     resp.EnvChangedCount,
		HadFailure:          resp.HadFailure,
		CWDPersisted:        resp.CWDPersisted,
		EnvironmentWarnings: resp.EnvironmentWarnings,
		SuggestedTimeout:    resp.SuggestedTimeout,
		Error:               resp.Error,
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
		Version: "0.2.0",
	}, nil)
	New(cfg).Register(server)
	return server
}
