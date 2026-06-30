# nvtwiki — Overview

nvtwiki is a Go CLI that acts as a deterministic **orchestrator** for Claude
Code (`claude -p` headless). It builds and maintains a wiki-style knowledge base
for software projects so that plans, phases, designs, and decisions stay
queryable and resist decay.

## Three-layer model

Everything lives inside a single self-contained `wiki/` folder; the directory
holding it is the project, located by walking up from the current directory.

| Layer | Path (inside `wiki/`) | Owner | Mutability |
| --- | --- | --- | --- |
| Raw sources | `raw/` | human | immutable, read-only to agent |
| Wiki pages | `entities/`, `concepts/`, `sources/`, `synthesis/`, `index.md`, `log.md`, `overview.md` | Claude | fully owned by the LLM |
| Schema | `CLAUDE.md`, `schema.yaml` | maintainer | read-only to agent; controls its behavior |

## Responsibility split

CLI does the mechanical, deterministic, gate-enforcing work; Claude does the
semantic work (reading, writing, synthesizing, conflict detection).

| Concern | CLI | Claude |
| --- | --- | --- |
| Scaffold structure | yes | |
| Validate frontmatter syntax | yes | |
| Link-graph: orphans, broken links | yes | |
| Compare `status` / `superseded_by` | yes | |
| Query log, stats, list raw | yes | |
| Read & understand source content | | yes |
| Write / update pages | | yes |
| Detect content-level contradictions | | yes |
| Decide file-back, synthesize answers | | yes |

## Core invariant

The CLI never edits page content. Claude only writes page files — it never
writes into `raw/` or the control files. This is enforced mechanically (see
`safety-guardrails.md`), not by prompt instruction alone.

## Architecture pattern

Hard-gate FSM: the CLI is the gate, Claude is the executor. The CLI prepares
context, forces the workflow, and validates the result; Claude reads, writes,
and synthesizes.

Related: `cli-commands.md`, `data-conventions.md`, `orchestration.md`,
`safety-guardrails.md`.
