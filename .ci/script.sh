#!/bin/bash

if [ -z ${GOPATH} ]; then
    export PATH="${HOME}/go/bin:${PATH}"
else
    export PATH="${GOPATH}/bin:${PATH}"
fi

BUILD_IMAGE="quay.io/jpkroehling/opentelemetry-operator" make ci
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to build the operator."
    exit ${RT}
fi
