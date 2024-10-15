#!/usr/bin/env bash

set -ex

# CA key and cert
openssl req -new -nodes -x509 -days 9650 -keyout ca.key -out ca.crt -subj "/C=US/ST=California/L=Mountain View/O=Your Organization/OU=Your Unit/CN=localhost"
# Server, E.g. use NDS:*.default.svc.cluster.local for arbitrary collector name deployed in the default namespace
openssl req -new -nodes -x509 -CA ca.crt -CAkey ca.key -days 9650 -set_serial 01 -keyout server.key -out server.crt -subj "/C=US/ST=California/L=Mountain View/O=Your Organization/OU=Your Unit/CN=svc.cluster.local/CN=localhost"  -addext "subjectAltName = DNS:simplest-collector,DNS:localhost"
# Client
openssl req -new -nodes -x509 -CA ca.crt -CAkey ca.key -days 9650 -set_serial 01 -keyout client.key -out client.crt -subj "/C=US/ST=California/L=Mountain View/O=Your Organization/OU=Your Unit/CN=svc.cluster.local/CN=localhost"

kubectl create configmap ca --from-file=ca.crt=ca.crt  -o yaml --dry-run=client > ca.yaml
kubectl create secret tls server-certs --cert=server.crt --key=server.key -o yaml --dry-run=client > server-secret.yaml
kubectl create secret tls client-certs --cert=client.crt --key=client.key -o yaml --dry-run=client > client-secret.yaml
