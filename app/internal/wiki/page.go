// Package wiki models pages in a project's wiki/ tree: frontmatter parsing,
// page discovery, and cross-reference link extraction. It is deterministic and
// never mutates page content.
package wiki

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// errUnterminatedFrontmatter marks a file that opens a frontmatter block but
// never closes it.
var errUnterminatedFrontmatter = errors.New("unterminated frontmatter block")

// Page is a single markdown file under a project's wiki/ directory.
type Page struct {
	// WikiRel is the slash-separated path relative to the wiki/ dir, e.g.
	// "concepts/hard-gate-fsm.md".
	WikiRel string
	// Abs is the absolute filesystem path.
	Abs string
	// HasFront reports whether the file opened with a frontmatter block.
	HasFront bool
	// Front holds the parsed frontmatter mapping (nil when HasFront is false).
	Front map[string]interface{}
	// FrontErr holds a YAML parse error for the frontmatter block, if any.
	FrontErr error
	// Body is the markdown content after the frontmatter block.
	Body string
	// Links are cross-reference links to other .md files inside the wiki.
	Links []Link
}

// Base returns the file's base name, e.g. "index.md".
func (p Page) Base() string { return filepath.Base(p.WikiRel) }

// Link is a resolved cross-reference from one page to a markdown target.
type Link struct {
	// Target is the path exactly as written in the markdown, e.g. "../x.md".
	Target string
	// Resolved is the cleaned slash path relative to the wiki/ dir.
	Resolved string
	// Line is the 1-based line number where the link appears.
	Line int
}

// markdownLink matches the path inside a markdown link: `](path)`. It also
// matches the wikilink-with-path convention `[[label]](path)` because the
// closing `]` before `(` is captured the same way.
var markdownLink = regexp.MustCompile(`\]\(([^)]+)\)`)

// reservedDirs are wiki/ subdirectories that hold non-page material: immutable
// sources and control files. They are skipped during page discovery.
var reservedDirs = map[string]bool{"raw": true, ".claude": true, "hooks": true}

// reservedFiles are control files that live alongside pages but are not pages.
var reservedFiles = map[string]bool{"CLAUDE.md": true}

// ScanPages walks wikiDir and returns every *.md page parsed into a Page. The
// reserved raw/, .claude/, and hooks/ directories and the CLAUDE.md control
// file are skipped — only agent-owned pages are returned.
func ScanPages(wikiDir string) ([]Page, error) {
	var pages []Page
	err := filepath.Walk(wikiDir, func(abs string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if abs != wikiDir && reservedDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") || reservedFiles[info.Name()] {
			return nil
		}
		rel, err := filepath.Rel(wikiDir, abs)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		wikiRel := filepath.ToSlash(rel)
		front, body, hasFront, ferr := ParseFrontmatter(string(content))
		pages = append(pages, Page{
			WikiRel:  wikiRel,
			Abs:      abs,
			HasFront: hasFront,
			Front:    front,
			FrontErr: ferr,
			Body:     body,
			Links:    ExtractLinks(body, wikiRel),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pages, nil
}

// ParseFrontmatter splits a YAML frontmatter block delimited by lines
// containing only `---`. It returns the parsed mapping, the remaining body,
// whether a frontmatter block was present, and any YAML parse error.
func ParseFrontmatter(content string) (map[string]interface{}, string, bool, error) {
	trimmed := strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(trimmed, "---\n") && trimmed != "---" {
		return nil, content, false, nil
	}
	rest := strings.TrimPrefix(trimmed, "---\n")
	// Find the closing delimiter line.
	lines := strings.Split(rest, "\n")
	end := -1
	for i, ln := range lines {
		if strings.TrimRight(ln, "\r") == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		// Opened but never closed: treat as malformed frontmatter.
		return nil, content, true, errUnterminatedFrontmatter
	}
	block := strings.Join(lines[:end], "\n")
	body := strings.Join(lines[end+1:], "\n")
	var front map[string]interface{}
	if err := yaml.Unmarshal([]byte(block), &front); err != nil {
		return nil, body, true, err
	}
	if front == nil {
		front = map[string]interface{}{}
	}
	return front, body, true, nil
}

// ExtractLinks finds markdown links to .md targets within the wiki and resolves
// them relative to the page's directory. External links (http, mailto) and
// non-markdown targets are ignored.
func ExtractLinks(body, pageWikiRel string) []Link {
	var links []Link
	pageDir := pathDir(pageWikiRel)
	for _, m := range markdownLink.FindAllStringSubmatchIndex(body, -1) {
		target := body[m[2]:m[3]]
		clean := stripAnchor(strings.TrimSpace(target))
		if clean == "" || isExternal(clean) || !strings.HasSuffix(clean, ".md") {
			continue
		}
		resolved := cleanJoin(pageDir, clean)
		links = append(links, Link{
			Target:   target,
			Resolved: resolved,
			Line:     1 + strings.Count(body[:m[0]], "\n"),
		})
	}
	return links
}

func isExternal(s string) bool {
	low := strings.ToLower(s)
	return strings.HasPrefix(low, "http://") ||
		strings.HasPrefix(low, "https://") ||
		strings.HasPrefix(low, "mailto:") ||
		strings.HasPrefix(low, "#")
}

func stripAnchor(s string) string {
	if i := strings.IndexByte(s, '#'); i >= 0 {
		return s[:i]
	}
	return s
}

// pathDir returns the slash directory of a wiki-relative path ("" for root).
func pathDir(p string) string {
	i := strings.LastIndexByte(p, '/')
	if i < 0 {
		return ""
	}
	return p[:i]
}

// cleanJoin joins a base dir and a relative target into a cleaned slash path
// relative to the wiki root.
func cleanJoin(dir, target string) string {
	joined := target
	if dir != "" {
		joined = dir + "/" + target
	}
	return filepath.ToSlash(filepath.Clean(joined))
}
