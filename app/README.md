# nvtwiki

A deterministic CLI orchestrator that drives Claude Code (`claude -p`, headless)
to build and maintain a wiki-style knowledge base for a software project.

The split is the whole point:

- **The CLI owns the mechanical gates** — scaffolding, frontmatter validation,
  link/orphan/supersession lint, navigation, and the post-write gate. It never
  edits page content.
- **Claude owns the semantics** — reading raw sources and writing wiki pages.
  The agent can never write into `raw/`.

## Layout of a knowledge base

A project owns a single self-contained `wiki/` folder. There is no multi-project
layout: the directory that holds `wiki/` is the project, and the CLI locates it
by walking up from the current directory.

```
<project>/
  wiki/                   # the whole knowledge base, located by the CLI
    raw/                  # immutable sources — read-only to the agent
    entities/  concepts/  sources/  synthesis/   # agent-owned pages
    index.md  log.md  overview.md
    schema.yaml           # single source of truth for page frontmatter
    CLAUDE.md             # agent control file (the schema layer)
    hooks/block-raw-write.sh
    .claude/settings.json # registers the PreToolUse hook
```

Three layers inside `wiki/`: immutable `raw/`, agent-owned pages, and the schema
layer (`CLAUDE.md` + `schema.yaml`). Orchestration runs `claude -p` with its
working directory set to `wiki/`. Hard invariant: the CLI never edits page
content; the agent may only write page files — `raw/` and the control files are
read-only and a hook enforces it.

## Build

```sh
cd app
go build -o nvtwiki .
```

Requires Go 1.21+. The orchestration commands (`query`, `ingest`) additionally
require the `claude` CLI on PATH; the raw-write hook requires `jq`.

## Commands

Run any command from the project directory or any subdirectory of it; the CLI
walks up to find `wiki/`. No project argument is needed.

Deterministic (no LLM):

```sh
nvtwiki init [dir]                 # scaffold a self-contained wiki/ in dir (default .)
nvtwiki validate                   # frontmatter vs schema.yaml
nvtwiki lint                       # orphans, broken links, superseded_by/status
nvtwiki status                     # page stats + ingest progress
nvtwiki log [-n N] [--op ingest|query|lint]
nvtwiki raw                        # raw sources not yet ingested
```

Orchestration (call `claude -p`):

```sh
# Ingest one raw source into the wiki, then gate the result.
nvtwiki ingest <raw-file> [--max-turns 40] [--budget 0]

# Answer a question from the wiki (read-only by default).
nvtwiki query "<question>" [--save] [--max-turns 12] [--budget 0]
```

`<raw-file>` is relative to the wiki's `raw/` directory. `--save` lets a query
persist a synthesis page (write mode + post-write gate). `--budget` fails the
command if the run's reported cost exceeds the given USD ceiling; `--max-turns`
is the real spend control.

## Safety

- `block-raw-write.sh` is a PreToolUse hook that denies any `Write`/`Edit`/
  `MultiEdit` whose target escapes the `wiki/` directory, lands under `raw/`, or
  hits a control file (`schema.yaml`, `CLAUDE.md`, `.claude/`, `hooks/`). The
  agent may only write page files. It is a code-layer guarantee independent of
  prompt instructions.
- Orchestration commands are never granted the `Bash` tool.
- The CLI only ever creates scaffold files; it never rewrites page content.

## Tests

```sh
cd app
go test ./...
```

Covers frontmatter parsing, schema validation, lint rules, navigation, the
`claude -p` argument/JSON contract, and an adversarial test that the raw-write
hook actually blocks writes into `raw/`.
