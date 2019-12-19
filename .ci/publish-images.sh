#!/bin/bash

BASE_BUILD_IMAGE=${BASE_BUILD_IMAGE:-"opentelemetry/opentelemetry-operator"}
OPERATOR_VERSION=${OPERATOR_VERSION:-$(git describe --tags)}

## if we are on a release tag, let's extract the version number
## the other possible value, currently, is 'master' (or another branch name)
## if we are not running in the CI, it fallsback to the `git describe` above
if [[ $OPERATOR_VERSION == v* ]]; then
    OPERATOR_VERSION=$(echo ${OPERATOR_VERSION} | grep -Po "([\d\.]+)")
    MAJOR_MINOR=$(echo ${OPERATOR_VERSION} | awk -F. '{print $1"."$2}')
fi

BUILD_IMAGE=${BUILD_IMAGE:-"${BASE_BUILD_IMAGE}:${OPERATOR_VERSION}"}

echo "Building image ${BUILD_IMAGE}"
make install-tools build container BUILD_IMAGE="${BUILD_IMAGE}"
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to build the operator's container image."
    exit ${RT}
fi

if [ -n "${MAJOR_MINOR}" ]; then
    MAJOR_MINOR_IMAGE="${BASE_BUILD_IMAGE}:${MAJOR_MINOR}"
    docker tag "${BUILD_IMAGE}" "${MAJOR_MINOR_IMAGE}"
    docker push "${MAJOR_MINOR_IMAGE}"
fi

## now, push to quay.io
if [ -n "${QUAY_USERNAME}" -a -n "${QUAY_PASSWORD}" ]; then
    echo "Performing a 'docker login' for Quay"
    echo "${QUAY_PASSWORD}" | docker login -u "${QUAY_USERNAME}" quay.io --password-stdin

    echo "Tagging ${BUILD_IMAGE} as quay.io/${BUILD_IMAGE}"
    docker tag "${BUILD_IMAGE}" "quay.io/${BUILD_IMAGE}"

    echo "Pushing 'quay.io/${BUILD_IMAGE}'"
    docker push "quay.io/${BUILD_IMAGE}"

    if [ -n "${MAJOR_MINOR_IMAGE}" ]; then
        echo "Pushing 'quay.io/${MAJOR_MINOR_IMAGE}' to quay.io"
        docker tag "${MAJOR_MINOR_IMAGE}" "quay.io/${MAJOR_MINOR_IMAGE}"
        docker push "quay.io/${MAJOR_MINOR_IMAGE}"
    fi
else
    echo "Couldn't publish images to ${BUILD_IMAGE}, as the credentials aren't set"
    exit 1
fi
