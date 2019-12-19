#!/bin/bash

git diff -s --exit-code
if [[ $? != 0 ]]; then
    echo "The repository isn't clean. We won't proceed, as we don't know if we should commit those changes or not."
    exit 1
fi

BASE_BUILD_IMAGE=${BASE_BUILD_IMAGE:-"quay.io/opentelemetry/opentelemetry-operator"}
BASE_TAG=${BASE_TAG:-$(git describe --tags)}
OPERATOR_VERSION=${OPERATOR_VERSION:-${BASE_TAG}}
OPERATOR_VERSION=$(echo ${OPERATOR_VERSION} | grep -Po "([\d\.]+)")
TAG=${TAG:-"v${OPERATOR_VERSION}"}
BUILD_IMAGE=${BUILD_IMAGE:-"${BASE_BUILD_IMAGE}:${TAG}"}
CREATED_AT=$(date -u -Isecond)
PREVIOUS_VERSION=$(grep operator= opentelemetry.version | awk -F= '{print $2}')

if [[ ${BASE_TAG} =~ ^release/v.[[:digit:].]+(\-.*)?$ ]]; then
    echo "Releasing ${OPERATOR_VERSION} from ${BASE_TAG}"
else
    echo "The release tag does not match the expected format: ${BASE_TAG}"
    exit 1
fi

if [ -z "${GH_WRITE_TOKEN}" ]; then
    echo "The GitHub write token isn't set. Skipping release process."
    exit 1
fi

# changes to deploy/operator.yaml
sed "s~image: quay.io/opentelemetry/opentelemetry-operator.*~image: ${BUILD_IMAGE}~gi" -i deploy/operator.yaml

# change the versions.txt
sed "s~operator=${PREVIOUS_VERSION}~operator=${OPERATOR_VERSION}~gi" -i opentelemetry.version

operator-sdk olm-catalog gen-csv \
    --csv-channel=alpha \
    --default-channel \
    --operator-name opentelemetry-operator \
    --update-crds \
    --csv-version=${OPERATOR_VERSION} \
    --from-version=${PREVIOUS_VERSION}

git diff -s --exit-code
if [[ $? == 0 ]]; then
    echo "No changes detected. Skipping."
else
    git add \
      deploy/olm-catalog/opentelemetry-operator/opentelemetry-operator.package.yaml \
      deploy/operator.yaml \
      opentelemetry.version \
      deploy/olm-catalog/opentelemetry-operator/${OPERATOR_VERSION}/

    git diff -s --exit-code
    if [[ $? != 0 ]]; then
        echo "There are more changes than expected. Skipping the release."
        git diff
        exit 1
    fi

    git config user.email "opentelemetry-operator@opentelemetry.io"
    git config user.name "OpenTelemetry Operator Release"

    git commit -qm "Release ${TAG}"
    git tag ${TAG}
    git push --repo=https://${GH_WRITE_TOKEN}@github.com/open-telemetry/opentelemetry-collector.git --tags
    git push https://${GH_WRITE_TOKEN}@github.com/open-telemetry/opentelemetry-collector.git refs/tags/${TAG}:master
fi
