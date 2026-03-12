#!/bin/bash
# Create a release issue for the next operator version.
#
# Usage:
#   hack/create-release-issue.sh [--version VERSION] [--dry-run]
#
# Options:
#   --version VERSION   Override the version (e.g. 0.146.0). If empty, auto-detected from RELEASE.md.
#   --dry-run           Print what would happen without creating an issue.

set -euo pipefail

REPO="${GITHUB_REPOSITORY:-open-telemetry/opentelemetry-operator}"
SERVER="${GITHUB_SERVER_URL:-https://github.com}"
VERSION=""
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --version) VERSION="$2"; shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# --- Step 1: Parse RELEASE.md ---
NEXT_LINE=$(awk '/^\| v[0-9]/' RELEASE.md | head -1)
PARSED_VERSION=$(echo "$NEXT_LINE" | awk -F'|' '{print $2}' | xargs | sed 's/^v//')
MANAGER=$(echo "$NEXT_LINE" | awk -F'|' '{print $3}' | xargs | sed 's/^@//')

VERSION="${VERSION:-$PARSED_VERSION}"

if [ -z "$VERSION" ]; then
  echo "Error: could not determine version from RELEASE.md and no --version given." >&2
  exit 1
fi

echo "Version: $VERSION"
echo "Release manager: $MANAGER"

# --- Step 2: Check if collector release exists ---
MINOR=$(echo "${VERSION}" | cut -d. -f1-2)
RELEASED=$(gh api "repos/open-telemetry/opentelemetry-collector-releases/releases" \
  --jq "[.[] | select(.tag_name | startswith(\"v${MINOR}.\")) | select(.draft | not)] | length")

if [ "$RELEASED" -gt 0 ]; then
  echo "Collector release for v${MINOR}.x: found"
else
  echo "Collector release for v${MINOR}.x: not found"
  echo "No matching collector release yet. Use --version to override or wait for the collector release."
  exit 0
fi

# --- Step 3: Check for existing issue ---
EXISTING=$(gh issue list --repo "$REPO" --search "Prepare release v${VERSION} in:title" --state open --json number --jq 'length')
if [ "$EXISTING" -gt 0 ]; then
  echo "Issue already exists for v${VERSION}, skipping."
  exit 0
fi

# --- Step 4: Create the issue ---
BODY="## Release v${VERSION}

Release manager: @${MANAGER}

### Checklist

- [ ] Create the \`[chore] Prepare release v${VERSION}\` PR ([instructions](${SERVER}/${REPO}/blob/main/RELEASE.md))
- [ ] After merge: publish the auto-created draft GitHub release once all release workflows complete
- [ ] Update the operator version in the [Helm Chart](https://github.com/open-telemetry/opentelemetry-helm-charts/blob/main/charts/opentelemetry-operator/CONTRIBUTING.md)
- [ ] Verify the [\`community-operators-prod\`](https://github.com/redhat-openshift-ecosystem/community-operators-prod) PR is approved and merged
- [ ] Verify the [\`community-operators\`](https://github.com/k8s-operatorhub/community-operators) PR is approved and merged"

if [ "$DRY_RUN" = true ]; then
  echo ""
  echo "--- DRY RUN: would create issue ---"
  echo "Title: Prepare release v${VERSION}"
  echo "Assignee: ${MANAGER}"
  echo ""
  echo "$BODY"
  exit 0
fi

gh issue create \
  --repo "$REPO" \
  --title "Prepare release v${VERSION}" \
  --assignee "${MANAGER}" \
  --body "$BODY"
