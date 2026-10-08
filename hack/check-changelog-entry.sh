#!/usr/bin/env bash
# Checks that a branch which changes code also adds a changelog entry under
# .chloggen/, counting both committed and not-yet-committed entries. Runs as
# the last step of `make precommit`; CI enforces the same rule via the
# changelog workflow, which can only be skipped with '[chore]' in the PR title
# or the 'Skip Changelog' label.
#
# Skip locally with: CHLOG=skip make precommit
set -euo pipefail

if [ "${CHLOG:-}" = "skip" ]; then
  exit 0
fi

base=$(git merge-base HEAD origin/main 2>/dev/null) \
  || base=$(git merge-base HEAD main 2>/dev/null) \
  || exit 0

# A changelog entry counts whether it is committed, staged, or still untracked.
if git diff --diff-filter=A --name-only "$base" -- .chloggen | grep -q '\.yaml$'; then
  exit 0
fi
if git ls-files --others --exclude-standard .chloggen | grep -q '\.yaml$'; then
  exit 0
fi

# Only nudge when the branch actually changes code the changelog could cover:
# Go sources and manifests, excluding tests, docs, and generated trees.
changed=$( (git diff --name-only "$base"; git ls-files --others --exclude-standard) \
  | grep -vE '^(docs/|tests/|bundle/|config/)|_test\.go$|zz_generated|\.md$' \
  | grep -cE '\.(go|ya?ml)$' || true)
if [ "$changed" -eq 0 ]; then
  exit 0
fi

cat >&2 <<'EOF'
No changelog entry found for this branch, but it changes code.

User-facing changes (operator or operand behavior, CRD schemas, defaults,
flags, feature gates) need an entry:

    make chlog-new       # creates .chloggen/<branch>.yaml
    # fill in change_type, component, note, and issues, then:
    make chlog-validate

If this change genuinely needs no entry (test-only, CI, docs, internal
refactoring), skip this check with:

    CHLOG=skip make precommit

and say why in the PR description; add '[chore]' to the PR title or the
'Skip Changelog' label so CI skips it too.
EOF
exit 1
