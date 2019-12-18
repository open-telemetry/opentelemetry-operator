#!/bin/bash

if [ -z "${BUILD_IMAGE}" ]; then
    echo "Build image not provided"
    exit 1
fi

command -v "buildah" > /dev/null
if [ $? = 0 ]; then
    echo "Using buildah" > build/_output/build-container.log
    CMD="buildah bud"
else
    echo "Using Docker" > build/_output/build-container.log
    CMD="docker"
fi

${CMD} -f build/Dockerfile -t ${BUILD_IMAGE} . >> build/_output/build-container.log 2>&1