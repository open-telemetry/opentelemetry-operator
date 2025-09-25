# E2E Metrics Consistency Test

## Goals

- Verify Target Allocator (TA) deduplication and target distribution
- Verify consistency between direct scraping and TA-distributed scraping

## Topology

```text
OTel #1 (direct-scrape)    → scrapes directly
Target Allocator           → discovers & deduplicates targets, distributes to OTel #2/#3
OTel #2/#3 (ta-distributed)→ receive sharded targets from TA
```

## Steps

- Step 0: Deploy RBAC, blackbox-exporter, TA + distributed collectors (2 replicas)
- Step 1: Deploy direct-scrape (1 replica)
- Step 2/3: Run the metrics consistency verification job (Go)

## Key Details

- Static scrape_configs + Kubernetes SD (blackbox-exporter)
- PrometheusCR disabled by default; enable with selectors if needed
- Per-pod services for the distributed collectors:
  - `ta-distributed-0-collector:8889`
  - `ta-distributed-1-collector:8889`
- Direct-scrape service: `direct-scrape-collector:8889`
- TA API base URL: `http://ta-distributed-targetallocator`

## Run

```bash
cd tests/e2e-metrics-consistency
chainsaw test --test-dir .
```

## Assertions & Expected Results

- `ta-distributed-targetallocator` (Deployment) readyReplicas = 1
- `ta-distributed-collector` (StatefulSet) readyReplicas = 2
- `direct-scrape-collector` (StatefulSet) readyReplicas = 1
- Verification job completes successfully; logs include:
  - Deduplication effective
  - Targets distributed to multiple collectors
  - Direct scrape and distributed scrape results are consistent
