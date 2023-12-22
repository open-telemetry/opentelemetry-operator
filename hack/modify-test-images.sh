#!/bin/bash

SED_BIN=${SED_BIN:-sed}

DEFAULT_OPERATOROPAMPBRIDGE_IMG=${DEFAULT_OPERATOROPAMPBRIDGE_IMG:-local/opentelemetry-operator-opamp-bridge:e2e}
DEFAULT_OPERATOR_IMG=${DEFAULT_OPERATOR_IMG:-local/opentelemetry-operator:e2e}

${SED_BIN} -i "s#${DEFAULT_OPERATOR_IMG}#${OPERATOR_IMG}#g" tests/e2e-multi-instrumentation/*.yaml

${SED_BIN} -i "s#${DEFAULT_OPERATOROPAMPBRIDGE_IMG}#${OPERATOROPAMPBRIDGE_IMG}#g" tests/e2e-opampbridge/opampbridge/*.yaml
