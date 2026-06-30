package nav

import (
	"os"
	"path/filepath"
	"testing"

	"nvtwiki/internal/kb"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"  raw/x.md ": "raw/x.md",
		"./raw/x.md":  "raw/x.md",
		"raw/x.md":    "raw/x.md",
	}
	for in, want := range cases {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsIngested(t *testing.T) {
	set := map[string]bool{"raw/widgets.md": true}
	if !isIngested("widgets.md", set) {
		t.Error("expected widgets.md ingested via raw/ form")
	}
	if isIngested("other.md", set) {
		t.Error("other.md should not be ingested")
	}
	if isIngested("nested/widgets.md", set) {
		t.Error("nested raw path should not match a top-level source ref")
	}
}

func TestSourceStrings(t *testing.T) {
	got := sourceStrings([]interface{}{"a.md", 42, "b.md"})
	if len(got) != 2 || got[0] != "a.md" || got[1] != "b.md" {
		t.Fatalf("expected [a.md b.md], got %v", got)
	}
	if sourceStrings("not-a-list") != nil {
		t.Error("non-list should yield nil")
	}
}

func TestLogFilterAndTail(t *testing.T) {
	dir := t.TempDir()
	wikiDir := filepath.Join(dir, "wiki")
	if err := os.MkdirAll(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logBody := "# Log\n\n" +
		"## [2026-06-01] init | project created\n" +
		"## [2026-06-02] ingest | first source\n" +
		"## [2026-06-03] query | a question\n" +
		"## [2026-06-04] ingest | second source\n"
	if err := os.WriteFile(filepath.Join(wikiDir, "log.md"), []byte(logBody), 0o644); err != nil {
		t.Fatal(err)
	}
	root := &kb.Root{Dir: dir}

	all, err := Log(root, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(all))
	}

	ingest, err := Log(root, "ingest", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ingest) != 2 {
		t.Fatalf("expected 2 ingest entries, got %d", len(ingest))
	}

	tail, err := Log(root, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) != 1 || tail[0].Date != "2026-06-04" {
		t.Fatalf("expected last entry 2026-06-04, got %v", tail)
	}
}
