# Standalone OpAMP Bridge Manifests

This directory contains a Kustomize base for running the OpAMP Bridge without installing the OpenTelemetry Operator, CRDs, or cert-manager.

It deploys the bridge into the `opentelemetry-opamp-bridge` namespace with a service account, ConfigMap-based configuration, RBAC for ConfigMaps and workload resources, and a single Deployment.

Use `make deploy-standalone-bridge` to deploy these manifests, or run `kustomize build cmd/operator-opamp-bridge/manifests/standalone` to inspect the rendered resources.

## OpenShift notes

The default Deployment is compatible with the OpenShift restricted SCC profile: it runs as non-root without pinning a UID, disables privilege escalation, drops all Linux capabilities, uses the runtime-default seccomp profile, and runs with a read-only root filesystem.