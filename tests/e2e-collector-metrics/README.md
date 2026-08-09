# Go full-deployment e2e

Full-deployment e2e tests written in Go for cases that need **semantic checks** —
"the target got discovered and labeled correctly, end to end" — which are painful in
chainsaw/bash + `jq`.

Each test deploys through the **operator** (an `OpenTelemetryCollector` CR with the
target allocator + prometheus receiver), scrapes a sample app, exports OTLP to a
prometheus-operator-managed **Prometheus**, and asserts on the resulting target
labels over **PromQL** — all in typed Go.

```
sample-app ──scrape──▶ collector (TA + prometheus receiver) ──OTLP──▶ Prometheus ◀──PromQL── test
```

## What it validates

The focus is **target labeling, end to end** — that a target discovered and
relabeled by the allocator ends up with the labels Prometheus would give it — not
metric values or sample structure. Two suites cover the allocator's two discovery
paths:

| Suite | Path | Key assertion |
|---|---|---|
| `TestServiceMonitorDifferential` | `prometheusCR` (ServiceMonitor) | **live differential vs prometheus-operator** (see below) — the allocator must label a ServiceMonitor target *identically* to prometheus-operator scraping it natively. |
| `TestRawScrapeConfigMetrics` | raw `scrape_configs` (`static_configs`) | the static target carries *exactly* `job`/`instance` and no service-discovery labels (`Exact`). |

### The ServiceMonitor differential

A single prometheus-operator-managed `Prometheus` runs with the OTLP receiver enabled
and `translation_strategy: NoTranslation`. The same instance plays both roles:

```
                      ┌──────────── ServiceMonitor (sample app) ────────────┐
   sm-ta  ──▶ collector (TA) ──scrape──▶ ──OTLP──▶  Prometheus  ◀──scrape── sm-oracle
                                                    (one instance)
            series get pipeline="ta"                series get pipeline="oracle"
```

Both pipelines scrape the **same pod** and write to the **same Prometheus**,
distinguished only by a `pipeline` label (`ta` vs `oracle`) added by an otherwise
identical ServiceMonitor. `prom.SameLabelsAcross(ctx, t, e2e.Differential{Query: "up",
PartitionLabel: "pipeline", WantPartitions: 2})` then asserts that, after dropping
`pipeline`, the two partitions carry the **same target label
set**. Because `NoTranslation` makes the OTLP round-trip label-preserving, any
divergence — the allocator dropping, adding or rewriting a target label relative to
prometheus-operator — fails the differential. prometheus-operator is the oracle, so
this directly tests the allocator as a prometheus-operator compatibility layer.

## Design

Same module as the operator; gated by `//go:build e2e`; run as a normal Go test
against a kind cluster — no chainsaw. The reusable framework lives in
[`internal/testing/e2e`](../../internal/testing/e2e) and uses only libraries the
operator already depends on, so the e2e tests add **nothing** to its module footprint:

| Concern | Library | Used for |
|---|---|---|
| Lifecycle | `sigs.k8s.io/e2e-framework` | `env`/`features`, typed resource waits |
| Cluster ops | controller-runtime + client-go | server-side-apply manifests (unstructured), namespaces, RBAC |
| PromQL | API server **service proxy** + `common/model` | query Prometheus with no port-forward; typed results |

Framework helpers, imported under the package's own name `e2e`:

- `e2e.Apply(ctx, t, cfg, ns, manifests)` — server-side-applies multi-doc YAML as
  unstructured objects (no scheme registration needed for the CRDs).
- `e2e.ApplyObjects(ctx, t, cfg, ns, objs...)` — server-side-applies **typed** objects,
  for the resources a test needs to vary (here: the two ServiceMonitors).
- `e2e.SetupNamespace(ctx, t, cfg)` — creates a namespace named after the test,
  registers its deletion as cleanup, and returns a context carrying its name;
  `e2e.Namespace(t, ctx)` reads it back inside the assessments.
- `e2e.CreateNamespace` / `e2e.DeleteNamespace` / `e2e.WaitForStatefulSet` / `e2e.WaitForDeployment`.
- `e2e.EnsurePrometheusOperator(ctx, t, cfg)` — idempotently installs prometheus-operator
  (version read from `go.mod`) so a `Prometheus` CR can be used as a live oracle.
- `e2e.BindTargetAllocatorClusterRole(ctx, t, cfg, ns, sa)` — apply the project's
  shipped ClusterRole (`config/target-allocator/clusterrole.yaml`) and bind it to the
  named ServiceAccount (the allocator's or the oracle Prometheus's).
- `e2e.NewProm(cfg, ns, service, port)` — a handle to a Prometheus reachable over the
  API server's service proxy (no port-forward). It carries the namespace/Service/port,
  so assertions only name the query:
  - `prom.Query(ctx, t, query)` — one instant query, no retry.
  - `prom.Eventually(ctx, t, query, check)` — retry the query until
    `check(model.Vector) error` passes (`require.EventuallyWithT` under the hood).
  - `prom.SameLabelsAcross(ctx, t, e2e.Differential{Query, PartitionLabel, WantPartitions, Ignore})`
    — the **differential**: partition the result by `PartitionLabel` and assert every
    partition carries the same target label set (dropping `PartitionLabel` + `Ignore`),
    retrying until `WantPartitions` are present and agree.
- `e2e.HasSeries(e2e.Series{Labels, Present, Absent, Exact, Value})` — a `check` for a
  single series: `Labels` (exact value), `Present` (any value), `Absent` (must not
  exist), `Exact` (carry *only* `Labels`∪`Present`, nothing else — pins a complete
  target identity while allowing a dynamic pod/instance), and an optional `Value`
  predicate (`e2e.Equals` / `e2e.AtLeast`).

Manifests live in `testdata/` and are `go:embed`-ed. They are static: every test runs
in its own namespace, so object names can be fixed, and the one object that genuinely
differs between the two pipelines (the `ServiceMonitor`) is built as a typed Go struct
and applied with `ApplyObjects`. A test (`metrics_test.go`) is then just a
`features.Feature`: deploy sample app + oracle Prometheus + collector + RBAC → wait →
assert with `SameLabelsAcross` (differential) and/or `Eventually` + `HasSeries`
(exact identity).

## Running

```bash
make prepare-e2e            # deploys the operator into kind (shared with the chainsaw e2e)
make e2e-collector-metrics  # runs this Go suite
```

The suite installs prometheus-operator itself (idempotently, via
`e2e.EnsurePrometheusOperator`) on first run, so no extra prep step is needed — but it
fetches the release bundle over the network.

## Notes

- **Oracle backend:** a single prometheus-operator-managed `Prometheus` per test,
  with `enableOTLPReceiver: true` and `otlp.translationStrategy: NoTranslation` so OTLP
  target labels are stored byte-for-byte; `tsdb.outOfOrderTimeWindow` absorbs the
  reordering of OTLP export batches. Its ServiceAccount reuses the shipped
  target-allocator ClusterRole for discovery.
- **prometheus-operator install** needs network access to fetch the
  `${PrometheusOperatorVersion}/bundle.yaml`; it is applied server-side (the CRDs are
  too large for client-side apply) and left in place (idempotent across runs).
- **Target allocator RBAC:** the operator does **not** auto-create the allocator's
  RBAC — the permissions a target allocator needs depend on what the user asks it to
  discover (pods, endpoints, ServiceMonitors, …). So the test grants it explicitly,
  reusing the project's shipped ClusterRole (`config/target-allocator/clusterrole.yaml`,
  which covers both core discovery and the Prometheus CRDs). Without it the allocator
  can't list collector pods or ServiceMonitors, allocates no targets, nothing scraped.
- **No traffic needed:** the assertions use `up` and the always-present `version`
  gauge, so the tests need no load against the sample app.
- Images (prometheus-operator, `quay.io/prometheus/prometheus` — deployed by
  prometheus-operator from the `Prometheus` CR — and
  `quay.io/brancz/prometheus-example-app`) are pulled from the internet; load them into
  kind for hermetic/offline runs.
- The framework is backend-agnostic; `Prom.Eventually` is one query helper —
  add others (range queries, alerts) as suites need them.
