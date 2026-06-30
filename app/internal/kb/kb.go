// Package kb describes the knowledge-base layout and locates the wiki root.
//
// A knowledge base is a single directory containing a wiki/ folder. The wiki/
// folder is self-contained: immutable raw/ sources, agent-owned pages, and the
// control files (schema.yaml, CLAUDE.md, the raw-write hook). There is no
// multi-project layout — the directory that holds wiki/ is the project.
package kb

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// WikiDirName is the folder whose presence marks a knowledge base.
	WikiDirName = "wiki"
	// RawDirName holds immutable source material inside the wiki.
	RawDirName = "raw"
	// SchemaName is the frontmatter source of truth, inside the wiki.
	SchemaName = "schema.yaml"
	// AgentSchemaName is the agent-control file, inside the wiki.
	AgentSchemaName = "CLAUDE.md"
)

// Root is a located knowledge base: the directory that contains a wiki/ folder.
type Root struct {
	Dir string // absolute path to the directory holding wiki/
}

// Find walks upward from start looking for a directory that contains a wiki/
// subdirectory. The first such directory is the knowledge-base root.
func Find(start string) (*Root, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		if dirExists(filepath.Join(dir, WikiDirName)) {
			return &Root{Dir: dir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("no %s/ directory found in %s or any parent directory (run `nvtwiki init`)", WikiDirName, start)
		}
		dir = parent
	}
}

// WikiDir is the agent-owned wiki tree.
func (r *Root) WikiDir() string { return filepath.Join(r.Dir, WikiDirName) }

// RawDir is the immutable source directory, inside the wiki.
func (r *Root) RawDir() string { return filepath.Join(r.WikiDir(), RawDirName) }

// SchemaPath is the frontmatter schema, inside the wiki.
func (r *Root) SchemaPath() string { return filepath.Join(r.WikiDir(), SchemaName) }

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
