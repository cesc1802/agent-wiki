# nvtwiki

The `nvtwiki` CLI: a deterministic orchestrator that drives Claude Code
(`claude -p`, headless) to build and maintain a wiki-style knowledge base.

Full overview, the CLI/agent split, knowledge-base layout, command reference,
and safety model live in the project README: [`../README.md`](../README.md).

## Develop

```sh
go build -o nvtwiki .   # build (Go 1.21+)
go test ./...           # run the suite
./nvtwiki auth status    # verify the claude CLI is on PATH
```
