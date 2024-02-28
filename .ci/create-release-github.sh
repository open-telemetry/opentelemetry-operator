#!/bin/bash

NOTES_FILE=/tmp/notes.md
OPERATOR_VERSION=$(git describe --tags --abbrev=0)
PREVIOUS_OPERATOR_VERSION=$(git describe --tags --abbrev=0 "${OPERATOR_VERSION}^")
# Note: Changelog headers don't have the `v` prefix, so we need to drop the first letter in the sed expression below
sed -n "/${OPERATOR_VERSION:1}/,/${PREVIOUS_OPERATOR_VERSION:1}/{/${PREVIOUS_OPERATOR_VERSION:1}/!p;}" CHANGELOG.md >${NOTES_FILE}

gh config set prompt disabled
gh release create \
    -t "Release ${OPERATOR_VERSION}" \
    --notes-file ${NOTES_FILE} \
    "${OPERATOR_VERSION}" \
    'dist/opentelemetry-operator.yaml#Installation manifest for Kubernetes'
