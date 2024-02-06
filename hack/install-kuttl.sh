#!/bin/bash

if (${KUTTL} version | grep ${KUTTL_VERSION}) > /dev/null 2>&1; then
  exit 0;
fi

OS=$(go env GOOS)
ARCH=$(uname -m)

curl -Lo ${KUTTL} https://github.com/kudobuilder/kuttl/releases/download/v${KUTTL_VERSION}/kubectl-kuttl_${KUTTL_VERSION}_${OS}_${ARCH}
chmod +x ${KUTTL}

