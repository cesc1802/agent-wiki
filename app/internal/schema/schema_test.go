package schema

import (
	"strings"
	"testing"
	"time"

	"nvtwiki/internal/wiki"
)

func testSchema() *Schema {
	return &Schema{
		Exempt: []string{"index.md", "log.md"},
		Fields: map[string]FieldSpec{
			"title":         {Type: "string", Required: true},
			"type":          {Type: "enum", Required: true, Values: []string{"entity", "concept"}},
			"status":        {Type: "enum", Required: true, Values: []string{"active", "superseded"}},
			"superseded_by": {Type: "string", Required: true, Nullable: true},
			"sources":       {Type: "list", Required: true},
			"created":       {Type: "date", Required: true},
		},
	}
}

func page(base string, front map[string]interface{}) wiki.Page {
	p := wiki.Page{WikiRel: base}
	if front != nil {
		p.HasFront = true
		p.Front = front
	}
	return p
}

func TestValidateExempt(t *testing.T) {
	if probs := testSchema().ValidatePage(page("index.md", nil)); probs != nil {
		t.Fatalf("exempt page should have no problems, got %v", probs)
	}
}

func TestValidateMissingFrontmatter(t *testing.T) {
	probs := testSchema().ValidatePage(page("concepts/x.md", nil))
	if len(probs) != 1 || !strings.Contains(probs[0], "missing frontmatter") {
		t.Fatalf("expected missing-frontmatter problem, got %v", probs)
	}
}

func TestValidateValidPageWithTimeDate(t *testing.T) {
	front := map[string]interface{}{
		"title":         "Widget",
		"type":          "concept",
		"status":        "active",
		"superseded_by": nil,
		"sources":       []interface{}{"raw/widgets.md"},
		"created":       time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	}
	if probs := testSchema().ValidatePage(page("concepts/widget.md", front)); len(probs) != 0 {
		t.Fatalf("expected valid page, got problems: %v", probs)
	}
}

func TestValidateQuotedDate(t *testing.T) {
	front := map[string]interface{}{
		"title": "W", "type": "concept", "status": "active",
		"superseded_by": nil, "sources": []interface{}{"raw/x.md"},
		"created": "2026-06-30",
	}
	if probs := testSchema().ValidatePage(page("concepts/w.md", front)); len(probs) != 0 {
		t.Fatalf("quoted date should be valid, got: %v", probs)
	}
}

func TestValidateBadDate(t *testing.T) {
	front := map[string]interface{}{
		"title": "W", "type": "concept", "status": "active",
		"superseded_by": nil, "sources": []interface{}{"raw/x.md"},
		"created": "30-06-2026",
	}
	probs := testSchema().ValidatePage(page("concepts/w.md", front))
	if len(probs) != 1 || !strings.Contains(probs[0], "YYYY-MM-DD") {
		t.Fatalf("expected date-format problem, got: %v", probs)
	}
}

func TestValidateMissingRequired(t *testing.T) {
	front := map[string]interface{}{"title": "W"}
	probs := testSchema().ValidatePage(page("concepts/w.md", front))
	// type, status, superseded_by, sources, created missing = 5.
	if len(probs) != 5 {
		t.Fatalf("expected 5 missing-required problems, got %d: %v", len(probs), probs)
	}
}

func TestValidateNullNonNullable(t *testing.T) {
	front := map[string]interface{}{
		"title": "W", "type": "concept", "status": "active",
		"superseded_by": nil, "sources": []interface{}{"raw/x.md"},
		"created": nil,
	}
	probs := testSchema().ValidatePage(page("concepts/w.md", front))
	if len(probs) != 1 || !strings.Contains(probs[0], "must not be null") {
		t.Fatalf("expected null problem for created, got: %v", probs)
	}
}

func TestValidateEnumAndList(t *testing.T) {
	front := map[string]interface{}{
		"title": "W", "type": "gizmo", "status": "active",
		"superseded_by": nil, "sources": "not-a-list",
		"created": "2026-06-30",
	}
	probs := testSchema().ValidatePage(page("concepts/w.md", front))
	joined := strings.Join(probs, "\n")
	if !strings.Contains(joined, `value "gizmo" not in`) {
		t.Fatalf("expected enum problem, got: %v", probs)
	}
	if !strings.Contains(joined, "must be a list") {
		t.Fatalf("expected list problem, got: %v", probs)
	}
}
