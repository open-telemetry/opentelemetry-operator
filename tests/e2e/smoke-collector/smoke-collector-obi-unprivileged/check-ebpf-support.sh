#!/bin/sh
# Returns 0 if unprivileged eBPF tracing is expected to work, 1 if not.
# On non-OpenShift (kind), eBPF always works with the base-6 capabilities.
# On OpenShift, kernel >= 6.4 is required for the sk_msg/sockhash path
# that avoids SYS_ADMIN. Older kernels (5.14 on RHEL 9 / OCP 4.x) cannot
# use unprivileged eBPF tracing.
if ! kubectl api-resources --api-group=security.openshift.io -o name 2>/dev/null | grep -q securitycontextconstraints; then
  exit 0
fi
KERNEL=$(kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.kernelVersion}')
MAJOR=$(echo "$KERNEL" | cut -d. -f1)
MINOR=$(echo "$KERNEL" | cut -d. -f2)
if [ "$MAJOR" -gt 6 ] || { [ "$MAJOR" -eq 6 ] && [ "$MINOR" -ge 4 ]; }; then
  exit 0
fi
echo "SKIP: OpenShift kernel $KERNEL < 6.4 — unprivileged eBPF tracing not supported"
exit 1
