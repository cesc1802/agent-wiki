# nvtwiki — Safety & Guardrails

## Protecting the source of truth

`raw/` and the control files (`schema.yaml`, `CLAUDE.md`, `.claude/`, `hooks/`)
are immutable to the agent; it may write only page files. Protection is layered
and does not rely on model judgment (boundaries in a prompt can be lost on
context compaction):

1. **Deny rule** in the per-command permission config blocks writes matching
   `raw/**`. Hard guarantee at the permission layer.
2. **PreToolUse hook** (`hooks/block-raw-write.sh`, inside `wiki/`) blocks at the
   code layer any `Write`/`Edit` that escapes the `wiki/` working directory,
   lands under `raw/`, or targets a control file. Installed by `init`.
3. **No `Bash` tool** for `ingest`/`query`/`lint` — blocks `rm`, `git push`,
   and arbitrary code execution.

The hook is authoritative: even if a deny rule is misconfigured, the hook
rejects the tool call. The adversarial test (tell Claude to edit a `raw/` file)
must be blocked.

## Cost control

- `--max-turns` caps agent turns so a loop cannot silently burn quota.
- `--max-budget-usd` per run; the wrapper parses `total_cost_usd` from the
  `claude -p` JSON output and reports/accumulates it.
- `claude -p` self-aborts on repeated blocked tool calls; the CLI surfaces that
  as an error rather than a hang.

## Human-in-the-loop

Autonomous mode shifts human control from "approve each action" to "review the
diff after the run":

- Every change lands in the working tree; the human reviews `git diff` before
  committing. **Git is the human-in-the-loop.**
- No auto-commit, no auto-push.
- Post-run gate: `ingest` runs `validate` + `lint`; failure marks the ingest
  incomplete.

Related: decision 0010 (`raw/` immutability mechanism).
