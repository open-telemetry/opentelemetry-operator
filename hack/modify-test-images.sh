#!/bin/bash

sed -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/smoke-targetallocator/00-install.yaml
sed -i "s#local/opentelemetry-operator-targetallocator:e2e#${TARGETALLOCATOR_IMG}#g" tests/e2e/targetallocator-features/00-install.yaml
