#!/bin/bash

GOFMT=goimports

command -v ${GOFMT} > /dev/null
if [ $? != 0 ]; then
    if [ -n "${GOPATH}" ]; then
        GOFMT="${GOPATH}/bin/goimports"
    fi
fi

${GOFMT} -local "github.com/open-telemetry/opentelemetry-operator" -l -w $(git ls-files "*\.go" | grep -v vendor)
