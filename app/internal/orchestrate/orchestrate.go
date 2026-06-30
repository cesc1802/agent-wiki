// Package orchestrate drives Claude Code in headless mode (`claude -p`). It
// builds the command line from a per-command permission profile, runs it inside
// a knowledge-base root (so the agent-control CLAUDE.md and the raw/ write hook
// apply), parses the JSON result, and enforces a cost ceiling.
//
// This package never edits page content itself; it only invokes the agent and
// reports what the agent did and what it cost.
package orchestrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// binary is the Claude Code executable name resolved from PATH.
const binary = "claude"

// Profile is a per-command permission policy passed to `claude -p`. The tool
// allow/deny lists are the first guard; the PreToolUse hook in the KB is the
// authoritative second guard for raw/ writes.
type Profile struct {
	Name            string
	AllowedTools    []string
	DisallowedTools []string
	// PermissionMode maps to --permission-mode (empty = default). "acceptEdits"
	// lets the agent apply file edits without an interactive prompt, which is
	// required in headless mode for write operations.
	PermissionMode string
}

// Query is the read-only profile: the agent may read and search but cannot
// write, edit, or shell out.
var Query = Profile{
	Name:            "query",
	AllowedTools:    []string{"Read", "Glob", "Grep"},
	DisallowedTools: []string{"Write", "Edit", "MultiEdit", "Bash", "WebFetch", "WebSearch"},
}

// Ingest is the write profile: the agent may read, search, and write pages.
// Bash and network tools stay disallowed; the hook confines writes to wiki/.
var Ingest = Profile{
	Name:            "ingest",
	AllowedTools:    []string{"Read", "Glob", "Grep", "Write", "Edit", "MultiEdit"},
	DisallowedTools: []string{"Bash", "WebFetch", "WebSearch"},
	PermissionMode:  "acceptEdits",
}

// Request is one headless agent invocation.
type Request struct {
	Prompt       string
	Profile      Profile
	WorkDir      string  // KB root; claude runs here so hooks + CLAUDE.md apply
	MaxTurns     int     // 0 = leave to claude's default
	MaxBudgetUSD float64 // 0 = no ceiling
}

// Result is the parsed outcome of a `claude -p --output-format json` run.
type Result struct {
	CostUSD   float64
	NumTurns  int
	IsError   bool
	Text      string // the agent's final result text
	SessionID string
	Raw       string // raw stdout, for diagnostics
}

// ErrBudgetExceeded is returned (alongside a populated Result) when a run's
// reported cost is above the request's MaxBudgetUSD.
var ErrBudgetExceeded = errors.New("run cost exceeded budget")

// ErrClaudeNotFound is returned when the claude executable is not on PATH.
var ErrClaudeNotFound = errors.New("claude executable not found on PATH (install Claude Code)")

// BuildArgs renders the claude CLI arguments for a request. It is pure so the
// command line can be asserted in tests without invoking claude.
func BuildArgs(req Request) []string {
	args := []string{"-p", req.Prompt, "--output-format", "json"}
	if req.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(req.MaxTurns))
	}
	if req.Profile.PermissionMode != "" {
		args = append(args, "--permission-mode", req.Profile.PermissionMode)
	}
	if len(req.Profile.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(req.Profile.AllowedTools, ","))
	}
	if len(req.Profile.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(req.Profile.DisallowedTools, ","))
	}
	return args
}

// claudeJSON mirrors the fields of the `--output-format json` envelope that we
// consume. Unknown fields are ignored.
type claudeJSON struct {
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
	IsError      bool    `json:"is_error"`
	Result       string  `json:"result"`
	SessionID    string  `json:"session_id"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	NumTurns     int     `json:"num_turns"`
}

// ParseResult parses the JSON envelope emitted by `claude -p --output-format
// json`.
func ParseResult(stdout []byte) (*Result, error) {
	trimmed := bytes.TrimSpace(stdout)
	if len(trimmed) == 0 {
		return nil, errors.New("empty output from claude")
	}
	var cj claudeJSON
	if err := json.Unmarshal(trimmed, &cj); err != nil {
		return nil, fmt.Errorf("parse claude JSON output: %w", err)
	}
	return &Result{
		CostUSD:   cj.TotalCostUSD,
		NumTurns:  cj.NumTurns,
		IsError:   cj.IsError,
		Text:      cj.Result,
		SessionID: cj.SessionID,
		Raw:       string(trimmed),
	}, nil
}

// Available reports whether the claude executable is on PATH.
func Available() bool {
	_, err := exec.LookPath(binary)
	return err == nil
}

// Run invokes claude headlessly for the request and returns the parsed result.
// It returns ErrClaudeNotFound when claude is absent, and ErrBudgetExceeded
// (with a non-nil Result) when the run cost exceeds MaxBudgetUSD.
func Run(ctx context.Context, req Request) (*Result, error) {
	if !Available() {
		return nil, ErrClaudeNotFound
	}
	cmd := exec.CommandContext(ctx, binary, BuildArgs(req)...)
	cmd.Dir = req.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	res, parseErr := ParseResult(stdout.Bytes())
	if parseErr != nil {
		if runErr != nil {
			return nil, fmt.Errorf("claude run failed: %w; stderr: %s", runErr, strings.TrimSpace(stderr.String()))
		}
		return nil, parseErr
	}
	if runErr != nil && !res.IsError {
		// Non-zero exit but parseable result: surface the exit error.
		return res, fmt.Errorf("claude exited with error: %w; stderr: %s", runErr, strings.TrimSpace(stderr.String()))
	}
	if req.MaxBudgetUSD > 0 && res.CostUSD > req.MaxBudgetUSD {
		return res, fmt.Errorf("%w: spent $%.4f, ceiling $%.4f", ErrBudgetExceeded, res.CostUSD, req.MaxBudgetUSD)
	}
	return res, nil
}
