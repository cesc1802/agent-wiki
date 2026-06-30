# Knowledge Base — Agent Control File

You are maintaining a project knowledge wiki. This file is the schema layer: it
controls how you read sources and write wiki pages. `nvtwiki` (a deterministic
CLI) orchestrates you and validates your output mechanically afterward.

Your working directory is this wiki/ folder. All paths below are relative to it.

## Three layers

- `raw/` — immutable source material. **Read-only. Never write, edit, move, or
  delete anything under `raw/`.** A code-layer hook blocks it.
- the page tree (`entities/`, `concepts/`, `sources/`, `synthesis/`,
  `index.md`, `log.md`, `overview.md`) — you own this entirely. All pages you
  create or update live here.
- control files (`CLAUDE.md`, `schema.yaml`, `.claude/`, `hooks/`) — these
  configure the orchestrator. **Read-only.** The hook blocks writes to them too.
  `schema.yaml` is the single source of truth for page frontmatter.

## Hard rules

1. Write only page files in the page tree. Writes to `raw/` and the control
   files are blocked.
2. Every page except `index.md` and `log.md` MUST start with frontmatter that
   satisfies `schema.yaml`.
3. Never invent facts the sources do not state. If a source is silent, say so.
4. Page content is in Vietnamese; keep technical terms in English.

## Frontmatter (see schema.yaml)

```yaml
---
title: Page title
type: entity | concept | source | synthesis | overview
status: active | superseded | archived
superseded_by: null            # path to replacement page, or null
sources:
  - raw/<file>.md
created: YYYY-MM-DD
updated: YYYY-MM-DD
---
```

## Cross-references

Link with a wikilink plus the real relative path:

```markdown
See [[hard-gate-fsm]](../concepts/hard-gate-fsm.md).
```

## index.md and log.md

- `index.md` is the catalog: every page listed by category (Overview, Entities,
  Concepts, Sources, Synthesis) with its path and a one-line summary. Update it
  whenever you add or change a page. It is read first on every query.
- `log.md` is an append-only timeline. Append an entry per operation with the
  prefix `## [YYYY-MM-DD] <op> | <title>` where `<op>` is `ingest`, `query`, or
  `lint`.

## Ingest workflow

When asked to ingest a raw source file, run these steps in order:

1. Read the named raw file fully.
2. Read `index.md` to learn what already exists.
3. Create a `sources/` page summarizing the raw file (type `source`, with the
   raw path in `sources:`).
4. Create or update `entities/` pages for the systems, components, and people
   the source introduces.
5. Create or update `concepts/` pages for the ideas, patterns, and decisions.
6. Add cross-reference links between related pages (both directions where it
   helps navigation).
7. If a new page supersedes an old one, set the old page's `status: superseded`
   and `superseded_by:` to the new page's path.
8. Update `index.md` to catalog every new or changed page.
9. Append an `ingest` entry to `log.md`.

Do not skip steps. The CLI runs `validate` and `lint` after you finish; missing
frontmatter, broken links, or orphan pages fail the gate.

## Query workflow

When asked a question:

1. Read `index.md` to locate relevant pages.
2. Read those pages.
3. Answer concisely, citing the wiki pages and the underlying `raw/` sources.
4. Only when explicitly told to save: create a `synthesis/` page capturing the
   answer, update `index.md`, and append a `query` entry to `log.md`.

If you cannot answer from the wiki, say what is missing rather than guessing.
