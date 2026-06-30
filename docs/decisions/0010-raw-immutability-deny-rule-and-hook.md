# 0010 raw/ Immutability via Deny Rule and PreToolUse Hook

Date: 2026-06-30

## Status

Accepted

## Context

`raw/` holds immutable source material. The hard invariant is that the agent
never writes into `raw/`. Relying on prompt instructions is unsafe: boundary
text can be lost on context compaction, and the agent has write tools during
ingest.

## Decision

Protect `raw/` with two independent mechanical layers, neither depending on
model judgment:

1. A deny rule in the per-command permission config blocking `raw/**` writes.
2. A PreToolUse hook (`hooks/block-raw-write.sh`, installed by `init`) that
   inspects `Write`/`Edit` tool calls and rejects any target path outside the
   active project's `wiki/`.

Additionally, orchestration commands are never granted the `Bash` tool, which
removes file mutation via shell.

The hook is authoritative: it must block a `raw/` write even if the deny rule is
misconfigured. This is validated by an adversarial test that instructs the agent
to edit a `raw/` file and asserts the write is blocked.

## Alternatives Considered

1. Prompt-only instruction ("do not write to raw/"). Rejected: not durable
   across compaction; no mechanical guarantee.
2. Filesystem read-only permissions on `raw/`. Rejected as the sole mechanism:
   OS-level and brittle across environments; the hook is portable and testable.
   May complement the hook later.

## Consequences

Positive:

- Source of truth is mechanically protected regardless of model behavior.
- Adversarial test gives durable proof the boundary holds.

Tradeoffs:

- Two layers must stay consistent with the project path layout.
- The hook must correctly resolve relative/absolute paths to avoid false
  negatives.

## Follow-Up

- Implement and adversarially test in Phase B.
