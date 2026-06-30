# 0011 schema.yaml As Single Source of Truth for Frontmatter

Date: 2026-06-30

## Status

Accepted

## Context

The frontmatter convention is encoded in three places that can drift: the
agent-facing `CLAUDE.md`, the CLI validator, and the orchestration prompt
template. Drift is the spec's top risk (PRD R1): inconsistent rules across these
surfaces produce false lint results and erode trust.

## Decision

`schema.yaml` at the knowledge-base root is the single source of truth for the
frontmatter schema (required fields, types, enum values). The CLI validator
loads `schema.yaml` at runtime and validates against it — it does not hardcode
field rules. `CLAUDE.md` references the same schema rather than restating it as
independent truth.

## Alternatives Considered

1. Hardcode the schema in the Go validator. Rejected: a fourth divergent copy;
   schema changes would require recompiling the binary.
2. Generate `CLAUDE.md` and the validator from one source at build time.
   Deferred: more machinery than needed now; runtime read of `schema.yaml` plus
   a documentation reference is sufficient at this scale.

## Consequences

Positive:

- One file to change when the schema evolves; validator picks it up at runtime.
- Reduces R1 drift risk to keeping `CLAUDE.md` as a reference, not a copy.

Tradeoffs:

- The validator must handle a missing/malformed `schema.yaml` with a clear
  error rather than silently passing.

## Follow-Up

- Implement schema loading + validation in Phase A.
- `init` writes the canonical `schema.yaml`.
