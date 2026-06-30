package lint

import (
	"strings"
	"testing"

	"nvtwiki/internal/wiki"
)

// active builds a page with active status and the given links.
func mkPage(rel string, links []wiki.Link, front map[string]interface{}) wiki.Page {
	p := wiki.Page{WikiRel: rel, Links: links}
	if front != nil {
		p.HasFront = true
		p.Front = front
	}
	return p
}

func hasFinding(findings []Finding, rule, pageSubstr string) bool {
	for _, f := range findings {
		if f.Rule == rule && strings.Contains(f.Page, pageSubstr) {
			return true
		}
	}
	return false
}

func TestBrokenLink(t *testing.T) {
	pages := []wiki.Page{
		mkPage("index.md", []wiki.Link{{Target: "concepts/ghost.md", Resolved: "concepts/ghost.md", Line: 3}}, nil),
	}
	findings := Run(pages)
	if !hasFinding(findings, "broken-link", "index.md") {
		t.Fatalf("expected broken-link finding, got %v", findings)
	}
}

func TestOrphanAndExemption(t *testing.T) {
	pages := []wiki.Page{
		mkPage("index.md", []wiki.Link{{Resolved: "concepts/linked.md"}}, nil),
		mkPage("concepts/linked.md", nil, nil), // linked from index -> not orphan
		mkPage("concepts/orphan.md", nil, nil), // nobody links -> orphan
		mkPage("overview.md", nil, nil),        // exempt
	}
	findings := Run(pages)
	if !hasFinding(findings, "orphan", "concepts/orphan.md") {
		t.Fatalf("expected orphan finding for orphan.md, got %v", findings)
	}
	if hasFinding(findings, "orphan", "concepts/linked.md") {
		t.Fatal("linked.md should not be an orphan")
	}
	if hasFinding(findings, "orphan", "overview.md") {
		t.Fatal("overview.md is exempt from orphan check")
	}
}

func TestSupersededMissingTarget(t *testing.T) {
	pages := []wiki.Page{
		mkPage("index.md", []wiki.Link{{Resolved: "concepts/old.md"}}, nil),
		mkPage("concepts/old.md", nil, map[string]interface{}{
			"status": "superseded", "superseded_by": "",
		}),
	}
	findings := Run(pages)
	if !hasFinding(findings, "superseded", "concepts/old.md") {
		t.Fatalf("expected superseded finding for empty superseded_by, got %v", findings)
	}
}

func TestActiveWithSupersededBy(t *testing.T) {
	pages := []wiki.Page{
		mkPage("index.md", []wiki.Link{{Resolved: "concepts/a.md"}, {Resolved: "concepts/b.md"}}, nil),
		mkPage("concepts/a.md", nil, map[string]interface{}{
			"status": "active", "superseded_by": "b.md",
		}),
		mkPage("concepts/b.md", nil, map[string]interface{}{
			"status": "active", "superseded_by": nil,
		}),
	}
	findings := Run(pages)
	if !hasFinding(findings, "superseded", "concepts/a.md") {
		t.Fatalf("expected finding: active page with superseded_by, got %v", findings)
	}
}

func TestSupersededCycle(t *testing.T) {
	pages := []wiki.Page{
		mkPage("index.md", []wiki.Link{{Resolved: "a.md"}, {Resolved: "b.md"}}, nil),
		mkPage("a.md", nil, map[string]interface{}{"status": "superseded", "superseded_by": "b.md"}),
		mkPage("b.md", nil, map[string]interface{}{"status": "superseded", "superseded_by": "a.md"}),
	}
	findings := Run(pages)
	cycleCount := 0
	for _, f := range findings {
		if f.Rule == "superseded" && strings.Contains(f.Msg, "cycle") {
			cycleCount++
		}
	}
	if cycleCount != 2 {
		t.Fatalf("expected both pages flagged in cycle, got %d: %v", cycleCount, findings)
	}
}
