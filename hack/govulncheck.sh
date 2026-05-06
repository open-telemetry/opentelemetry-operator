#!/usr/bin/env bash
# Run govulncheck and filter out CVEs that are explicitly excepted.
#
# govulncheck always exits 0 with -format json, leaving pass/fail to the
# consumer. We mirror text-mode behavior: fail on findings with a function
# frame in the trace (i.e. reachable in the call graph). We also fail if any
# excepted CVE is no longer detected, so the exception list stays current.
set -euo pipefail

# Excepted CVEs. See https://github.com/open-telemetry/opentelemetry-operator/issues/4926
EXCEPTED_CVES='["CVE-2026-34040","CVE-2026-33997"]'

GOVULNCHECK="${GOVULNCHECK:-govulncheck}"

vuln_json=$(mktemp)
trap 'rm -f "$vuln_json"' EXIT

"$GOVULNCHECK" -format json ./... > "$vuln_json"

called_osv=$(jq -r '
  .finding? | select(. != null) |
  select(.trace | any(has("function"))) |
  .osv
' "$vuln_json" | sort -u)

called_osv_json=$(printf '%s\n' "$called_osv" | jq -R . | jq -sc 'map(select(. != ""))')
called_cves=$(jq -r --argjson ids "$called_osv_json" '
  .osv? | select(. != null) |
  select(.id as $id | $ids | index($id)) |
  (.aliases // [])[]
' "$vuln_json" | sort -u)

called_cves_json=$(printf '%s\n' "$called_cves" | jq -R . | jq -sc 'map(select(. != ""))')
stale=$(jq -nr --argjson ex "$EXCEPTED_CVES" --argjson called "$called_cves_json" '
  ($ex - $called)[]
')

if [ -n "$stale" ]; then
  echo "Stale exceptions (CVE no longer detected as called by govulncheck):" >&2
  echo "$stale" >&2
  echo "Remove these from EXCEPTED_CVES in hack/govulncheck.sh." >&2
  exit 1
fi

if [ -z "$called_osv" ]; then
  echo "No called vulnerabilities found."
  exit 0
fi

excepted_osv=$(jq -r --argjson ex "$EXCEPTED_CVES" '
  .osv? | select(. != null) |
  select(((.aliases // []) - $ex) != (.aliases // [])) |
  .id
' "$vuln_json" | sort -u)

unexpected=$(comm -23 <(echo "$called_osv") <(echo "$excepted_osv"))

if [ -n "$unexpected" ]; then
  echo "Unexpected vulnerabilities found (not in exception list):" >&2
  echo "$unexpected" >&2
  exit 1
fi

echo "Only excepted vulnerabilities found:"
echo "$called_osv"
