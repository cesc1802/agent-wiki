#!/usr/bin/env bash
# block-raw-write.sh — Claude Code PreToolUse hook.
#
# Enforces the hard invariant: the agent may write only page files inside the
# wiki/ working directory. Immutable source material (raw/) and the control
# files (schema.yaml, CLAUDE.md, .claude/, hooks/) are denied, as is any path
# that escapes the wiki/ directory. This is a code-layer guarantee that does not
# depend on prompt instructions or model judgment.
#
# claude runs with its working directory set to the wiki/ folder, so the agent
# addresses pages by paths relative to it (e.g. concepts/x.md) or absolute paths
# that contain a /wiki/ segment. Both are handled below.
#
# Requires jq. Reads the PreToolUse JSON event on stdin.
set -euo pipefail

input=$(cat)

tool=$(printf '%s' "$input" | jq -r '.tool_name // empty')
case "$tool" in
  Write | Edit | MultiEdit) ;;
  *) exit 0 ;; # not a write tool; allow
esac

path=$(printf '%s' "$input" | jq -r '.tool_input.file_path // .tool_input.path // empty')
[ -z "$path" ] && exit 0

# Deny reserved targets first, wherever they sit in the path: immutable raw/
# sources and the control files that configure the orchestrator.
case "$path" in
  raw/* | */raw/*) deny=1 ;;
  .claude/* | */.claude/*) deny=1 ;;
  hooks/* | */hooks/*) deny=1 ;;
  schema.yaml | */schema.yaml) deny=1 ;;
  CLAUDE.md | */CLAUDE.md) deny=1 ;;
  *..*) deny=1 ;; # parent-traversal escape attempt
  /*)
    # Absolute path: allow only when it lands inside a wiki/ directory.
    case "$path" in
      */wiki/*) exit 0 ;;
      *) deny=1 ;;
    esac
    ;;
  *) exit 0 ;; # relative path inside the wiki working dir: allow
esac

[ "${deny:-0}" -eq 1 ] || exit 0

reason="nvtwiki: write to '$path' is blocked. The agent may only write page files inside the wiki/ working directory; raw/ sources and control files (schema.yaml, CLAUDE.md, .claude/, hooks/) are read-only."
jq -n --arg r "$reason" \
  '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'
exit 2
