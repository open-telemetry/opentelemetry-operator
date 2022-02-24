#!/bin/bash

OPERATOR_VERSION=$(git describe --tags)

gh config set prompt disabled
gh release create \
    -t "Release ${OPERATOR_VERSION}" \
    "${OPERATOR_VERSION}" \
    'dist/opentelemetry-operator.yaml#Installation manifest for Kubernetes'
