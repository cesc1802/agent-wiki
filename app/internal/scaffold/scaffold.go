// Package scaffold creates the self-contained wiki/ structure from embedded
// templates. It only ever creates files; it never edits existing page content.
package scaffold

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nvtwiki/internal/kb"
)

//go:embed assets
var assets embed.FS

// wikiSubdirs are the directories created inside every wiki/. raw/ holds
// immutable sources; the rest are page categories.
var wikiSubdirs = []string{
	kb.RawDirName, "entities", "concepts", "sources", "synthesis",
}

// Init scaffolds a self-contained wiki/ inside dir. Everything the orchestrator
// and the agent need lives under wiki/: page categories, seed pages, and the
// control files (schema.yaml, CLAUDE.md, the raw-write hook, and the Claude
// settings that register it). Existing files are left untouched; Init reports
// the wiki-relative paths it created. date seeds the log and overview pages.
func Init(dir, date string) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, err
	}
	wikiDir := filepath.Join(absDir, kb.WikiDirName)
	name := filepath.Base(absDir)
	subst := map[string]string{"{{PROJECT}}": name, "{{DATE}}": date}

	var created []string
	add := func(rel string) { created = append(created, kb.WikiDirName+"/"+rel) }

	// Category directories, each kept tracked with a .gitkeep file.
	for _, sub := range wikiSubdirs {
		d := filepath.Join(wikiDir, sub)
		if err := os.MkdirAll(d, 0o755); err != nil {
			return created, err
		}
		if err := os.WriteFile(filepath.Join(d, ".gitkeep"), nil, 0o644); err != nil {
			return created, err
		}
	}

	// Seed pages (name/date substituted).
	seeds := map[string]string{
		"index.md":    "assets/wiki/index.md",
		"log.md":      "assets/wiki/log.md",
		"overview.md": "assets/wiki/overview.md",
	}
	for dest, src := range seeds {
		wrote, err := writeAssetIfAbsent(src, filepath.Join(wikiDir, dest), 0o644, subst)
		if err != nil {
			return created, err
		}
		if wrote {
			add(dest)
		}
	}

	// Control files: schema + agent-control file at the wiki root.
	controlFiles := map[string]string{
		kb.SchemaName:      "assets/schema.yaml",
		kb.AgentSchemaName: "assets/CLAUDE.md",
	}
	for dest, src := range controlFiles {
		wrote, err := writeAssetIfAbsent(src, filepath.Join(wikiDir, dest), 0o644, nil)
		if err != nil {
			return created, err
		}
		if wrote {
			add(dest)
		}
	}

	// The raw-write hook and the settings that register it for `claude -p`.
	// claude runs with its working directory set to wiki/, so
	// $CLAUDE_PROJECT_DIR resolves here and the hook path matches.
	binaries := []struct {
		src, dest string
		mode      os.FileMode
	}{
		{"assets/block-raw-write.sh", filepath.Join("hooks", "block-raw-write.sh"), 0o755},
		{"assets/claude-settings.json", filepath.Join(".claude", "settings.json"), 0o644},
	}
	for _, b := range binaries {
		wrote, err := writeAssetIfAbsent(b.src, filepath.Join(wikiDir, b.dest), b.mode, nil)
		if err != nil {
			return created, err
		}
		if wrote {
			add(filepath.ToSlash(b.dest))
		}
	}

	return created, nil
}

// writeAssetIfAbsent writes an embedded asset to dest unless dest exists. When
// subst is non-nil, placeholder keys are replaced in the content. It returns
// whether the file was written.
func writeAssetIfAbsent(src, dest string, mode os.FileMode, subst map[string]string) (bool, error) {
	if _, err := os.Stat(dest); err == nil {
		return false, nil
	}
	data, err := assets.ReadFile(src)
	if err != nil {
		return false, fmt.Errorf("read embedded asset %s: %w", src, err)
	}
	content := string(data)
	for k, v := range subst {
		content = strings.ReplaceAll(content, k, v)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(dest, []byte(content), mode); err != nil {
		return false, err
	}
	return true, nil
}
