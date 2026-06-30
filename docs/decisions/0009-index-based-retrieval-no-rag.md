# 0009 Index-Based Retrieval, No RAG

Date: 2026-06-30

## Status

Accepted

## Context

nvtwiki must let agents and humans find knowledge across many pages. A common
default is embeddings plus a vector database. The target scale is roughly 100
sources and a few hundred pages per project.

## Decision

Use `wiki/index.md` as the navigation entry point. No embeddings, no vector
database, no RAG pipeline. Query reads `index.md` first to locate relevant
pages, then reads those pages directly.

## Alternatives Considered

1. Vector DB + embeddings. Rejected at target scale: operational weight and
   indexing cost without retrieval benefit a curated index does not already
   provide; conflicts with the index-based philosophy.

## Consequences

Positive:

- No extra infrastructure or indexing pipeline to operate.
- Markdown stays the only artifact; git provides versioning and provenance.

Tradeoffs:

- `index.md` can grow toward the context-window limit at large scale.

## Follow-Up

- Monitor `index.md` size; split by category only when it approaches a few
  hundred pages per project. Tracked as PRD Phase E concern, not built now.
