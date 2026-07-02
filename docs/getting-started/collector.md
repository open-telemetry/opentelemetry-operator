# Creating an OpenTelemetry Collector

Once the `opentelemetry-operator` deployment is ready, create an OpenTelemetry Collector (otelcol) instance, like:

```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: simplest
spec:
  config:
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    processors:
      memory_limiter:
        check_interval: 1s
        limit_percentage: 75
        spike_limit_percentage: 15

    exporters:
      debug: {}

    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [memory_limiter]
          exporters: [debug]
EOF
```

**_WARNING:_** Until the OpenTelemetry Collector format is stable, changes may be required in the above example to remain
compatible with the latest version of the OpenTelemetry Collector image being referenced.

This will create an OpenTelemetry Collector instance named `simplest`, exposing a `jaeger-grpc` port to consume spans from your instrumented applications and exporting those spans via `debug`, which writes the spans to the console (`stdout`) of the OpenTelemetry Collector instance that receives the span.

The `config` node holds the `YAML` that should be passed down as-is to the underlying OpenTelemetry Collector instances. Refer to the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) documentation for a reference of the possible entries.

> 🚨 **NOTE:** At this point, the Operator does _not_ validate the whole contents of the configuration file: if the configuration is invalid, the instance might still be created but the underlying OpenTelemetry Collector might crash.

> 🚨 **Note:** For private GKE clusters, you will need to either add a firewall rule that allows master nodes access to port `9443/tcp` on worker nodes, or change the existing rule that allows access to port `80/tcp`, `443/tcp` and `10254/tcp` to also allow access to port `9443/tcp`. More information can be found in the [Official GCP Documentation](https://cloud.google.com/load-balancing/docs/tcp/setting-up-tcp#config-hc-firewall). See the [GKE documentation](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#add_firewall_rules) on adding rules and the [Kubernetes issue](https://github.com/kubernetes/kubernetes/issues/79739) for more detail.

The Operator does examine the configuration file for a few purposes:

- To discover configured receivers and their ports. If it finds receivers with ports, it creates a pair of kubernetes services, one headless, exposing those ports within the cluster. If the port is using environment variable expansion or cannot be parsed, an error will be returned. The headless service contains a `service.beta.openshift.io/serving-cert-secret-name` annotation that will cause OpenShift to create a secret containing a certificate and key. This secret can be mounted as a volume and the certificate and key used in those receivers' TLS configurations.

- To check if Collector observability is enabled (controlled by `spec.observability.metrics.enableMetrics`). In this case, a Service and ServiceMonitor/PodMonitor are created for the Collector instance. As a consequence, if the metrics service address contains an invalid port or uses environment variable expansion for the port, an error will be returned. A workaround for the environment variable case is to set `enableMetrics` to `false` and manually create the previously mentioned objects with the correct port if you need them.
