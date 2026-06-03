#!/bin/bash

# Create a directory to store certificates
CERT_DIR="/tmp/chainsaw-certs"
rm -rf "$CERT_DIR"
mkdir -p "$CERT_DIR"

# Get hostname domain from OpenShift
hostname_domain="*.apps.$(oc get dns cluster -o jsonpath='{.spec.baseDomain}')"

# Set certificate information
CERT_SUBJECT="/C=US/ST=California/L=San Francisco/O=My Organization/CN=opentelemetry"

# Create a temporary OpenSSL configuration file for SANs
openssl_config="$CERT_DIR/openssl.cnf"
cat <<EOF > "$openssl_config"
[ req ]
default_bits       = 2048
distinguished_name = req_distinguished_name
req_extensions     = v3_req

[ req_distinguished_name ]
countryName                = Country Name (2 letter code)
countryName_default        = US
stateOrProvinceName        = State or Province Name (full name)
stateOrProvinceName_default= California
localityName               = Locality Name (eg, city)
localityName_default       = San Francisco
organizationName           = Organization Name (eg, company)
organizationName_default   = My Organization
commonName                 = Common Name (eg, your name or your server's hostname)
commonName_max             = 64

[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = opentelemetry
DNS.2 = $hostname_domain
DNS.3 = chainsaw-oidc-server-collector
EOF

# Generate private key for the server
openssl genpkey -algorithm RSA -out "$CERT_DIR/server.key"

# Create CSR for the server with SANs
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" -subj "$CERT_SUBJECT" -config "$openssl_config"

# Generate self-signed certificate for the server with SANs
openssl x509 -req -days 365 -in "$CERT_DIR/server.csr" -signkey "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" -extensions v3_req -extfile "$openssl_config"

# Generate a CA certificate (self-signed)
openssl req -new -x509 -days 365 -key "$CERT_DIR/server.key" -out "$CERT_DIR/ca.crt" -subj "$CERT_SUBJECT"

echo "Certificates generated successfully in $CERT_DIR directory."

# Delete any existing ConfigMaps
kubectl delete configmap -n chainsaw-oidcauthextension chainsaw-certs

# Create a Kubernetes ConfigMap for the server certificate, private key, and CA certificate in chainsaw-multi-cluster-send namespace
kubectl create configmap chainsaw-certs -n chainsaw-oidcauthextension \
  --from-file=server.crt="$CERT_DIR/server.crt" \
  --from-file=server.key="$CERT_DIR/server.key" \
  --from-file=ca.crt="$CERT_DIR/ca.crt"

echo "ConfigMaps created successfully."

