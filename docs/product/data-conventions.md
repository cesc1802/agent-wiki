# nvtwiki — Data Conventions

Source of truth for the frontmatter schema is `schema.yaml` inside `wiki/`.
`CLAUDE.md` and the CLI validator both reference it; they must not encode the
schema independently (see decision 0011).

## Frontmatter

Every page in `wiki/` except `index.md` and `log.md` must carry:

```yaml
---
title: Page title
type: entity | concept | source | synthesis | overview
status: active | superseded | archived
superseded_by: null            # path to replacement page, or null
sources:                       # raw/ files this page is based on
  - raw/<file>.md
created: YYYY-MM-DD
updated: YYYY-MM-DD
---
```

## Cross-reference

Wikilink plus real relative path (Claude Code resolves paths, not `[[ ]]`):

```markdown
Decision based on [[hard-gate-fsm]](../concepts/hard-gate-fsm.md).
```

The validator treats the parenthesized path as the link target for graph
analysis.

## index.md (catalog)

Content-oriented catalog of every page by category (Overview / Entities /
Concepts / Sources / Synthesis). Each line: wikilink + path + one-line summary +
metadata. Updated on every ingest and file-back. It is the navigation entry
point — read first on query. No frontmatter required.

## log.md (append-only timeline)

Grep-friendly entry prefix: `## [YYYY-MM-DD] <op> | <title>` where `op` is one
of `ingest`, `query`, `lint`.

```bash
grep "^## \[" wiki/log.md | tail -5
```

## Language

Page content is Vietnamese; technical terms stay in English. Do not invent
content the source does not state.
