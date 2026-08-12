# Collector metrics e2e

End-to-end tests for how the target allocator labels targets. Each test deploys an
`OpenTelemetryCollector` CR (target allocator + prometheus receiver) through the
operator, scrapes a sample app, exports OTLP to a prometheus-operator-managed
Prometheus, and asserts on the resulting target labels over PromQL:

```
sample-app ──scrape──▶ collector (TA + prometheus receiver) ──OTLP──▶ Prometheus ◀──PromQL── test
```

The tests are plain Go tests built on the helpers in
[`internal/testing/e2e`](../../internal/testing/e2e).

## What it validates

The focus is target labeling, end to end — a target discovered and relabeled by the
allocator must end up with the labels Prometheus would give it — not metric values or
sample structure. The two tests cover the allocator's two discovery paths:

| Test | Path | Key assertion |
|---|---|---|
| `TestServiceMonitorDifferential` | `prometheusCR` (ServiceMonitor) | live differential vs prometheus-operator (see below): the allocator must label a ServiceMonitor target *identically* to prometheus-operator scraping it natively. |
| `TestRawScrapeConfigMetrics` | raw `scrape_configs` (`static_configs`) | the static target carries *exactly* `job`/`instance` and no service-discovery labels. |

### The ServiceMonitor differential

A single prometheus-operator-managed `Prometheus` runs with the OTLP receiver enabled
and `translation_strategy: NoTranslation`, so OTLP target labels are stored
byte-for-byte. The same instance plays both roles:

```
                      ┌──────────── ServiceMonitor (sample app) ────────────┐
   sm-ta  ──▶ collector (TA) ──scrape──▶ ──OTLP──▶  Prometheus  ◀──scrape── sm-oracle
                                                    (one instance)
            series get pipeline="ta"                series get pipeline="oracle"
```

Both pipelines scrape the same pod and write to the same Prometheus, distinguished
only by a `pipeline` label added by an otherwise identical ServiceMonitor. The test
asserts that, after dropping `pipeline`, both partitions carry the same target label
set. Any divergence — the allocator dropping, adding or rewriting a target label
relative to prometheus-operator — fails the differential. prometheus-operator is the
oracle: this directly tests the allocator as a prometheus-operator compatibility
layer.

## Running

```bash
make prepare-e2e            # build the operator and deploy it into kind
make e2e-collector-metrics  # run this suite
```

The suite installs prometheus-operator itself (idempotently, pinned to the version in
`go.mod`) so it can use a `Prometheus` CR as the oracle. That bundle and the test
images (`quay.io/prometheus/prometheus`, `quay.io/prometheus/node-exporter`) are
fetched from the internet.

## Notes

- The operator does not auto-create the target allocator's RBAC — the permissions
  depend on what the allocator is asked to discover — so each test binds the project's
  shipped ClusterRole (`config/target-allocator/clusterrole.yaml`) to the allocator's
  (and the oracle's) ServiceAccount explicitly. Without it, the allocator discovers
  nothing and nothing is scraped.
- The assertions use `up` and the sample app's always-present
  `node_exporter_build_info` gauge, so the tests need no traffic against the app.
- `tsdb.outOfOrderTimeWindow` on the oracle Prometheus absorbs the reordering of OTLP
  export batches.
