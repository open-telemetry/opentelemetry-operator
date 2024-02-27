#!/bin/bash

NOTES_FILE=/tmp/notes.md
# Note: Changelog headers don't have the `v` prefix, so we need to drop the first letter in the sed expression below
sed -n "/${DESIRED_VERSION:1}/,/${CURRENT_VERSION:1}/{/${CURRENT_VERSION:1}/!p;}" CHANGELOG.md >${NOTES_FILE}

gh config set prompt disabled
gh release create \
    -t "Release ${DESIRED_VERSION}" \
    --notes-file ${NOTES_FILE} \
    --draft \
    "${DESIRED_VERSION}" \
    'dist/opentelemetry-operator.yaml#Installation manifest for Kubernetes'
