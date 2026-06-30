# agent-wiki

`nvtwiki` is a deterministic CLI orchestrator that drives Claude Code (`claude
-p`, headless) to build and maintain a wiki-style knowledge base for a software
project.

The split is the whole point:

- **The CLI owns the mechanical gates** — scaffolding, frontmatter validation,
  link/orphan/supersession lint, navigation, and the post-write gate. It never
  edits page content.
- **Claude owns the semantics** — reading raw sources and writing wiki pages.
  The agent can never write into `raw/`.

The application lives in [`app/`](app/). This repository is itself developed
with a repository harness — see the [Harness](#harness) section.

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

## Install

The fastest path is the install script, which downloads a prebuilt binary from
[GitHub Releases](https://github.com/cesc1802/agent-wiki/releases), verifies its
sha256 checksum, and installs it. It supports macOS, Linux, and Windows (Git
Bash / MSYS / Cygwin) on amd64 and arm64.

```sh
# Latest release into ~/.local/bin
curl -fsSL https://raw.githubusercontent.com/cesc1802/agent-wiki/master/install.sh | sh
```

To install as a global binary you can call from anywhere, add `--global` (installs
to `/usr/local/bin`, which is on `PATH` by default; uses `sudo` when needed):

```sh
curl -fsSL https://raw.githubusercontent.com/cesc1802/agent-wiki/master/install.sh | sh -s -- --global
```

To pin a specific version, pass `--version` (accepts `v1.2.3` or `1.2.3`):

```sh
curl -fsSL https://raw.githubusercontent.com/cesc1802/agent-wiki/master/install.sh | sh -s -- --version v1.2.3 --global
```

Options (everything after `sh -s --` is passed to the script):

| Flag | Description |
| --- | --- |
| `-v, --version <tag>` | Release tag to install (`v1.2.3` or `1.2.3`). Default: latest release. |
| `-g, --global` | Install system-wide to `/usr/local/bin`; uses `sudo` when the directory is not writable. |
| `-d, --bindir <dir>` | Install directory. Default: `$HOME/.local/bin` (or `/usr/local/bin` with `--global`). |
| `-h, --help` | Show help and exit. |

The `VERSION` and `BINDIR` environment variables are honored as fallbacks when
the matching flag is absent. If the install directory is not on your `PATH`, the
script prints the exact `export PATH=...` line to add to your shell profile.

After installing, confirm it resolves:

```sh
nvtwiki --version
```

## Build

To build from source instead:

```sh
cd app
go build -o nvtwiki .
```

Requires Go 1.21+. The orchestration commands (`query`, `ingest`) additionally
require the `claude` CLI on PATH (verify with `nvtwiki auth status`); the
raw-write hook requires `jq`.

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
nvtwiki auth status                # check the claude executable is on PATH
```

`auth status` reports whether the `claude` CLI the orchestration commands need
is installed (and its resolved path), exiting non-zero when it is absent so
scripts and CI can gate on it.

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
`claude -p` argument and stream-event parsing contract, claude PATH resolution,
and an adversarial test that the raw-write hook actually blocks writes into
`raw/`.

## Harness

This repository is developed with a repository harness: a set of files that give
coding agents the project context they need before changing code — where to
start, how risky the work is, and what proof is required. Start with:

- [`AGENTS.md`](AGENTS.md) — stable agent shim with local notes and doc links.
- [`docs/HARNESS.md`](docs/HARNESS.md) — the human-agent collaboration model.
- [`docs/FEATURE_INTAKE.md`](docs/FEATURE_INTAKE.md) — tiny/normal/high-risk work
  classification.
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) and
  [`docs/CONTEXT_RULES.md`](docs/CONTEXT_RULES.md) — boundary and per-lane
  reading rules.
- [`docs/TOOL_REGISTRY.md`](docs/TOOL_REGISTRY.md) — optional external tool model.
- `scripts/bin/harness-cli query matrix` — behavior-to-proof validation status.
