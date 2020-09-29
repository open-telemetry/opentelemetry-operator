#!/bin/bash

source ./.ci/tagAndPushFunc.sh

BASE_BUILD_IMAGE=${BASE_BUILD_IMAGE:-"otel/opentelemetry-operator"}
OPERATOR_VERSION=${OPERATOR_VERSION:-$(git describe --tags | grep -Po "([\d\.]+)")}
BUILD_IMAGE=${BUILD_IMAGE:-"${BASE_BUILD_IMAGE}:${OPERATOR_VERSION}"}

echo "Building image ${BUILD_IMAGE}"
make docker-build IMG="${BUILD_IMAGE}" VERSION=${OPERATOR_VERSION}

## push to Docker Hub
if [ "${DOCKER_PASSWORD}x" != "x" -a "${DOCKER_USERNAME}x" != "x" ]; then
    echo "Performing a 'docker login'"
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

    BASE_BUILD_IMAGE=${DOCKER_BASE_BUILD_IMAGE:-"otel/opentelemetry-operator"}
    tagAndPush ${BUILD_IMAGE} ${BASE_BUILD_IMAGE} ${OPERATOR_VERSION}
fi

## push to quay.io
if [ "${QUAY_PASSWORD}x" != "x" -a "${QUAY_USERNAME}x" != "x" ]; then
    echo "Performing a 'docker login' for Quay"
    echo "${QUAY_PASSWORD}" | docker login -u "${QUAY_USERNAME}" quay.io --password-stdin

    BASE_BUILD_IMAGE=${QUAY_BASE_BUILD_IMAGE:-"quay.io/opentelemetry/opentelemetry-operator"}
    tagAndPush ${BUILD_IMAGE} ${BASE_BUILD_IMAGE} ${OPERATOR_VERSION}
fi

