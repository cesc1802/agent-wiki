package scaffold

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"nvtwiki/internal/kb"
)

// runHook feeds a PreToolUse event JSON to the scaffolded block-raw-write.sh and
// returns its exit code and combined output.
func runHook(t *testing.T, hookPath, eventJSON string) (int, string) {
	t.Helper()
	cmd := exec.Command("bash", hookPath)
	cmd.Stdin = strings.NewReader(eventJSON)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return 0, string(out)
	}
	var exitErr *exec.ExitError
	if !asExit(err, &exitErr) {
		t.Fatalf("hook failed to run (need bash + jq): %v", err)
	}
	return exitErr.ExitCode(), string(out)
}

func asExit(err error, target **exec.ExitError) bool {
	if e, ok := err.(*exec.ExitError); ok {
		*target = e
		return true
	}
	return false
}

// TestRawWriteGuard is the adversarial check: no Write/Edit/MultiEdit may land
// outside a wiki/ directory. raw/ and arbitrary paths must be denied; wiki/
// paths and non-write tools must be allowed.
func TestRawWriteGuard(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed; skipping hook guard test")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available; skipping hook guard test")
	}

	dir := t.TempDir()
	if _, err := Init(dir, "2026-06-30"); err != nil {
		t.Fatalf("init: %v", err)
	}
	root := &kb.Root{Dir: dir}
	hookPath := filepath.Join(root.WikiDir(), "hooks", "block-raw-write.sh")

	rawPath := filepath.Join(root.RawDir(), "secret.md")
	wikiPath := filepath.Join(root.WikiDir(), "concepts", "x.md")
	indexPath := filepath.Join(root.WikiDir(), "index.md")

	denied := []struct {
		name, tool, path string
	}{
		{"write into raw", "Write", rawPath},
		{"edit into raw", "Edit", rawPath},
		{"multiedit into raw", "MultiEdit", rawPath},
		{"write to absolute escape", "Write", "/etc/passwd"},
		{"write to schema control file", "Write", root.SchemaPath()},
		{"write to CLAUDE.md control file", "Write", filepath.Join(root.WikiDir(), "CLAUDE.md")},
		{"write to .claude settings", "Write", filepath.Join(root.WikiDir(), ".claude", "settings.json")},
		{"write to hook script", "Write", hookPath},
		{"relative escape attempt", "Write", "../escape.md"},
		{"raw/wiki bypass attempt", "Write", filepath.Join(root.RawDir(), "wiki", "evil.md")},
	}
	for _, c := range denied {
		event := `{"tool_name":"` + c.tool + `","tool_input":{"file_path":"` + c.path + `"}}`
		code, out := runHook(t, hookPath, event)
		if code != 2 {
			t.Errorf("%s: expected deny (exit 2), got exit %d, out=%s", c.name, code, out)
		}
		if !strings.Contains(out, "deny") {
			t.Errorf("%s: expected deny decision in output, got %s", c.name, out)
		}
	}

	allowed := []struct {
		name, tool, path string
	}{
		{"write to wiki concept", "Write", wikiPath},
		{"edit wiki index", "Edit", indexPath},
		{"multiedit wiki", "MultiEdit", wikiPath},
		{"relative page write (cwd is wiki)", "Write", "concepts/y.md"},
		{"relative index write", "Edit", "index.md"},
	}
	for _, c := range allowed {
		event := `{"tool_name":"` + c.tool + `","tool_input":{"file_path":"` + c.path + `"}}`
		code, out := runHook(t, hookPath, event)
		if code != 0 {
			t.Errorf("%s: expected allow (exit 0), got exit %d, out=%s", c.name, code, out)
		}
	}

	// A non-write tool aimed at raw/ is allowed (reads are fine).
	readEvent := `{"tool_name":"Read","tool_input":{"file_path":"` + rawPath + `"}}`
	if code, out := runHook(t, hookPath, readEvent); code != 0 {
		t.Errorf("read of raw should be allowed, got exit %d out=%s", code, out)
	}
}
