#!/bin/bash

SED_BIN=${SED_BIN:-sed}

${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/smoke-targetallocator/00-install.yaml
${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/targetallocator-features/00-install.yaml
${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/prometheus-config-validation/*.yaml
