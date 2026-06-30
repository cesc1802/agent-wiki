// Package nav provides read-only navigation over a project: page statistics,
// timeline log access, and raw-source ingest tracking. Deterministic, no LLM.
package nav

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"nvtwiki/internal/kb"
	"nvtwiki/internal/wiki"
)

// Stats summarizes a project's wiki and raw sources.
type Stats struct {
	TotalPages    int
	ByType        map[string]int
	ByStatus      map[string]int
	NoFrontmatter int
	RawTotal      int
	RawIngested   int
	RawPending    []string // raw paths (relative to raw/) with no source page
}

// Status scans the wiki and raw directories.
func Status(root *kb.Root) (*Stats, error) {
	pages, err := wiki.ScanPages(root.WikiDir())
	if err != nil {
		return nil, err
	}
	s := &Stats{ByType: map[string]int{}, ByStatus: map[string]int{}}
	sourceSet := map[string]bool{}
	for _, p := range pages {
		s.TotalPages++
		if !p.HasFront || p.FrontErr != nil {
			if p.Base() != "index.md" && p.Base() != "log.md" {
				s.NoFrontmatter++
			}
			continue
		}
		if t, ok := p.Front["type"].(string); ok {
			s.ByType[t]++
		}
		if st, ok := p.Front["status"].(string); ok {
			s.ByStatus[st]++
		}
		for _, src := range sourceStrings(p.Front["sources"]) {
			sourceSet[normalize(src)] = true
		}
	}

	rawRels, err := listRaw(root.RawDir())
	if err != nil {
		return nil, err
	}
	for _, rel := range rawRels {
		s.RawTotal++
		if isIngested(rel, sourceSet) {
			s.RawIngested++
		} else {
			s.RawPending = append(s.RawPending, rel)
		}
	}
	sort.Strings(s.RawPending)
	return s, nil
}

// Pending returns raw sources that have no corresponding source page.
func Pending(root *kb.Root) ([]string, error) {
	s, err := Status(root)
	if err != nil {
		return nil, err
	}
	return s.RawPending, nil
}

// isIngested reports whether a raw file (path relative to raw/) is referenced
// by some page's sources list. The canonical source form is "raw/<rel>".
func isIngested(rel string, sourceSet map[string]bool) bool {
	return sourceSet[normalize("raw/"+rel)]
}

func normalize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "./")
	return filepath.ToSlash(s)
}

func sourceStrings(v interface{}) []string {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func listRaw(rawDir string) ([]string, error) {
	var rels []string
	err := filepath.Walk(rawDir, func(abs string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() || info.Name() == ".gitkeep" {
			return nil
		}
		rel, err := filepath.Rel(rawDir, abs)
		if err != nil {
			return err
		}
		rels = append(rels, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rels, nil
}

// LogEntry is one timeline entry parsed from log.md.
type LogEntry struct {
	Raw  string
	Op   string
	Date string
}

var logLineRe = regexp.MustCompile(`^## \[([0-9]{4}-[0-9]{2}-[0-9]{2})\]\s*([a-zA-Z]+)`)

// Log reads the wiki's log.md and returns matching entries. opFilter empty
// means all ops; n <= 0 means all entries, otherwise the last n.
func Log(root *kb.Root, opFilter string, n int) ([]LogEntry, error) {
	data, err := os.ReadFile(filepath.Join(root.WikiDir(), "log.md"))
	if err != nil {
		return nil, err
	}
	var entries []LogEntry
	for _, line := range strings.Split(string(data), "\n") {
		m := logLineRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		op := m[2]
		if opFilter != "" && op != opFilter {
			continue
		}
		entries = append(entries, LogEntry{Raw: line, Op: op, Date: m[1]})
	}
	if n > 0 && len(entries) > n {
		entries = entries[len(entries)-n:]
	}
	return entries, nil
}
