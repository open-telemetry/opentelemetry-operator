# E2E test apps

Container images used by the e2e test suites: sample workloads for the
auto-instrumentation tests (`java`, `nodejs`, `python`, `dotnet`, `golang`,
`apache-httpd`), a fake OpAMP server (`bridge-server`), and a scrape target
with basic auth (`metrics-basic-auth`).

## Image lifecycle

Test manifests reference these images as published from `main`:
`ghcr.io/open-telemetry/opentelemetry-operator/e2e-test-app-<name>:main`
(plus `e2e-test-app-python:main-3.10` for the oldest supported Python). The
`publish-test-e2e-images` workflow builds and publishes them on merge; Go-based
apps have unit tests that gate publishing.

When a change set touches anything under `tests/test-e2e-apps/`, the published
images do not include it yet — so the e2e workflow builds all app images from
source (tagged exactly as the manifests reference them) and loads them into
kind, shadowing the registry versions. That way incompatibilities between
changed apps and the tests surface in PR CI, not after the merge. Detection is
not fine-grained: any change under this directory rebuilds all the apps.

To do the same locally after modifying an app:

```bash
make load-image-test-e2e-apps
```

Kubernetes only prefers the loaded image while the pull policy is
`IfNotPresent` (the default for `:main`-tagged images) — don't set
`imagePullPolicy: Always` on these images in test manifests.

Dependency updates (Go modules and base images) for all apps are grouped by
renovate into a single weekly PR.
