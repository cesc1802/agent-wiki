# 0012 Single Self-Contained wiki/ Per Directory

Date: 2026-06-30

## Status

Accepted

## Context

The original model (decisions 0008–0011) made nvtwiki a multi-project knowledge
base: a root marked by `nvtwiki.yaml` held `projects/<name>/{raw,wiki}/`, and
every orchestration command took a project-name argument. In practice each
checkout only ever holds one knowledge base, so the project layer added an
indirection and a required argument without buying isolation that directory
boundaries do not already provide.

## Decision

Collapse the multi-project model to a single self-contained `wiki/` folder per
directory.

- `init [dir]` scaffolds `dir/wiki/` containing `raw/`, `entities/`, `concepts/`,
  `sources/`, `synthesis/`, the seed pages `index.md`/`log.md`/`overview.md`, and
  the control files `schema.yaml`, `CLAUDE.md`, `hooks/block-raw-write.sh`,
  `.claude/settings.json`. `raw/` and the control files live inside `wiki/`.
- The `project` command and `nvtwiki.yaml` are removed. No command takes a
  project argument.
- Every command resolves the target by walking up from the current directory to
  the nearest ancestor containing `wiki/` (`kb.Find`). Orchestration commands run
  `claude -p` with the working directory set to `wiki/`, so the agent addresses
  pages by paths relative to it.

## Alternatives Considered

1. Keep `projects/<name>/` with a default project. Rejected: still requires the
   `nvtwiki.yaml` marker and project plumbing for a layer no checkout uses.
2. Single `wiki/` but keep control files at the repo root above `wiki/`. Rejected:
   the wiki is no longer portable as one folder, and the agent's working
   directory would need to sit above its own protected files.

## Consequences

Positive:

- `wiki/` is portable and self-describing; copying the folder copies everything.
- Commands need no project argument; cwd-relative resolution is the only input.
- Agent working directory equals the protected boundary, simplifying the hook.

Tradeoffs:

- One directory holds one wiki; multiple knowledge bases mean multiple
  directories rather than one shared root.
- `raw/` and the control files now sit inside the scanned tree, so page scanning
  must skip reserved dirs/files (`raw/`, `.claude/`, `hooks/`, `CLAUDE.md`).

## Follow-Up

- Supersedes the multi-project structure in 0008; the `raw/` immutability
  mechanism (0010) and schema single-source rule (0011) carry over, with the hook
  now also guarding the control files and preventing escapes out of `wiki/`.
- Decisions 0008/0009 retain their original multi-project wording as historical
  records.
