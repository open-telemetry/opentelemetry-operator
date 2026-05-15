#!/bin/bash
set -euo pipefail
oc patch apiserver cluster --type json \
  -p '[{"op":"remove","path":"/spec/tlsSecurityProfile"}]' 2>/dev/null || true
echo "APIServer TLS profile reverted to default (nil)"
