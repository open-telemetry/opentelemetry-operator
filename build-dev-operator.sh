#!/bin/bash

set -e
set -u

make container IMG=quay.io/${QUAY_USERNAME}/splunk-otel-operator:v${OPERATOR_VERSION}
make bundle IMG=quay.io/${QUAY_USERNAME}/splunk-otel-operator:v${OPERATOR_VERSION} VERSION=${OPERATOR_VERSION}
make bundle-build IMG=quay.io/${QUAY_USERNAME}/splunk-otel-operator:v${OPERATOR_VERSION}  VERSION=${OPERATOR_VERSION} BUNDLE_IMG=quay.io/${QUAY_USERNAME}/splunk-otel-operator-bundle:v${BUNDLE_VERSION}

docker push quay.io/${QUAY_USERNAME}/splunk-otel-operator-bundle:v${BUNDLE_VERSION}
docker push quay.io/${QUAY_USERNAME}/splunk-otel-operator:v${OPERATOR_VERSION}
