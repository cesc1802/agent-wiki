package orchestrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func argString(args []string) string { return strings.Join(args, " ") }

func TestBuildArgsQuery(t *testing.T) {
	args := BuildArgs(Request{Prompt: "what is X?", Profile: Query, MaxTurns: 6})
	got := argString(args)
	for _, want := range []string{
		"-p what is X?",
		"--output-format stream-json",
		"--verbose",
		"--max-turns 6",
		"--allowedTools Read,Glob,Grep",
		"--disallowedTools Write,Edit,MultiEdit,Bash,WebFetch,WebSearch",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("args missing %q\n full: %s", want, got)
		}
	}
	if strings.Contains(got, "--permission-mode") {
		t.Error("query profile should not set a permission mode")
	}
}

func TestBuildArgsIngest(t *testing.T) {
	args := BuildArgs(Request{Prompt: "ingest plan", Profile: Ingest})
	got := argString(args)
	if !strings.Contains(got, "--permission-mode acceptEdits") {
		t.Errorf("ingest should accept edits, got: %s", got)
	}
	if !strings.Contains(got, "--allowedTools Read,Glob,Grep,Write,Edit,MultiEdit") {
		t.Errorf("ingest allowed tools wrong, got: %s", got)
	}
	if !strings.Contains(got, "--disallowedTools Bash,WebFetch,WebSearch") {
		t.Errorf("ingest must still disallow Bash, got: %s", got)
	}
	if strings.Contains(got, "--max-turns") {
		t.Error("no max-turns flag expected when MaxTurns is 0")
	}
}

func TestBothProfilesDisallowBash(t *testing.T) {
	for _, p := range []Profile{Query, Ingest} {
		found := false
		for _, d := range p.DisallowedTools {
			if d == "Bash" {
				found = true
			}
		}
		if !found {
			t.Errorf("profile %q must disallow Bash", p.Name)
		}
	}
}

func TestParseResult(t *testing.T) {
	out := `{"type":"result","subtype":"success","is_error":false,` +
		`"result":"The answer is 42.","session_id":"abc","total_cost_usd":0.0123,"num_turns":3}`
	res, err := ParseResult([]byte("\n" + out + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "The answer is 42." {
		t.Errorf("text = %q", res.Text)
	}
	if res.CostUSD != 0.0123 || res.NumTurns != 3 || res.IsError {
		t.Errorf("unexpected fields: %#v", res)
	}
	if res.SessionID != "abc" {
		t.Errorf("session id = %q", res.SessionID)
	}
}

func TestParseResultEmpty(t *testing.T) {
	if _, err := ParseResult([]byte("   \n")); err == nil {
		t.Fatal("expected error on empty output")
	}
}

func TestParseResultBadJSON(t *testing.T) {
	if _, err := ParseResult([]byte("not json")); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestHandleStreamLineAssistantToolUse(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"writing the page"},` +
		`{"type":"tool_use","name":"Write","input":{"file_path":"sources/plan.md"}}]}}`)
	var got []Activity
	rl := handleStreamLine(line, func(a Activity) { got = append(got, a) })
	if rl != nil {
		t.Errorf("assistant line should not be a result line, got %q", rl)
	}
	if len(got) != 1 || got[0].Tool != "Write" || got[0].Target != "sources/plan.md" {
		t.Errorf("unexpected activity: %#v", got)
	}
}

func TestHandleStreamLineResult(t *testing.T) {
	line := []byte(`{"type":"result","subtype":"success","is_error":false,"result":"done"}` + "\n")
	rl := handleStreamLine(line, func(Activity) { t.Fatal("result line must not emit activity") })
	res, err := ParseResult(rl)
	if err != nil {
		t.Fatalf("result line did not parse: %v", err)
	}
	if res.Text != "done" {
		t.Errorf("text = %q", res.Text)
	}
}

func TestHandleStreamLineIgnoresNoise(t *testing.T) {
	for _, line := range [][]byte{
		[]byte("   \n"),
		[]byte("not json"),
		[]byte(`{"type":"system","subtype":"init"}`),
	} {
		if rl := handleStreamLine(line, func(Activity) { t.Fatalf("noise emitted activity: %q", line) }); rl != nil {
			t.Errorf("noise treated as result: %q", line)
		}
	}
}

func TestBestTarget(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{`{"file_path":"a.md","content":"x"}`, "a.md"},
		{`{"pattern":"foo"}`, "foo"},
		{`{"query":"bar"}`, "bar"},
		{`{"unrelated":"z"}`, ""},
		{`{}`, ""},
	}
	for _, c := range cases {
		if got := bestTarget([]byte(c.input)); got != c.want {
			t.Errorf("bestTarget(%s) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, binary)
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}

	t.Setenv("PATH", dir)
	path, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve with claude on PATH: %v", err)
	}
	if path != bin {
		t.Errorf("Resolve path = %q, want %q", path, bin)
	}
	if !Available() {
		t.Error("Available should be true when claude is on PATH")
	}

	t.Setenv("PATH", "")
	if _, err := Resolve(); !errors.Is(err, ErrClaudeNotFound) {
		t.Errorf("Resolve with empty PATH error = %v, want ErrClaudeNotFound", err)
	}
	if Available() {
		t.Error("Available should be false when claude is absent")
	}
}

// TestRunBudgetSentinel documents the budget error sentinel; the live path is
// covered by integration use, not unit tests (it would spend real money).
func TestRunBudgetSentinel(t *testing.T) {
	if !errors.Is(ErrBudgetExceeded, ErrBudgetExceeded) {
		t.Fatal("sentinel identity broken")
	}
}
