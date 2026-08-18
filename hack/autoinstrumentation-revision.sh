#!/usr/bin/env bash

# Canonical helpers for the operator-owned revision suffix on autoinstrumentation
# image tags (<sdk-version>-<revision>, e.g. 2.30.0-1). See
# autoinstrumentation/README.md for the tagging scheme.
#
# Usage:
#   hack/autoinstrumentation-revision.sh languages
#       List the languages that use the <sdk-version>-<revision> tag scheme
#       (discovered from autoinstrumentation/<language>/revision.txt).
#   hack/autoinstrumentation-revision.sh sdk-version <language>
#       Print the upstream SDK version for a language.
#   hack/autoinstrumentation-revision.sh revision <language>
#       Print the current operator-owned revision for a language.
#   hack/autoinstrumentation-revision.sh check
#       Validate revision bumps for a pull request. Diffs against BASE_SHA, or
#       the merge-base with the target branch (origin/main, then main, or
#       TARGET_BRANCH) when BASE_SHA is unset.

set -euo pipefail

cd "$(dirname "$0")/.."

# discover_languages populates the LANGUAGES array with every language that has a
# revision.txt. Go (upstream image) and nginx (shares apache-httpd) have none.
discover_languages() {
  shopt -s nullglob
  LANGUAGES=()
  local f
  for f in autoinstrumentation/*/revision.txt; do
    LANGUAGES+=("$(basename "$(dirname "$f")")")
  done
  if (( ${#LANGUAGES[@]} == 0 )); then
    echo "No autoinstrumentation/*/revision.txt files found." >&2
    return 1
  fi
}

# source_file <language> prints the path to the file holding the upstream SDK
# version. New languages default to version.txt; non-standard sources must be
# registered here and in extract_version.
source_file() {
  case "$1" in
    python) echo "autoinstrumentation/python/requirements.txt" ;;
    nodejs) echo "autoinstrumentation/nodejs/package.json" ;;
    *) echo "autoinstrumentation/$1/version.txt" ;;
  esac
}

# extract_version <language> <source-contents> prints the SDK version parsed from
# the given source file contents.
extract_version() {
  local lang="$1" source="$2"
  case "$lang" in
    python) printf '%s' "$source" | head -n 1 | cut -d '=' -f 3 | tr -d '[:space:]' ;;
    nodejs) printf '%s' "$source" | jq -r '.dependencies."@opentelemetry/auto-instrumentations-node"' ;;
    *) printf '%s' "$source" | tr -d '[:space:]' ;;
  esac
}

# sdk_version <language> prints the SDK version from the working tree.
sdk_version() {
  local lang="$1" src
  src="$(source_file "$lang")"
  if [[ ! -f "$src" ]]; then
    echo "no SDK version source found at '$src' for '$lang'; register it in source_file()/extract_version() in hack/autoinstrumentation-revision.sh" >&2
    return 1
  fi
  extract_version "$lang" "$(cat "$src")"
}

# revision <language> prints the current revision from the working tree.
revision() {
  tr -d '[:space:]' < "autoinstrumentation/$1/revision.txt"
}

cmd_languages() {
  discover_languages
  printf '%s\n' "${LANGUAGES[@]}"
}

cmd_sdk_version() {
  local lang="${1:?usage: sdk-version <language>}"
  echo "$(sdk_version "$lang")"
}

cmd_revision() {
  local lang="${1:?usage: revision <language>}"
  echo "$(revision "$lang")"
}

cmd_check() {
  if [[ -z "${BASE_SHA:-}" ]]; then
    local ref
    for ref in "${TARGET_BRANCH:-}" origin/main main; do
      [[ -z "$ref" ]] && continue
      if BASE_SHA="$(git merge-base HEAD "$ref" 2>/dev/null)"; then
        break
      fi
    done
    if [[ -z "${BASE_SHA:-}" ]]; then
      echo "Could not determine a base commit; set BASE_SHA (or TARGET_BRANCH) explicitly." >&2
      exit 1
    fi
  fi

  discover_languages

  # show_at_base <path> prints file contents at BASE_SHA, empty if absent there.
  show_at_base() {
    git show "$BASE_SHA:$1" 2>/dev/null || true
  }

  local errors=0 lang dir rev_file src_file head_rev head_sdk base_rev base_sdk content_changed
  for lang in "${LANGUAGES[@]}"; do
    dir="autoinstrumentation/$lang"
    rev_file="$dir/revision.txt"
    src_file="$(source_file "$lang")"

    if [[ ! -f "$src_file" ]]; then
      echo "::error file=$rev_file::no SDK version source found at '$src_file' for '$lang'; register it in source_file()/extract_version() in hack/autoinstrumentation-revision.sh"
      errors=$((errors + 1))
      continue
    fi

    head_rev="$(revision "$lang")"
    head_sdk="$(extract_version "$lang" "$(cat "$src_file")")"
    base_rev="$(show_at_base "$rev_file" | tr -d '[:space:]')"
    base_sdk="$(extract_version "$lang" "$(show_at_base "$src_file")")"

    if ! [[ "$head_rev" =~ ^[1-9][0-9]*$ ]]; then
      echo "::error file=$rev_file::$rev_file must contain a positive integer, got '$head_rev'"
      errors=$((errors + 1))
      continue
    fi

    # Any image content (anything under the language dir except revision.txt) changed?
    content_changed="$(git diff --name-only "$BASE_SHA" -- "$dir" ":(exclude)$rev_file")"

    if [[ -n "$base_sdk" && "$head_sdk" != "$base_sdk" ]]; then
      # Upstream SDK version changed -> revision must reset to 1.
      if [[ "$head_rev" != "1" ]]; then
        echo "::error file=$rev_file::$lang SDK version changed ($base_sdk -> $head_sdk); reset $rev_file to 1"
        errors=$((errors + 1))
      fi
    elif [[ -n "$content_changed" && -n "$base_rev" ]]; then
      # Image content changed with the same SDK version -> revision must increment.
      if (( head_rev <= base_rev )); then
        echo "::error file=$rev_file::$lang image content changed with no SDK version bump; increment $rev_file ($base_rev -> $((base_rev + 1)))"
        echo "$content_changed" | sed 's/^/  changed: /'
        errors=$((errors + 1))
      fi
    fi
  done

  if (( errors > 0 )); then
    echo "Found $errors autoinstrumentation revision problem(s). See https://github.com/open-telemetry/opentelemetry-operator/blob/main/autoinstrumentation/README.md#image-tagging"
    exit 1
  fi

  echo "All autoinstrumentation revisions are valid."
}

case "${1:-}" in
  languages) shift; cmd_languages "$@" ;;
  sdk-version) shift; cmd_sdk_version "$@" ;;
  revision) shift; cmd_revision "$@" ;;
  check) shift; cmd_check "$@" ;;
  *)
    echo "usage: $0 {languages|sdk-version <language>|revision <language>|check}" >&2
    exit 1
    ;;
esac
