#!/usr/bin/env bash
# Build after a Go source or module file is edited, and report failures to Claude.
set -euo pipefail

file=$(jq -r '.tool_input.file_path // empty')
case "$file" in
  *.go | */go.mod | */go.sum) ;;
  *) exit 0 ;;
esac

cd "$CLAUDE_PROJECT_DIR"
if ! output=$(make build 2>&1); then
  echo "$output" >&2
  exit 2
fi
