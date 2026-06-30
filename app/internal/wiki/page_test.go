package wiki

import "testing"

func TestParseFrontmatter(t *testing.T) {
	const valid = "---\ntitle: Widget\ntype: concept\n---\nBody line one.\n"
	front, body, has, err := ParseFrontmatter(valid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Fatal("expected frontmatter present")
	}
	if front["title"] != "Widget" || front["type"] != "concept" {
		t.Fatalf("unexpected front: %#v", front)
	}
	if body != "Body line one.\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestParseFrontmatterNone(t *testing.T) {
	front, body, has, err := ParseFrontmatter("# Just a heading\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Fatal("expected no frontmatter")
	}
	if front != nil {
		t.Fatalf("expected nil front, got %#v", front)
	}
	if body != "# Just a heading\n" {
		t.Fatalf("body should be untouched, got %q", body)
	}
}

func TestParseFrontmatterUnterminated(t *testing.T) {
	_, _, has, err := ParseFrontmatter("---\ntitle: Widget\nstill going\n")
	if !has {
		t.Fatal("expected frontmatter detected")
	}
	if err != errUnterminatedFrontmatter {
		t.Fatalf("expected unterminated error, got %v", err)
	}
}

func TestParseFrontmatterBOM(t *testing.T) {
	const withBOM = "\ufeff---\ntitle: Widget\n---\nbody\n"
	front, _, has, err := ParseFrontmatter(withBOM)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has || front["title"] != "Widget" {
		t.Fatalf("BOM not stripped: has=%v front=%#v", has, front)
	}
}

func TestParseFrontmatterBadYAML(t *testing.T) {
	_, _, has, err := ParseFrontmatter("---\ntitle: [unclosed\n---\nbody\n")
	if !has {
		t.Fatal("expected frontmatter detected")
	}
	if err == nil {
		t.Fatal("expected YAML parse error")
	}
}

func TestExtractLinks(t *testing.T) {
	body := "See [[Overview]](../overview.md) and [[Sib]](sibling.md#anchor).\n" +
		"External [site](https://example.com) and image [x](pic.png) ignored.\n"
	links := ExtractLinks(body, "concepts/widget.md")
	if len(links) != 2 {
		t.Fatalf("expected 2 md links, got %d: %#v", len(links), links)
	}
	if links[0].Resolved != "overview.md" {
		t.Fatalf("expected ../overview.md to resolve to overview.md, got %q", links[0].Resolved)
	}
	if links[1].Resolved != "concepts/sibling.md" {
		t.Fatalf("expected sibling resolution, got %q", links[1].Resolved)
	}
	if links[0].Line != 1 || links[1].Line != 1 {
		t.Fatalf("expected both links on line 1, got %d/%d", links[0].Line, links[1].Line)
	}
}

func TestExtractLinksLineNumbers(t *testing.T) {
	body := "line one\nline two [[A]](a.md)\n\nline four [[B]](b.md)\n"
	links := ExtractLinks(body, "index.md")
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].Line != 2 {
		t.Fatalf("expected first link on line 2, got %d", links[0].Line)
	}
	if links[1].Line != 4 {
		t.Fatalf("expected second link on line 4, got %d", links[1].Line)
	}
}
