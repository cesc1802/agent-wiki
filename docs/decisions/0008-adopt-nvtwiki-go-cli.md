# 0008 Adopt nvtwiki As First Product, Go CLI Stack

Date: 2026-06-30

## Status

Accepted

## Context

This harness had no product. The first user-provided spec (`prd.md`) describes
nvtwiki: a CLI orchestrator driving Claude Code to build and maintain a markdown
wiki. The harness requires a stack decision before implementation shape.

## Decision

Adopt nvtwiki as the harness's first product. Implement it as a single Go CLI
binary (module `nvtwiki`), using cobra for the multi-subcommand surface and
`gopkg.in/yaml.v3` for YAML config and frontmatter parsing.

Layering follows `docs/ARCHITECTURE.md`: command parsing (interface) is thin;
scaffold/validate/lint/nav/orchestrate logic lives in `internal/` packages with
no dependency on the CLI layer.

## Alternatives Considered

1. Stdlib `flag` dispatcher only. Rejected: ~12 subcommands with per-command
   flags make a hand-rolled dispatcher more complex than cobra.
2. A different language (Rust/Python). Rejected: the spec specifies Go, and Go's
   single static binary fits a repository-local tool.

## Consequences

Positive:

- Single static binary, easy to run as a repository-local orchestrator.
- Cobra gives consistent help/flag handling across commands.

Tradeoffs:

- Adds cobra + yaml.v3 dependencies (justified by command-surface size).

## Follow-Up

- Build phases A–D per `plans/260630-1117-nvtwiki-build/plan.md`.
- Phase E (semantic lint, full budget accounting, cron) deferred.
