# nvtwiki — CLI Command Contract

Public command surface. Two groups: deterministic (no LLM) and orchestration
(calls `claude -p`).

Each project owns a single self-contained `wiki/` folder. Commands take no
project argument; the CLI locates the wiki by walking up from the current
directory to the nearest folder containing `wiki/`.

## Deterministic group (no LLM)

| Command | Behavior |
| --- | --- |
| `init [dir]` | Scaffold a self-contained `wiki/` in `dir` (default `.`): `index.md`, `log.md`, `overview.md`, the `raw/`, `entities/`, `concepts/`, `sources/`, `synthesis/` directories, and the control files `schema.yaml`, `CLAUDE.md`, `hooks/block-raw-write.sh`, `.claude/settings.json` (registers the hook) |
| `validate` | Check every page's frontmatter against `schema.yaml` |
| `lint` | Mechanical lint: orphan pages, broken links, bad/cyclic `superseded_by`, status contradictions |
| `status` | Counts by type/status; ingested vs not-yet-ingested sources |
| `log [-n N] [--op ingest\|query\|lint]` | Tail/filter `log.md` timeline |
| `raw` | List `raw/` files with no corresponding source page (ingest debt) |

## Orchestration group (calls `claude -p`)

`claude -p` runs with its working directory set to `wiki/`, so the agent
addresses pages by paths relative to it.

| Command | Permission profile | Write scope |
| --- | --- | --- |
| `ingest <raw-file>` | `Read,Write,Edit,MultiEdit,Glob,Grep`; mode `acceptEdits`; `--max-turns` default 40 | wiki pages only |
| `query "<q>" [--save]` | read-only `Read,Glob,Grep` (`--max-turns` default 12); `--save` switches to the write profile for file-back | none unless `--save` |

`<raw-file>` is the path relative to the wiki's `raw/` directory.

Deferred to Phase E: `lint --semantic` (LLM-assisted semantic lint).

## Flags (orchestration)

- `--max-turns <n>` — cap agent turns (loop / quota protection).
- `--budget <f>` — fail the command (non-zero exit) when the run's
  `total_cost_usd` from the `claude -p` JSON output exceeds this many USD. The
  agent's answer/changes are still surfaced; the check is post-run, so it bounds
  reporting, not mid-run spend. `--max-turns` is the real spend control. `0`
  (default) means no ceiling.

## Exit behavior

- Deterministic commands exit non-zero when validation/lint finds problems.
- `ingest` (and `query --save`) run a post-run validate+lint gate; gate failure
  means the operation is reported as incomplete (non-zero exit), even if
  `claude -p` succeeded.

Never grants the `Bash` tool to orchestration commands.
