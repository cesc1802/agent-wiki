// Package cli wires the nvtwiki command surface (cobra). It is a thin interface
// layer: parsing and output only. All logic lives in internal packages.
//
// Every command operates on a single wiki, located by walking up from the
// current directory to the nearest folder containing a wiki/ directory.
package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"nvtwiki/internal/kb"
	"nvtwiki/internal/lint"
	"nvtwiki/internal/nav"
	"nvtwiki/internal/orchestrate"
	"nvtwiki/internal/scaffold"
	"nvtwiki/internal/schema"
	"nvtwiki/internal/wiki"
)

// Execute runs the root command and returns a process exit code.
func Execute() int {
	root := &cobra.Command{
		Use:           "nvtwiki",
		Short:         "Agent-maintained project knowledge wiki orchestrator",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		initCmd(),
		validateCmd(),
		lintCmd(),
		statusCmd(),
		logCmd(),
		rawCmd(),
		queryCmd(),
		ingestCmd(),
		authCmd(),
	)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return exitCode
}

// exitCode lets commands signal a non-zero result (e.g. validation failures)
// without treating them as usage errors.
var exitCode int

func today() string { return time.Now().Format("2006-01-02") }

// resolveRoot finds the wiki root from the current directory.
func resolveRoot() (*kb.Root, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return kb.Find(cwd)
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [dir]",
		Short: "Scaffold a self-contained wiki/ (pages, schema, CLAUDE.md, hook)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			created, err := scaffold.Init(dir, today())
			if err != nil {
				return err
			}
			if len(created) == 0 {
				fmt.Printf("wiki already initialized at %s (nothing to create)\n", filepath.Join(dir, kb.WikiDirName))
				return nil
			}
			fmt.Printf("initialized wiki at %s:\n", filepath.Join(dir, kb.WikiDirName))
			for _, f := range created {
				fmt.Printf("  + %s\n", f)
			}
			return nil
		},
	}
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate page frontmatter against schema.yaml",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			sch, err := schema.Load(root.SchemaPath())
			if err != nil {
				return err
			}
			pages, err := wiki.ScanPages(root.WikiDir())
			if err != nil {
				return err
			}
			sort.SliceStable(pages, func(i, j int) bool { return pages[i].WikiRel < pages[j].WikiRel })

			problemCount := 0
			for _, p := range pages {
				probs := sch.ValidatePage(p)
				if len(probs) == 0 {
					continue
				}
				problemCount += len(probs)
				fmt.Printf("%s:\n", p.WikiRel)
				for _, pr := range probs {
					fmt.Printf("  - %s\n", pr)
				}
			}
			if problemCount == 0 {
				fmt.Printf("validate: %d pages OK\n", len(pages))
				return nil
			}
			fmt.Printf("validate: %d problem(s) across pages\n", problemCount)
			exitCode = 1
			return nil
		},
	}
}

func lintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint",
		Short: "Mechanical lint: orphans, broken links, superseded_by/status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			pages, err := wiki.ScanPages(root.WikiDir())
			if err != nil {
				return err
			}
			findings := lint.Run(pages)
			for _, f := range findings {
				fmt.Println(f.String())
			}
			if len(findings) == 0 {
				fmt.Printf("lint: clean (%d pages)\n", len(pages))
				return nil
			}
			fmt.Printf("lint: %d finding(s)\n", len(findings))
			exitCode = 1
			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show page stats and ingest progress",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s, err := nav.Status(root)
			if err != nil {
				return err
			}
			fmt.Printf("wiki: %s\n", root.WikiDir())
			fmt.Printf("pages: %d\n", s.TotalPages)
			fmt.Printf("  by type:   %s\n", formatCounts(s.ByType))
			fmt.Printf("  by status: %s\n", formatCounts(s.ByStatus))
			if s.NoFrontmatter > 0 {
				fmt.Printf("  without frontmatter: %d\n", s.NoFrontmatter)
			}
			fmt.Printf("raw sources: %d (%d ingested, %d pending)\n",
				s.RawTotal, s.RawIngested, len(s.RawPending))
			return nil
		},
	}
}

func logCmd() *cobra.Command {
	var n int
	var op string
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Tail/filter the wiki timeline (log.md)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			entries, err := nav.Log(root, op, n)
			if err != nil {
				return err
			}
			for _, e := range entries {
				fmt.Println(e.Raw)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&n, "lines", "n", 0, "show only the last N entries")
	cmd.Flags().StringVar(&op, "op", "", "filter by operation: ingest|query|lint")
	return cmd
}

func rawCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "raw",
		Short: "List raw sources not yet ingested (ingest debt)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			pending, err := nav.Pending(root)
			if err != nil {
				return err
			}
			if len(pending) == 0 {
				fmt.Println("no pending raw sources (all ingested)")
				return nil
			}
			fmt.Printf("%d raw source(s) not yet ingested:\n", len(pending))
			for _, r := range pending {
				fmt.Printf("  raw/%s\n", r)
			}
			return nil
		},
	}
}

func queryCmd() *cobra.Command {
	var save bool
	var maxTurns int
	var budget float64
	cmd := &cobra.Command{
		Use:   "query <question>",
		Short: "Answer a question from the wiki via Claude (read-only by default)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if !orchestrate.Available() {
				return fmt.Errorf("%w", orchestrate.ErrClaudeNotFound)
			}
			question := strings.Join(args, " ")

			profile := orchestrate.Query
			if save {
				// Persisting a synthesis page requires write access; the hook
				// still confines writes to wiki/ pages.
				profile = orchestrate.Ingest
			}
			res, runErr := orchestrate.Run(cmd.Context(), orchestrate.Request{
				Prompt:       orchestrate.QueryPrompt(question, save),
				Profile:      profile,
				WorkDir:      root.WikiDir(),
				MaxTurns:     maxTurns,
				MaxBudgetUSD: budget,
			})
			if res != nil {
				fmt.Println(res.Text)
				printRunStats(res, profile.Name)
			}
			if runErr != nil {
				exitCode = 1
				fmt.Fprintln(os.Stderr, "query:", runErr)
				return nil
			}
			if res.IsError {
				exitCode = 1
				return nil
			}
			if save {
				fmt.Println("\nrunning post-write gate (validate + lint)...")
				ok, gerr := gate(root)
				if gerr != nil {
					return gerr
				}
				if !ok {
					exitCode = 1
					fmt.Println("gate: FAILED — saved page(s) need correction")
				} else {
					fmt.Println("gate: passed")
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&save, "save", false, "persist the answer as a synthesis page (write mode)")
	cmd.Flags().IntVar(&maxTurns, "max-turns", 12, "cap the agent's turns")
	cmd.Flags().Float64Var(&budget, "budget", 0, "fail if run cost exceeds this many USD (0 = no ceiling)")
	return cmd
}

func ingestCmd() *cobra.Command {
	var maxTurns int
	var budget float64
	cmd := &cobra.Command{
		Use:   "ingest <raw-file>",
		Short: "Ingest a raw source into the wiki via Claude, then gate the result",
		Long: "Ingest a raw source into the wiki. <raw-file> is the path relative to " +
			"the wiki's raw/ directory. After the agent finishes, the result is " +
			"gated with validate + lint and a created/modified page summary is printed.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if !orchestrate.Available() {
				return fmt.Errorf("%w", orchestrate.ErrClaudeNotFound)
			}

			rawArg := filepath.ToSlash(args[0])
			rawAbs := filepath.Join(root.RawDir(), filepath.FromSlash(rawArg))
			if info, err := os.Stat(rawAbs); err != nil || info.IsDir() {
				return fmt.Errorf("raw source %q not found under %s", rawArg, root.RawDir())
			}
			// Path relative to the wiki/ working directory the agent runs in.
			rawRel := kb.RawDirName + "/" + rawArg

			before, err := snapshotWiki(root)
			if err != nil {
				return err
			}

			fmt.Printf("ingesting %s — agent working...\n", rawRel)
			res, runErr := orchestrate.Run(cmd.Context(), orchestrate.Request{
				Prompt:       orchestrate.IngestPrompt(rawRel),
				Profile:      orchestrate.Ingest,
				WorkDir:      root.WikiDir(),
				MaxTurns:     maxTurns,
				MaxBudgetUSD: budget,
				OnActivity:   printActivity,
			})
			if res != nil {
				fmt.Println(res.Text)
				printRunStats(res, orchestrate.Ingest.Name)
			}
			if runErr != nil {
				exitCode = 1
				fmt.Fprintln(os.Stderr, "ingest:", runErr)
				return nil
			}
			if res.IsError {
				exitCode = 1
				return nil
			}

			after, err := snapshotWiki(root)
			if err != nil {
				return err
			}
			created, modified := diffWiki(before, after)
			printWikiDiff(created, modified)

			fmt.Println("\nrunning post-write gate (validate + lint)...")
			ok, gerr := gate(root)
			if gerr != nil {
				return gerr
			}
			if !ok {
				exitCode = 1
				fmt.Println("gate: FAILED — ingested page(s) need correction")
			} else {
				fmt.Println("gate: passed")
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&maxTurns, "max-turns", 40, "cap the agent's turns")
	cmd.Flags().Float64Var(&budget, "budget", 0, "fail if run cost exceeds this many USD (0 = no ceiling)")
	return cmd
}

// authCmd groups checks for the external dependencies the agent-backed commands
// (query, ingest) require. Today that is the Claude Code executable.
func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Check the Claude Code dependency the agent commands rely on",
	}
	cmd.AddCommand(authStatusCmd())
	return cmd
}

// authStatusCmd reports whether the claude executable is reachable on PATH. It
// exits non-zero when claude is absent so it can gate scripts and CI.
func authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Report whether the claude executable is available on PATH",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := orchestrate.Resolve()
			if err != nil {
				fmt.Println("claude: not found")
				fmt.Println("  query and ingest are unavailable until Claude Code is installed and on PATH")
				exitCode = 1
				return nil
			}
			fmt.Println("claude: found")
			fmt.Printf("  path: %s\n", path)
			return nil
		},
	}
}

// snapshotWiki maps each wiki page's relative path to a content hash, for
// before/after diffing of an agent write.
func snapshotWiki(root *kb.Root) (map[string]string, error) {
	pages, err := wiki.ScanPages(root.WikiDir())
	if err != nil {
		return nil, err
	}
	snap := make(map[string]string, len(pages))
	for _, p := range pages {
		data, err := os.ReadFile(p.Abs)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		snap[p.WikiRel] = hex.EncodeToString(sum[:])
	}
	return snap, nil
}

// diffWiki returns sorted lists of created and modified page paths.
func diffWiki(before, after map[string]string) (created, modified []string) {
	for rel, h := range after {
		old, existed := before[rel]
		switch {
		case !existed:
			created = append(created, rel)
		case old != h:
			modified = append(modified, rel)
		}
	}
	sort.Strings(created)
	sort.Strings(modified)
	return created, modified
}

func printWikiDiff(created, modified []string) {
	fmt.Printf("\nchanges: %d created, %d modified\n", len(created), len(modified))
	for _, c := range created {
		fmt.Printf("  + %s\n", c)
	}
	for _, m := range modified {
		fmt.Printf("  ~ %s\n", m)
	}
}

// gate runs the mechanical checks every write operation must pass: schema
// validation and lint. It prints any problems and reports whether both passed.
func gate(root *kb.Root) (bool, error) {
	sch, err := schema.Load(root.SchemaPath())
	if err != nil {
		return false, err
	}
	pages, err := wiki.ScanPages(root.WikiDir())
	if err != nil {
		return false, err
	}
	sort.SliceStable(pages, func(i, j int) bool { return pages[i].WikiRel < pages[j].WikiRel })

	problems := 0
	for _, p := range pages {
		for _, pr := range sch.ValidatePage(p) {
			fmt.Printf("  validate %s: %s\n", p.WikiRel, pr)
			problems++
		}
	}
	findings := lint.Run(pages)
	for _, f := range findings {
		fmt.Printf("  lint %s\n", f.String())
	}
	return problems == 0 && len(findings) == 0, nil
}

// printActivity renders one agent step as a live progress line during a run.
func printActivity(a orchestrate.Activity) {
	if a.Target != "" {
		fmt.Printf("  · %s %s\n", a.Tool, a.Target)
	} else {
		fmt.Printf("  · %s\n", a.Tool)
	}
}

func printRunStats(res *orchestrate.Result, profile string) {
	fmt.Printf("\n— %s | cost $%.4f | %d turn(s)", profile, res.CostUSD, res.NumTurns)
	if res.SessionID != "" {
		fmt.Printf(" | session %s", res.SessionID)
	}
	fmt.Println()
}

func formatCounts(m map[string]int) string {
	if len(m) == 0 {
		return "(none)"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
