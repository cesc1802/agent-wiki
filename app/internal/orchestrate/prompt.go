package orchestrate

import (
	"fmt"
	"strings"
)

// QueryPrompt builds the headless prompt for answering a question from the
// wiki. The detailed workflow lives in the wiki's CLAUDE.md; this prompt states
// the task and whether to persist a synthesis page. The agent's working
// directory is the wiki/ folder, so paths are relative to it.
func QueryPrompt(question string, save bool) string {
	var b strings.Builder
	b.WriteString("Answer a question using only this wiki.\n\n")
	fmt.Fprintf(&b, "Question:\n%s\n\n", strings.TrimSpace(question))
	b.WriteString("Follow the \"Query workflow\" in CLAUDE.md:\n")
	b.WriteString("1. Read index.md to locate relevant pages.\n")
	b.WriteString("2. Read those pages.\n")
	b.WriteString("3. Answer concisely, citing the wiki pages you used and the underlying raw/ sources behind them. If the wiki does not contain the answer, say exactly what is missing instead of guessing.\n")
	if save {
		b.WriteString("4. Then persist the answer: create a synthesis/ page with valid frontmatter per schema.yaml, update index.md, and append a `query` entry to log.md.\n")
	} else {
		b.WriteString("\nThis is a read-only query: do not create, edit, or delete any files.\n")
	}
	return b.String()
}

// IngestPrompt builds the headless prompt for ingesting one raw source file
// into the wiki. rawRel is the path relative to the wiki/ working directory,
// e.g. "raw/plan.md".
func IngestPrompt(rawRel string) string {
	var b strings.Builder
	b.WriteString("Ingest a raw source into this wiki.\n\n")
	fmt.Fprintf(&b, "Raw source to ingest: %s\n\n", rawRel)
	b.WriteString("Follow the \"Ingest workflow\" in CLAUDE.md exactly, in order, without skipping steps:\n")
	fmt.Fprintf(&b, "- Read %s fully.\n", rawRel)
	b.WriteString("- Read index.md to learn what already exists.\n")
	b.WriteString("- Create a sources/ page (type source) recording the raw file, with its path in `sources:`.\n")
	b.WriteString("- Create or update entities/ and concepts/ pages for what the source introduces.\n")
	b.WriteString("- Add cross-reference links between related pages.\n")
	b.WriteString("- If a new page supersedes an old one, set the old page's status to superseded and its superseded_by to the new page's path.\n")
	b.WriteString("- Update index.md to catalog every new or changed page, and append an `ingest` entry to log.md.\n\n")
	b.WriteString("Every page except index.md and log.md must start with frontmatter valid against schema.yaml. Write only page files; raw/ and the control files are read-only and a hook will block writes to them.\n")
	return b.String()
}
