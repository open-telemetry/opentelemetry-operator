#!/usr/bin/env bash

# List external container images referenced in e2e test manifests.
# Scans all YAML files under tests/ for hardcoded image: references,
# filtering out locally-built images and template variables.
#
# Usage:
#   hack/list-e2e-images.sh [--test-apps | --external | --all]
#
# Options:
#   --test-apps  List only e2e-test-app images (published to ghcr.io)
#   --external   List only third-party images
#   --all        List both (default)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Extract all image: references from YAML files in tests/
extract_images() {
    grep -rh 'image:' "${REPO_ROOT}/tests/" \
        --include='*.yaml' --include='*.yml' \
    | sed 's/.*image:[[:space:]]*//' \
    | sed 's/[[:space:]]*$//' \
    | sed 's/^"//' \
    | sed 's/"$//' \
    | grep -v '^\$' \
    | grep -v '(\$' \
    | grep -v '{{' \
    | grep -v '^$' \
    | sort -u
}

list_test_apps() {
    extract_images \
    | grep '/e2e-test-app-' \
    | grep -v ':ve2e$'
}

list_external() {
    extract_images \
    | grep -v '/e2e-test-app-' \
    | grep -v "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator"
}

mode="${1:---all}"

case "${mode}" in
    --test-apps)
        list_test_apps
        ;;
    --external)
        list_external
        ;;
    --all)
        list_test_apps
        list_external
        ;;
    *)
        echo "Usage: $0 [--test-apps | --external | --all]" >&2
        exit 1
        ;;
esac
