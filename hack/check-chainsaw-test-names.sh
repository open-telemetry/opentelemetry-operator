#!/usr/bin/env bash

# Fails if two chainsaw tests share the same metadata.name.
#
# chainsaw silently renames colliding test names to <name>#NN at runtime. That
# suffix cannot be mapped back to a test directory in the JUnit/Codecov reports,
# which makes flaky-test triage harder (you cannot tell which directory the
# failing "<name>#01" came from). Keeping names unique avoids that.

set -euo pipefail

cd "$(dirname "$0")/.."

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

# Extract metadata.name (the first "  name:" after the "metadata:" line) from
# every chainsaw test and record it alongside its file.
while IFS= read -r f; do
  name=$(awk '/^metadata:/{m=1; next} m && /^  name:/{print $2; exit}' "$f")
  if [ -z "$name" ]; then
    echo "ERROR: $f has no metadata.name" >&2
    exit 1
  fi
  printf '%s\t%s\n' "$name" "$f" >>"$tmp"
done < <(find tests -name chainsaw-test.yaml | sort)

dupes=$(cut -f1 "$tmp" | sort | uniq -d)
if [ -n "$dupes" ]; then
  echo "ERROR: chainsaw tests with duplicate metadata.name found." >&2
  echo "chainsaw renames collisions to <name>#NN, which cannot be traced back to a" >&2
  echo "directory in JUnit/Codecov reports. Give each test a unique metadata.name:" >&2
  while IFS= read -r d; do
    echo "  ${d}:" >&2
    awk -F'\t' -v d="$d" '$1 == d {print "    " $2}' "$tmp" >&2
  done <<<"$dupes"
  exit 1
fi

echo "OK: $(wc -l <"$tmp" | tr -d ' ') chainsaw tests, all metadata.name unique"
