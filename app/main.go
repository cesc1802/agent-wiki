// Command nvtwiki is a deterministic CLI orchestrator for an agent-maintained
// project knowledge wiki. It owns the mechanical gates (scaffold, validate,
// lint, navigation) and orchestrates Claude Code for semantic work.
package main

import (
	"os"

	"nvtwiki/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
