#!/bin/bash

GOLINT=golint

command -v ${GOLINT} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        GOLINT="${GOPATH}/bin/golint"
    fi
fi

out=$(${GOLINT} ./...)
if [[ $out ]]; then
    echo "$out"
    exit 1
fi