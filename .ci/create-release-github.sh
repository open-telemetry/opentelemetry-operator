#!/bin/bash

NOTES_FILE=/tmp/notes.md
# Note: We expect the versions to not have the `v` prefix here
sed -n "/${DESIRED_VERSION}/,/${CURRENT_VERSION}/{/${CURRENT_VERSION}/!p;}" CHANGELOG.md >${NOTES_FILE}

gh config set prompt disabled
gh release create \
    -t "Release v${DESIRED_VERSION}" \
    --notes-file ${NOTES_FILE} \
    --draft \
    "v${DESIRED_VERSION}" \
    'dist/opentelemetry-operator.yaml#Installation manifest for Kubernetes' \
    'dist/opentelemetry-operator-openshift.yaml#Installation manifest for OpenShift'
