# Using Prometheus Custom Resources for service discovery

The target allocator can use Custom Resources from the prometheus-operator ecosystem, like ServiceMonitors and PodMonitors, for service discovery, performing
a function analogous to that of prometheus-operator itself. This is enabled via the `prometheusCR` section in the Collector CR.

See below for a minimal example:

```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: collector-with-ta-prometheus-cr
spec:
  mode: statefulset
  targetAllocator:
    enabled: true
    serviceAccount: everything-prometheus-operator-needs
    prometheusCR:
      enabled: true
      serviceMonitorSelector: {}
      podMonitorSelector: {}
      scrapeClasses: []
  config:
    receivers:
      prometheus:
        config: {}

    exporters:
      debug: {}

    service:
      pipelines:
        metrics:
          receivers: [prometheus]
          exporters: [debug]
EOF
```

The `scrapeClasses` attribute refers to the ScrapeClass feature of the Prometheus Operator.
Refer to https://prometheus-operator.dev/docs/developer/scrapeclass/ to learn more about scrape classes.
