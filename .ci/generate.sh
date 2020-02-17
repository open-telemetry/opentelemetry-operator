#!/bin/bash

OPENAPIGEN=openapi-gen
command -v ${OPENAPIGEN} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        OPENAPIGEN="${GOPATH}/bin/openapi-gen"
    fi
fi

CLIENTGEN=client-gen
command -v ${OPENAPIGEN} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        CLIENTGEN="${GOPATH}/bin/client-gen"
    fi
fi

# generate the Kubernetes stubs
operator-sdk generate k8s
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate the Kubernetes stubs."
    exit ${RT}
fi

# generate the CRD(s)
operator-sdk generate crds
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate CRDs."
    exit ${RT}
fi

# generate the schema validation (openapi) stubs
${OPENAPIGEN} --logtostderr=true -o "" -i ./pkg/apis/opentelemetry/v1alpha1 -O zz_generated.openapi -p ./pkg/apis/opentelemetry/v1alpha1 -h /dev/null -r "-"
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate the openapi (schema validation) stubs."
    exit ${RT}
fi

# generate the clients
${CLIENTGEN} \
    --input "opentelemetry/v1alpha1" \
    --input-base github.com/open-telemetry/opentelemetry-operator/pkg/apis \
    --go-header-file /dev/null \
    --output-package github.com/open-telemetry/opentelemetry-operator/pkg/client \
    --clientset-name versioned \
    --output-base ../../../
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate the OpenTelemetry Operator clients."
    exit ${RT}
fi
