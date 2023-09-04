#!/bin/bash

SED_BIN=${SED_BIN:-sed}

# Target allocator image
${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/smoke-targetallocator/*.yaml
${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/targetallocator-features/00-install.yaml
${SED_BIN} -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/prometheus-config-validation/*.yaml

# Instrumentation 
${SED_BIN} -i "s#local/apachehttpd-test:e2e#${APACHE_E2E}#g" tests/e2e-instrumentation/instrumentation-apache-httpd/01-install-app.yaml
