#!/usr/bin/env bash
# Advisory checks for a just-edited Go file: license header presence and
# formatting (gofumpt/gci via golangci-lint, when available). Intended as a
# post-edit hook for AI coding harnesses.
#
# Usage: check-go-file.sh <path-of-edited-file>
# Exit 0: no findings. Exit 1: findings printed to stderr.
# Missing tools fail open so the hook never breaks an edit loop.
set -uo pipefail

path="${1:-}"
case "$path" in
  *.go) ;;
  *) exit 0 ;;
esac
[ -f "$path" ] || exit 0
case "$path" in
  *zz_generated*) exit 0 ;;
esac

status=0

expected_header=$'// Copyright The OpenTelemetry Authors\n// SPDX-License-Identifier: Apache-2.0'
if [ "$(head -2 "$path")" != "$expected_header" ]; then
  {
    echo "$path: missing license header. Every Go file must start with:"
    echo '// Copyright The OpenTelemetry Authors'
    echo '// SPDX-License-Identifier: Apache-2.0'
  } >&2
  status=1
fi

repo_root=$(git rev-parse --show-toplevel 2>/dev/null) || repo_root=.
golangci="$repo_root/bin/golangci-lint"
if [ -x "$golangci" ]; then
  # `fmt --diff` prints the diff on stdout and exits 1 when the file needs
  # formatting, so the exit code must not discard the captured output.
  fmt_diff=$("$golangci" fmt --diff "$path" 2>/dev/null || true)
  if [ -n "$fmt_diff" ]; then
    {
      echo "$path: not formatted per project rules (gofumpt/gci). Run 'make fmt', or apply:"
      echo "$fmt_diff" | head -40
    } >&2
    status=1
  fi
fi

exit $status
