#!/bin/bash

# Get the HTTP and GRPC routes from OpenTelemetry receiver collector.
otlp_route_http=$(oc -n kuttl-multi-cluster-receive get route otlp-http-otlp-receiver-route -o json | jq '.spec.host' -r)
otlp_route_grpc=$(oc -n kuttl-multi-cluster-receive get route otlp-grpc-otlp-receiver-route -o json | jq '.spec.host' -r)

# Define the collector content
collector_content=$(cat <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: otel-sender
  namespace: kuttl-multi-cluster-send
spec:
  mode: deployment
  serviceAccount: kuttl-multi-cluster
  volumes:
    - name: kuttl-certs
      configMap: 
        name: kuttl-certs
  volumeMounts:
    - name: kuttl-certs
      mountPath: /certs
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
          http:
    processors:
      batch:
      memory_limiter:
        check_interval: 1s
        limit_percentage: 50
        spike_limit_percentage: 30
    exporters:
      otlphttp:
        endpoint: "https://${otlp_route_http}:443"
        tls:
          insecure: false
          cert_file: /certs/server.crt
          key_file: /certs/server.key
          ca_file: /certs/ca.crt
      otlp:
        endpoint: "${otlp_route_grpc}:443"
        tls:
          insecure: false
          cert_file: /certs/server.crt
          key_file: /certs/server.key
          ca_file: /certs/ca.crt
    service:
      pipelines:
        traces:    
          receivers: [otlp]
          processors: [memory_limiter, batch]
          exporters: [otlphttp, otlp]
EOF
)


# Process the template content and create the objects
echo "$collector_content" | oc -n kuttl-multi-cluster-send create -f -
