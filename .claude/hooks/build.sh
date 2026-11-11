#!/usr/bin/env bash
# Build after a Go source or module file is changed by Edit / Write / MultiEdit,
# and report failures to Claude. Their input names the file as "file_path".
set -euo pipefail

changed=false
while IFS= read -r file; do
  case "$file" in
    *.go | *go.mod | *go.sum) changed=true ;;
  esac
done < <(jq -r '.tool_input.file_path // empty')

if [ "$changed" = false ]; then
  exit 0
fi

cd "$CLAUDE_PROJECT_DIR"
if ! output=$(qsoku build 2>&1); then
  echo "$output" >&2
  exit 2
fi
