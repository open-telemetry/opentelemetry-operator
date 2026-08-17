#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${BASE_SHA:-}" ]]; then
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

shopt -s nullglob
LANGUAGES=()
for f in autoinstrumentation/*/revision.txt; do
  LANGUAGES+=("$(basename "$(dirname "$f")")")
done
if (( ${#LANGUAGES[@]} == 0 )); then
  echo "No autoinstrumentation/*/revision.txt files found." >&2
  exit 1
fi

source_file() {
  case "$1" in
    python) echo "autoinstrumentation/python/requirements.txt" ;;
    nodejs) echo "autoinstrumentation/nodejs/package.json" ;;
    *) echo "autoinstrumentation/$1/version.txt" ;;
  esac
}

read_sdk_version() {
  local lang="$1" source="$2"
  case "$lang" in
    python) printf '%s' "$source" | head -n 1 | cut -d '=' -f 3 | tr -d '[:space:]' ;;
    nodejs) printf '%s' "$source" | jq -r '.dependencies."@opentelemetry/auto-instrumentations-node"' ;;
    *) printf '%s' "$source" | tr -d '[:space:]' ;;
  esac
}

show_at_base() {
  git show "$BASE_SHA:$1" 2>/dev/null || true
}

errors=0

for lang in "${LANGUAGES[@]}"; do
  dir="autoinstrumentation/$lang"
  rev_file="$dir/revision.txt"
  src_file="$(source_file "$lang")"

  # New languages default to version.txt; non-standard sources (python, nodejs)
  # must be registered in source_file()/read_sdk_version() above.
  if [[ ! -f "$src_file" ]]; then
    echo "::error file=$rev_file::no SDK version source found at '$src_file' for '$lang'; register it in source_file()/read_sdk_version() in .github/workflows/scripts/check-autoinstrumentation-revision.sh"
    errors=$((errors + 1))
    continue
  fi

  head_rev="$(tr -d '[:space:]' < "$rev_file")"
  head_sdk="$(read_sdk_version "$lang" "$(cat "$src_file")")"

  base_rev="$(show_at_base "$rev_file" | tr -d '[:space:]')"
  base_sdk="$(read_sdk_version "$lang" "$(show_at_base "$src_file")")"

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
