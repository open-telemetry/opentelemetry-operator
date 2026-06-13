# Documentation restructure

**Status:** *Draft*

**Author:** Mikołaj Świątek (@swiatekm)

**Date:** 2026-06-13


## Objective

Reorganize the operator's user documentation into a structured `docs/` tree, fill known content gaps, and add small autogeneration for reference material that's currently hand-maintained.

The current state has four problems:

- The root [`README.md`](../../README.md) is ~970 lines, with most user-facing documentation living several headings deep inside it.
- `docs/` is a small set of standalone files with no coherent organization ([`compatibility.md`](../compatibility.md), [`cluster-observability.md`](../cluster-observability.md), [`crd-changelog.md`](../crd-changelog.md), [`api/`](../api/), `rfcs/`).
- Target Allocator user documentation lives at [`cmd/otel-allocator/README.md`](../../cmd/otel-allocator/README.md), next to its code, where users don't look.
- Substantial topics are simply missing: RBAC of the operator and of the components in the Kubernetes collector distribution, TLS between operator/collectors/Target Allocator, end-to-end guides for collecting logs/traces/Prometheus metrics, tail sampling, and reference deployment architectures.


## Summary

Move existing user content into a structured `docs/` tree organized around the operator's CRDs, supplemented with cross-cutting categories (getting-started, concepts, troubleshooting, reference). Slim the root [`README.md`](../../README.md) to a navigational role. In subsequent phases, author the content that's currently missing and add a small `make docs-gen` target that emits reference tables for CLI flags and feature gates. Stay on plain GitHub-rendered Markdown.

A proof-of-concept of Phase 1 exists at <https://github.com/swiatekm/opentelemetry-operator/pull/128>; the resulting tree is reproduced below.


## Goals and non-goals

**Goals:**

- Reorganize existing user-facing content into the new structure without rewriting prose.
- Fill the known content gaps: RBAC, TLS, log/trace/metric collection, tail sampling, reference architectures
- Add small autogeneration for CLI flag tables ([`main.go`](../../main.go), [`cmd/otel-allocator/main.go`](../../cmd/otel-allocator/main.go)) and the feature-gate table ([`pkg/featuregate/featuregate.go`](../../pkg/featuregate/featuregate.go)), following the pattern of the existing `make api-docs` target in the [`Makefile`](../../Makefile).
- Slim the root README to ~150 lines of navigational content.

**Non-goals:**

- Site generators (MkDocs, Hugo, etc.).
- Reconciling with the operator section on [opentelemetry.io](https://opentelemetry.io/docs/platforms/kubernetes/operator/).
- Changes to [`CONTRIBUTING.md`](../../CONTRIBUTING.md), [`RELEASE.md`](../../RELEASE.md) (except stale-link fixes), [`CHANGELOG.md`](../../CHANGELOG.md), or [`docs/rfcs/`](.).


## Proposed structure

The structure is based on a survey of established Kubernetes-operator documentation: [prometheus-operator](https://github.com/prometheus-operator/prometheus-operator/tree/main/Documentation), [cert-manager](https://cert-manager.io/docs/), [Strimzi](https://github.com/strimzi/strimzi-kafka-operator/tree/main/documentation), and [CloudNativePG](https://github.com/cloudnative-pg/cloudnative-pg/tree/main/docs). They differ in cosmetic detail but converge on the same skeleton - roughly [Diátaxis](https://diataxis.fr/)-shaped, with a domain-specific twist:

- **Getting started** and **concepts** as separate top-level sections.
- **One folder per major CRD or feature area.** Strimzi organizes around Kafka/KafkaConnect/KafkaUser; cert-manager around Issuer/Certificate/usage. This scales better than a single "Usage" bucket when the operator owns multiple substantial CRDs - which is the OTel operator's situation (`OpenTelemetryCollector`, `Instrumentation`, `TargetAllocator`, `OpAMPBridge`, plus the newer `ClusterObservability` controller).
- **Reference**, **troubleshooting**, and **contributing/RFCs** kept as their own top-level categories.
- **Security / RBAC** as a dedicated section in every mature operator we surveyed.

Eventual tree:

```
docs/
├── README.md                  table of contents
├── getting-started/           installation, upgrading, compatibility
├── concepts/                  architecture, CRDs, the K8s collector distro
├── collector/                 OpenTelemetryCollector CRD
├── auto-instrumentation/      Instrumentation CRD
│   └── languages/
├── target-allocator/          TargetAllocator CRD
├── opamp-bridge/              OpAMPBridge CRD
├── use-cases/                 task-oriented how-tos               (Phase 2)
├── architectures/             reference deployment patterns        (Phase 2)
├── security/                  RBAC, TLS, certificates              (Phase 2)
├── troubleshooting/           debug tips
├── reference/                 generated CRD docs, CRD changelog, feature gates
└── rfcs/                      design proposals (unchanged)
```

Each folder contains a `README.md` that links to its sibling files in reading order. GitHub renders this automatically when the folder is opened, which is how navigation works without a site generator.


## Rollout Plan

**Phase 1 - Mechanical reorganization.** Move existing content into the new tree. No rewrites, some minor edits. Slim the root README to navigational content. Only folders that receive moved content are created - no stub landing pages for planned sections. This is the scope of the proof-of-concept PR.

**Phase 2 - Fill the content gaps.** Independent PRs, each adding a section together with its folder:

- `docs/use-cases/`: collecting Prometheus metrics with the Target Allocator, collecting logs, collecting traces from auto-instrumentation, tail sampling.
- `docs/security/`: RBAC for the operator and for each component in the Kubernetes collector distribution; TLS between operator/collectors/Target Allocator; cert-manager interaction.
- `docs/architectures/`: agent-only, gateway, agent-plus-gateway, sidecar-per-app, multi-cluster.
- `docs/concepts/kubernetes-distro.md`: catalog of components in the operator-managed collector image.

These proceed independently. Each PR adds its folder to the root README's navigation.

**Phase 3 - Autogeneration.** Add an `internal/tools/docgen` Go module and a `make docs-gen` target. Outputs land under `docs/reference/`:

- `operator-flags.md` - from the `pflag.FlagSet` constructed in [`main.go`](../../main.go).
- `target-allocator-flags.md` - from [`cmd/otel-allocator/main.go`](../../cmd/otel-allocator/main.go).
- `feature-gates.md` - from [`pkg/featuregate/featuregate.go`](../../pkg/featuregate/featuregate.go) via `featuregate.GlobalRegistry().VisitAll(...)`.

CI gains a `docs-check` target mirroring the existing api-docs verify step in the [`Makefile`](../../Makefile).

Phase 3 has no dependency on Phase 2 and can land in parallel if a contributor picks it up.


## Open questions

1. **README example asymmetry.** The slimmed README in the Phase 1 PoC still contains a Collector CR example but nothing for the other three CRDs. Three resolutions:
    - **A.** Add minimal CR examples for `Instrumentation`, `TargetAllocator`, and `OpAMPBridge`. README grows back to ~250 lines but treats the CRDs symmetrically.
    - **B.** Move the Collector example out of the README into `docs/getting-started/README.md` and make the README's "Getting started" section purely navigational. README stays ~150 lines.
    - **C.** Keep as-is and rename the README section to "Quickstart: Collector" to acknowledge the asymmetry.

   **Recommendation: B.**

2. **Long-term home of [`cmd/otel-allocator/README.md`](../../cmd/otel-allocator/README.md).** The PoC moves its content to `docs/target-allocator/README.md` and leaves a one-line stub at the original path. Acceptable, or should the code-adjacent README hold developer-oriented content (build/test instructions) and `docs/target-allocator/` hold only user-facing material?

3. **`docs/rfcs/` location.** Currently top-level under `docs/`. Should it move under `docs/contributing/rfcs/` once a contributing section exists, or remain at the top level for visibility? Out of scope for Phase 1; flagged for later.


## Limitations

Plain Markdown on GitHub has known constraints versus a site generator: no global search across the docs tree, no versioned docs, and only the cross-link checking that [linkspector](https://github.com/UmbrellaDocs/linkspector) (already configured via `.linkspector.yml`) catches. These are acceptable trade-offs at this stage; revisiting them if the docs outgrow plain Markdown is a separate decision.
