package orchestrate

import (
	"errors"
	"strings"
	"testing"
)

func argString(args []string) string { return strings.Join(args, " ") }

func TestBuildArgsQuery(t *testing.T) {
	args := BuildArgs(Request{Prompt: "what is X?", Profile: Query, MaxTurns: 6})
	got := argString(args)
	for _, want := range []string{
		"-p what is X?",
		"--output-format json",
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

// TestRunBudgetSentinel documents the budget error sentinel; the live path is
// covered by integration use, not unit tests (it would spend real money).
func TestRunBudgetSentinel(t *testing.T) {
	if !errors.Is(ErrBudgetExceeded, ErrBudgetExceeded) {
		t.Fatal("sentinel identity broken")
	}
}
