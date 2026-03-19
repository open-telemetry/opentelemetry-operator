#!/bin/bash
# Functional TLS verification via nmap ssl-enum-ciphers - Intermediate profile.
# Verifies the collector's gRPC endpoint offers TLSv1.2 and TLSv1.3.
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

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

    if [ "$expected" = "intermediate" ]; then
      echo "$port_section" | grep "TLSv1.2" || fail "$description: port $port missing TLSv1.2"
      echo "$port_section" | grep "TLSv1.3" || fail "$description: port $port missing TLSv1.3"
    elif [ "$expected" = "modern" ]; then
      echo "$port_section" | grep "TLSv1.3" || fail "$description: port $port missing TLSv1.3"
      if echo "$port_section" | head -30 | grep -q "TLSv1.2"; then
        fail "$description: port $port still accepting TLSv1.2 under Modern profile"
      fi
    fi
  done

  echo "PASS: $description (ports $ports, profile=$expected)"
}

# Get collector service cluster IP for nmap scan (more reliable than pod IP across nodes)
COLLECTOR_IP=$(kubectl get service tls-profile-test-collector -n $NAMESPACE \
  -o jsonpath='{.spec.clusterIP}')

# Functional TLS check via tls-scanner
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tls-profile-test-collector -port 4317 \
  || fail "TLS check failed on collector:4317"
echo "PASS: collector:4317 TLS functional"

# nmap ssl-enum-ciphers verification
verify_nmap_tls_profile "$COLLECTOR_IP" "4317" intermediate "Collector gRPC"

echo "PASS: Intermediate profile verified via nmap"
