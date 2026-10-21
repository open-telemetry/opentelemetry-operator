#!/bin/bash
# Save current APIServer TLS profile and switch to a different profile.
# Uses Modern (TLS 1.3) on OCP >= 4.14, falls back to Custom (TLS 1.0 + GCM ciphers) on older versions.
set -ex

# Save the current APIServer TLS security profile for later restoration
CURRENT_PROFILE=$(oc get apiserver cluster -o jsonpath='{.spec.tlsSecurityProfile}')
echo "Current APIServer TLS profile: ${CURRENT_PROFILE:-<not set>}"

# Store in a ConfigMap so we can retrieve it later
oc create configmap tls-profile-backup \
  -n "$NAMESPACE" \
  --from-literal=profile="${CURRENT_PROFILE}" \
  --dry-run=client -o yaml | oc apply -f -

# Try Modern profile first (available on OCP >= 4.14)
if oc patch apiserver cluster --type merge --dry-run=server \
  -p '{"spec":{"tlsSecurityProfile":{"type":"Modern","modern":{}}}}' 2>/dev/null; then
  oc patch apiserver cluster --type merge -p '{"spec":{"tlsSecurityProfile":{"type":"Modern","modern":{}}}}'
  # Modern: expect min_version "1.3", no cipher_suites (TLS 1.3 ciphers not configurable in Go),
  # cluster groups injected as curve_preferences (groups remain configurable under TLS 1.3).
  oc create configmap tls-profile-expected -n "$NAMESPACE" \
    --from-literal=min_version="1.3" --from-literal=expect_ciphers="false" \
    --from-literal=expect_curves="true" \
    --from-literal=expected_curves='["X25519MLKEM768","X25519","P256","P384"]' \
    --from-literal=nmap_profile="modern" \
    --dry-run=client -o yaml | oc apply -f -
  echo "APIServer patched to Modern profile"
else
  # Fallback to Custom profile with TLS 1.0 and secure GCM ciphers.
  # Cannot use Old profile: it includes insecure ciphers (CBC, RSA key exchange)
  # that the OTel collector rejects. Cannot use Modern: not supported on OCP < 4.14.
  CUSTOM_PATCH='{"spec":{"tlsSecurityProfile":{"type":"Custom","custom":{"ciphers":["ECDHE-ECDSA-AES128-GCM-SHA256","ECDHE-RSA-AES128-GCM-SHA256","ECDHE-ECDSA-AES256-GCM-SHA384","ECDHE-RSA-AES256-GCM-SHA384","ECDHE-ECDSA-CHACHA20-POLY1305","ECDHE-RSA-CHACHA20-POLY1305"],"minTLSVersion":"VersionTLS10"}}}}'
  oc patch apiserver cluster --type merge -p "$CUSTOM_PATCH"
  # Custom TLS 1.0: expect min_version "1.0", cipher_suites present.
  # The Custom profile below sets no groups, so no curve_preferences are injected.
  oc create configmap tls-profile-expected -n "$NAMESPACE" \
    --from-literal=min_version="1.0" --from-literal=expect_ciphers="true" \
    --from-literal=expect_curves="false" \
    --from-literal=nmap_profile="custom-tls10" \
    --dry-run=client -o yaml | oc apply -f -
  echo "Modern profile not supported; APIServer patched to Custom (TLS 1.0 + GCM ciphers)"
fi
