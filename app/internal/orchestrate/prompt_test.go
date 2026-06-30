package orchestrate

import (
	"strings"
	"testing"
)

func TestQueryPromptReadOnly(t *testing.T) {
	p := QueryPrompt("  How does auth work?  ", false)
	if !strings.Contains(p, "How does auth work?") {
		t.Error("question not embedded (or not trimmed)")
	}
	if !strings.Contains(p, "index.md") {
		t.Error("should point the agent at the wiki index")
	}
	if !strings.Contains(p, "read-only") {
		t.Error("read-only query must forbid writes")
	}
	if strings.Contains(p, "synthesis/") {
		t.Error("non-save query should not mention persisting a synthesis page")
	}
}

func TestQueryPromptSave(t *testing.T) {
	p := QueryPrompt("q", true)
	if !strings.Contains(p, "synthesis/") || !strings.Contains(p, "log.md") {
		t.Error("save query must instruct persisting synthesis + log entry")
	}
	if strings.Contains(p, "do not create, edit, or delete") {
		t.Error("save query should not forbid writes")
	}
}

func TestIngestPrompt(t *testing.T) {
	p := IngestPrompt("raw/plan.md")
	for _, want := range []string{
		"raw/plan.md",
		"index.md",
		"sources/",
		"schema.yaml",
		"read-only",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("ingest prompt missing %q", want)
		}
	}
}
