#!/bin/bash

SED_BIN=${SED_BIN:-sed}

${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/smoke-targetallocator/*.yaml
${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/targetallocator-features/00-install.yaml
${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/prometheus-config-validation/*.yaml
${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/targetallocator-prometheuscr/*.yaml

${SED_BIN} -i "s#local/opentelemetry-operator:e2e#${OPERATOR_IMG}#g" tests/e2e-multi-instrumentation/*.yaml

${SED_BIN} -i "s#local/opentelemetry-operator-opamp-bridge:e2e#${OPERATOROPAMPBRIDGE_IMG}#g" tests/e2e-opampbridge/opampbridge/*.yaml