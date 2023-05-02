# Target Allocator

The TargetAllocator is an optional separately deployed component of an OpenTelemetry Collector setup, which is used to
distribute targets of the PrometheusReceiver on all deployed Collector instances. The release version matches the
operator's most recent release as well.

In essence, Prometheus Receiver configs are overridden with a http_sd_configs directive that points to the
Allocator, these are then loadbalanced/sharded to the collectors. The Prometheus Receiver configs that are overridden
are what will be distributed with the same name. In addition to picking up receiver configs, the TargetAllocator
can discover targets via Prometheus CRs (currently ServiceMonitor, PodMonitor) which it presents as scrape configs
and jobs on the `/scrape_configs` and `/jobs` endpoints respectively.

# Usage
The `spec.targetAllocator:` controls the TargetAllocator general properties. Full API spec can be found here: [api.md#opentelemetrycollectorspectargetallocator](../../docs/api.md#opentelemetrycollectorspectargetallocator)

A basic example that deploys.
```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: collector-with-ta
spec:
  mode: statefulset
  targetAllocator:
    enabled: true
  config: |
    receivers:
      prometheus:
        config:
          scrape_configs:
          - job_name: 'otel-collector'
            scrape_interval: 10s
            static_configs:
            - targets: [ '0.0.0.0:8888' ]

    exporters:
      logging:

    service:
      pipelines:
        metrics:
          receivers: [prometheus]
          processors: []
          exporters: [logging]
```

## PrometheusCR specifics
TargetAllocator discovery of PrometheusCRs can be turned on by setting
`.spec.targetAllocator.prometheusCR.enabled` to `true`

The CRs can be filtered by labels as documented here: [api.md#opentelemetrycollectorspectargetallocatorprometheuscr](../../docs/api.md#opentelemetrycollectorspectargetallocatorprometheuscr)

The prometheus receiver in the deployed collector also has to know where the Allocator service exists. This is done by a
OpenTelemetry Collector operator specific config.
```yaml
  config: |
    receivers:
      prometheus:
        config:
          scrape_configs:
          - job_name: 'otel-collector'
        target_allocator:
          endpoint: http://my-targetallocator-service
          interval: 30s
          collector_id: "${POD_NAME}"
```
Upstream documentation here: [Prometheusreceiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/prometheusreceiver#opentelemetry-operator)

The TargetAllocator service is named based on the OpenTelemetryCollector CR name. `collector_id` should be unique per
collector instance, such as the pod name. The `POD_NAME` environment variable is convenient since this is supplied
to collector instance pods by default.

The Prometheus CRDs also have to exist for the Allocator to pick them up. The best place to get them is from
prometheus-operator: [Releases](https://github.com/prometheus-operator/prometheus-operator/releases). Only the CRDs for
CRs that the Allocator watches for need to be deployed. They can be picked out from the bundle.yaml file.

### RBAC
The ServiceAccount that the TargetAllocator runs as, has to have access to the CRs. A role like this will provide that
access.
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: opentelemetry-targetallocator-cr-role
rules:
- apiGroups:
  - monitoring.coreos.com
  resources:
  - servicemonitors
  - podmonitors
  verbs:
  - '*'
```
In addition, the TargetAllocator needs the same permissions as a Prometheus instance would to find the matching targets
from the CR instances.
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: opentelemetry-targetallocator-role
rules:
- apiGroups: [""]
  resources:
  - nodes
  - nodes/metrics
  - services
  - endpoints
  - pods
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources:
  - configmaps
  verbs: ["get"]
- apiGroups:
  - discovery.k8s.io
  resources:
  - endpointslices
  verbs: ["get", "list", watch"]
- apiGroups:
  - networking.k8s.io
  resources:
  - ingresses
  verbs: ["get", "list", "watch"]
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]
```
These roles can be combined.

A ServiceAccount bound with the above permissions in the namespaces that are to be monitored can then be referenced in
the `targetAllocator:` part of the OpenTelemetryCollector CR.
```yaml
  targetAllocator:
    enabled: true
    serviceAccount: opentelemetry-targetallocator-sa
    prometheusCR:
      enabled: true
```
**Note**: The Collector part of this same CR *also* has a serviceAccount key which only affects the collector and *not*
the TargetAllocator.


# Design

If the Allocator is activated, all Prometheus configurations will be transferred in a separate ConfigMap which get in
turn mounted to the Allocator.    
This configuration will be resolved to target configurations and then split across all OpenTelemetryCollector instances.

TargetAllocators expose the results as [HTTP_SD endpoints](https://prometheus.io/docs/prometheus/latest/http_sd/)
split by collector.

Currently, the Target Allocator handles the sharding of targets. The operator sets the `$SHARD` variable to 0 to allow 
collectors to keep targets generated by a Prometheus CRD. Using Prometheus sharding and target allocator sharding is not
recommended currently and may lead to unknown results.
[See this thread for more information](https://github.com/open-telemetry/opentelemetry-operator/pull/1124#discussion_r984683577)

#### Endpoints
`/scrape_configs`:

```json
{
  "job1": {
    "follow_redirects": true,
    "honor_timestamps": true,
    "job_name": "job1",
    "metric_relabel_configs": [],
    "metrics_path": "/metrics",
    "scheme": "http",
    "scrape_interval": "1m",
    "scrape_timeout": "10s",
    "static_configs": []
  },
  "job2": {
    "follow_redirects": true,
    "honor_timestamps": true,
    "job_name": "job2",
    "metric_relabel_configs": [],
    "metrics_path": "/metrics",
    "relabel_configs": [],
    "scheme": "http",
    "scrape_interval": "1m",
    "scrape_timeout": "10s",
    "kubernetes_sd_configs": []
  }
}
```

`/jobs`:

```json
{
  "job1": {
    "_link": "/jobs/job1/targets"
  },
  "job2": {
    "_link": "/jobs/job1/targets"
  }
}

```

`/jobs/{jobID}/targets`:

```json
{
  "collector-1": {
    "_link": "/jobs/job1/targets?collector_id=collector-1",
    "targets": [
      {
        "Targets": [
          "10.100.100.100",
          "10.100.100.101",
          "10.100.100.102"
        ],
        "Labels": {
          "namespace": "a_namespace",
          "pod": "a_pod"
        }
      }
    ]
  }
}
```

`/jobs/{jobID}/targets?collector_id={collectorID}`:

```json
[
  {
    "targets": [
      "10.100.100.100",
      "10.100.100.101",
      "10.100.100.102"
    ],
    "labels": {
      "namespace": "a_namespace",
      "pod": "a_pod"
    }
  }
]
```


## Packages
### Watchers
Watchers are responsible for the translation of external sources into Prometheus readable scrape configurations and 
triggers updates to the DiscoveryManager

### DiscoveryManager
Watches the Prometheus service discovery for new targets and sets targets to the Allocator 

### Allocator
Shards the received targets based on the discovered Collector instances

### Collector
Client to watch for deployed Collector instances which will then provided to the Allocator. 

