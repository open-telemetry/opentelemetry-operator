#!/bin/bash

OPERATOR_VERSION=${OPERATOR_VERSION:-$(git describe --tags | grep -Po "([\d\.]+)")}

echo "Building bundle image"
make bundle-build

## push to Docker Hub
if [ "${DOCKER_PASSWORD}x" != "x" -a "${DOCKER_USERNAME}x" != "x" ]; then
    echo "Performing a 'docker login'"
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

    TARGET_IMAGE="otel/opentelemetry-operator-bundle:${OPERATOR_VERSION}"
    docker tag "controller-bundle:${OPERATOR_VERSION}" "${TARGET_IMAGE}"

    echo "Pushing ${TARGET_IMAGE}"
    docker push "${TARGET_IMAGE}"
fi

## push to quay.io
if [ "${QUAY_PASSWORD}x" != "x" -a "${QUAY_USERNAME}x" != "x" ]; then
    echo "Performing a 'docker login' for Quay"
    echo "${QUAY_PASSWORD}" | docker login -u "${QUAY_USERNAME}" quay.io --password-stdin

    TARGET_IMAGE="quay.io/opentelemetry/opentelemetry-operator-bundle:${OPERATOR_VERSION}"
    docker tag "controller-bundle:${OPERATOR_VERSION}" "${TARGET_IMAGE}"

    echo "Pushing ${TARGET_IMAGE}"
    docker push "${TARGET_IMAGE}"
fi

