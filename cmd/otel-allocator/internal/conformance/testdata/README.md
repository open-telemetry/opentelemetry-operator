# Target allocator conformance fixtures

Each subdirectory is one differential test case:

- `prometheus.yaml` - a **raw Prometheus** config (top-level `scrape_configs:`), consumed by both the target allocator
  and `promtool` (the golden source). Use
  `static_configs`; `__meta_*` labels may be set directly on static targets to exercise relabeling that would normally
  come from Kubernetes service discovery.
- `golden.json` - the committed golden: `promtool check service-discovery`
  output, one entry per discovered target (`discoveredLabels` pre-relabel,
  `labels` post-relabel; empty `labels` means the target was dropped).

## Scope

Fixtures use `static_configs` only. That is deliberate, not a coverage gap:

- **Relabeling is service-discovery-agnostic.** It operates on the discovered label set, so static targets carrying
  synthetic `__meta_*` labels exercise the same relabel/merge/identity logic that `kubernetes_sd`/`file_sd` feed into.
  Target *production* for those mechanisms is upstream Prometheus code the allocator uses unchanged via
  `discovery.Manager`, so the suite hands target groups straight to the merge/filter rather than running discovery
  (whose update debounce also adds ~5s per config). `file_sd` would additionally require reproducing Prometheus's
  `__meta_filepath` exactly.
- **Sharding** - the prometheus-operator `$(SHARD)`/hashmod rules the allocator neutralizes via `addNoShardingConfig` -
  is allocator-specific behavior that intentionally diverges from raw Prometheus, so it does not fit this differential
  model. It is covered by `TestApplyHashmodAction` in `internal/prehook`.

## What is asserted

For every fixture the suite compares the allocator's discovery+relabel result against the golden (see
`../conformance_test.go`):

- **keep/drop parity**: the allocator keeps a target iff Prometheus does.
- **identity grouping**: the allocator's identity-hash partition equals Prometheus's post-relabel-label partition. This
  guards against the recurring target-identity bug class.
- **merge fidelity**: the allocator's served pre-relabel labels equal Prometheus's `discoveredLabels` minus the scrape
  labels it defers to the collector (`job`, `__scheme__`, `__metrics_path__`, `__scrape_interval__`,
  `__scrape_timeout__`).

## Known divergences

Fixtures whose behavior currently differs from raw Prometheus are listed in
`divergentFixtures` in `../conformance_test.go` and skipped with a reason. See `seeded-labels/` for an example (relabel rules on scrape-seeded labels).

## Adding or updating a case

1. Add `cmd/otel-allocator/internal/conformance/testdata/<name>/prometheus.yaml`.
2. Regenerate goldens against raw Prometheus:

   ```
   make ta-conformance-regen      # downloads pinned promtool, runs -update
   ```

   Review the resulting `golden.json` and commit it alongside the fixture.
3. Run the suite (no promtool needed): `go test ./cmd/otel-allocator/internal/conformance/...`
   (also runs as part of `make test`).

