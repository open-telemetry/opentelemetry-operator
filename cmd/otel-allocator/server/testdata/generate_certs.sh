#!/usr/bin/env bash

set -euo pipefail

# This script generates both client and server certificates and keys

# get the dir where the script is exec'd from
# so we can move to the 'certs' subdir regardless of where this script is called from
SCRIPT_DIR=$(cd -P -- $(dirname -- $0) && pwd -P)
pushd $SCRIPT_DIR/certs > /dev/null

# Server Key and Cert
openssl genpkey -algorithm Ed25519 -out server.key
openssl req -new -x509 -sha256 -key server.key -out server.crt -days 365 -subj '/CN=localhost' -addext "subjectAltName = DNS:localhost"

echo "Server cert and key created"
echo "==========================="
openssl x509 -noout -text -in server.crt
echo "==========================="

# Client Key and Cert
openssl genpkey -algorithm Ed25519 -out client.key
openssl req -new -key client.key -out client.csr -subj '/CN=<some client UUID>'

# Sign it with the server cert
# IRL you wouldn't do this, the leaf cert for a server would not have the same key as the CA authority
# See https://github.com/joekir/YUBIHSM_mTLS_PKI as an example of that done more thoroughly
echo "00" > file.srl
openssl x509 -req -in client.csr -CA server.crt -CAkey server.key -CAserial file.srl -out client.crt

echo "Client cert and key created"
echo "==========================="
openssl x509 -noout -text -in client.crt
echo "==========================="

popd > /dev/null