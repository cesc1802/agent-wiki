// Package lint runs mechanical (non-semantic) health checks over a project
// wiki: broken cross-reference links, orphan pages, and superseded_by/status
// consistency. It never reads page content semantically and never mutates.
package lint

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"nvtwiki/internal/wiki"
)

// Finding is a single lint problem.
type Finding struct {
	Page string // wiki-relative path of the offending page
	Line int    // 1-based line, or 0 when not line-specific
	Rule string // broken-link | orphan | superseded
	Msg  string
}

// String renders a finding as `page:line [rule] message`.
func (f Finding) String() string {
	loc := f.Page
	if f.Line > 0 {
		loc = fmt.Sprintf("%s:%d", f.Page, f.Line)
	}
	return fmt.Sprintf("%s [%s] %s", loc, f.Rule, f.Msg)
}

// orphanExempt lists root pages that are never considered orphans.
var orphanExempt = map[string]bool{
	"index.md":    true,
	"log.md":      true,
	"overview.md": true,
}

// Run executes all mechanical checks and returns findings sorted by page then
// line for deterministic output.
func Run(pages []wiki.Page) []Finding {
	pageSet := make(map[string]bool, len(pages))
	for _, p := range pages {
		pageSet[p.WikiRel] = true
	}

	var findings []Finding
	findings = append(findings, brokenLinks(pages, pageSet)...)
	findings = append(findings, orphans(pages, pageSet)...)
	findings = append(findings, supersededIssues(pages, pageSet)...)

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Page != findings[j].Page {
			return findings[i].Page < findings[j].Page
		}
		return findings[i].Line < findings[j].Line
	})
	return findings
}

func brokenLinks(pages []wiki.Page, pageSet map[string]bool) []Finding {
	var out []Finding
	for _, p := range pages {
		for _, l := range p.Links {
			if !pageSet[l.Resolved] {
				out = append(out, Finding{
					Page: p.WikiRel, Line: l.Line, Rule: "broken-link",
					Msg: fmt.Sprintf("link target %q (resolved %q) does not exist", l.Target, l.Resolved),
				})
			}
		}
	}
	return out
}

func orphans(pages []wiki.Page, pageSet map[string]bool) []Finding {
	inbound := make(map[string]bool)
	for _, p := range pages {
		for _, l := range p.Links {
			if pageSet[l.Resolved] && l.Resolved != p.WikiRel {
				inbound[l.Resolved] = true
			}
		}
	}
	var out []Finding
	for _, p := range pages {
		if orphanExempt[p.Base()] || inbound[p.WikiRel] {
			continue
		}
		out = append(out, Finding{
			Page: p.WikiRel, Rule: "orphan",
			Msg: "no other page links to this page",
		})
	}
	return out
}

func supersededIssues(pages []wiki.Page, pageSet map[string]bool) []Finding {
	var out []Finding
	// edges holds page -> resolved superseded_by target for cycle detection.
	edges := make(map[string]string)

	for _, p := range pages {
		if !p.HasFront || p.FrontErr != nil {
			continue // validate reports these
		}
		status, _ := p.Front["status"].(string)
		sbVal, sbPresent := p.Front["superseded_by"]
		sb := ""
		if s, ok := sbVal.(string); ok {
			sb = strings.TrimSpace(s)
		}

		switch status {
		case "superseded":
			if !sbPresent || sb == "" {
				out = append(out, Finding{
					Page: p.WikiRel, Rule: "superseded",
					Msg: "status is superseded but superseded_by is empty",
				})
			}
		case "active":
			if sb != "" {
				out = append(out, Finding{
					Page: p.WikiRel, Rule: "superseded",
					Msg: fmt.Sprintf("status is active but superseded_by points to %q", sb),
				})
			}
		}

		if sb != "" {
			resolved := resolveFrom(p.WikiRel, sb)
			if !pageSet[resolved] {
				out = append(out, Finding{
					Page: p.WikiRel, Rule: "superseded",
					Msg: fmt.Sprintf("superseded_by target %q (resolved %q) does not exist", sb, resolved),
				})
			} else {
				edges[p.WikiRel] = resolved
			}
		}
	}

	for _, c := range findCycles(edges) {
		out = append(out, Finding{
			Page: c, Rule: "superseded",
			Msg: "superseded_by forms a cycle",
		})
	}
	return out
}

// resolveFrom resolves a target path relative to a page's directory, matching
// cross-reference link resolution.
func resolveFrom(pageWikiRel, target string) string {
	dir := ""
	if i := strings.LastIndexByte(pageWikiRel, '/'); i >= 0 {
		dir = pageWikiRel[:i]
	}
	joined := target
	if dir != "" {
		joined = dir + "/" + target
	}
	return filepath.ToSlash(filepath.Clean(joined))
}

// findCycles returns the set of nodes that participate in any cycle of the
// superseded_by graph (each node has at most one outgoing edge).
func findCycles(edges map[string]string) []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	inCycle := make(map[string]bool)

	var visit func(node string, path []string)
	visit = func(node string, path []string) {
		color[node] = gray
		path = append(path, node)
		if next, ok := edges[node]; ok {
			switch color[next] {
			case white:
				visit(next, path)
			case gray:
				// Found a back-edge: mark the cycle from `next` onward.
				mark := false
				for _, n := range path {
					if n == next {
						mark = true
					}
					if mark {
						inCycle[n] = true
					}
				}
			}
		}
		color[node] = black
	}

	for node := range edges {
		if color[node] == white {
			visit(node, nil)
		}
	}

	var out []string
	for n := range inCycle {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
