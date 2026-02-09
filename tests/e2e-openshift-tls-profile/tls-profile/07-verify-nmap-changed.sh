#!/bin/bash
# Functional TLS verification via nmap ssl-enum-ciphers - changed profile.
# Reads expected profile type from the tls-profile-expected ConfigMap.
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

NMAP_PROFILE=$(oc get configmap tls-profile-expected -n "$NAMESPACE" -o jsonpath='{.data.nmap_profile}')
echo "Verifying nmap profile: $NMAP_PROFILE"

# Verify TLS profile via nmap ssl-enum-ciphers on a specific IP:port.
# Args: $1=ip, $2=ports (comma-separated), $3=expected_profile, $4=description
verify_nmap_tls_profile() {
  local ip="$1" ports="$2" expected="$3" description="$4"

  echo "=== nmap ssl-enum-ciphers: $description ($ip ports $ports) ==="
  local result
  result=$(kubectl exec tls-scanner -n $NAMESPACE -- nmap -Pn --script ssl-enum-ciphers -p "$ports" "$ip")
  echo "$result"

  for port in ${ports//,/ }; do
    local port_section
    port_section=$(echo "$result" | grep -A 50 "${port}/tcp" || true)
    if [ -z "$port_section" ]; then
      fail "$description: port $port not found in nmap output"
    fi

    if [ "$expected" = "modern" ]; then
      echo "$port_section" | grep "TLSv1.3" || fail "$description: port $port missing TLSv1.3"
      if echo "$port_section" | head -30 | grep -q "TLSv1.2"; then
        fail "$description: port $port still accepting TLSv1.2 under Modern profile"
      fi
    elif [ "$expected" = "custom-tls10" ]; then
      # Custom TLS 1.0 with GCM ciphers: collector should still negotiate TLS 1.2+
      # because the GCM ciphers require TLS 1.2. Just verify TLS is working.
      echo "$port_section" | grep -E "TLSv1\.[23]" \
        || fail "$description: port $port not offering TLS 1.2 or 1.3"
    fi
  done

  echo "PASS: $description (ports $ports, profile=$expected)"
}

# Ensure tls-scanner pod is running (may have been evicted during profile change)
if ! kubectl get pod tls-scanner -n $NAMESPACE -o jsonpath='{.status.phase}' 2>/dev/null | grep -q Running; then
  echo "Recreating tls-scanner pod..."
  kubectl delete pod tls-scanner -n $NAMESPACE --force --grace-period=0 2>/dev/null || true
  sleep 5
  kubectl apply -f 00-install-tls-scanner.yaml -n $NAMESPACE
  kubectl wait --for=condition=Ready pod/tls-scanner -n $NAMESPACE --timeout=2m
fi

# Get collector service cluster IP for nmap scan (more reliable than pod IP across nodes)
COLLECTOR_IP=$(kubectl get service tls-profile-test-collector -n $NAMESPACE \
  -o jsonpath='{.spec.clusterIP}')

# Functional TLS check
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tls-profile-test-collector -port 4317 \
  || fail "TLS check failed on collector:4317"
echo "PASS: collector:4317 TLS functional"

# nmap verification
verify_nmap_tls_profile "$COLLECTOR_IP" "4317" "$NMAP_PROFILE" "Collector gRPC"

echo "PASS: Changed profile verified via nmap ($NMAP_PROFILE)"
