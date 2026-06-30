# nvtwiki — Orchestration Flow

Orchestration commands wrap `claude -p` headless. The CLI prepares context and
the permission profile; Claude reads `CLAUDE.md` from its working directory and
runs the relevant workflow.

## Ingest flow (example)

```text
nvtwiki ingest raw/phase-2-plan.md
  [1] CLI locates wiki/ (walking up from cwd) + builds prompt + sets cwd = wiki/
  [2] CLI invokes claude -p with a locked permission profile
  [3] Claude reads CLAUDE.md (in cwd) -> runs ingest workflow -> writes pages
  [4] CLI parses JSON output {result, session_id, total_cost_usd}
  [5] CLI runs validate + lint (GATE); on fail, reports the missing step
  [6] CLI prints report + cost; human reviews git diff before commit
```

## claude -p invocation contract

The wrapper invokes Claude in headless JSON mode and parses the final result
object. Expected fields consumed by the CLI:

- `result` — the agent's final text answer (surfaced to the user).
- `session_id` — recorded for traceability.
- `total_cost_usd` — used for cost reporting and budget enforcement.

The wrapper sets, per command (see `cli-commands.md`):

- `--allowedTools` — the permission profile tool list.
- `--permission-mode` — `acceptEdits` for ingest; read-only otherwise.
- `--max-turns` — turn cap.
- working directory = `wiki/`.

## Prompt construction

- `query`: include navigation context from `index.md` plus the question;
  instruct citation back to `raw/` sources; file-back only when `--save`.
- `ingest`: name the raw file to ingest and the required workflow output; the
  detailed step checklist lives in `CLAUDE.md`, not duplicated in the prompt.

## Post-run gate

After any write-capable run, the CLI runs `validate` then `lint`. The
orchestration command's success requires both the `claude -p` exit and the gate
to pass.

Related: `safety-guardrails.md`, `cli-commands.md`.
