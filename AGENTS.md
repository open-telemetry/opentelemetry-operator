# AGENTS.md

This file provides guidance to AI assistants when working with code in this repository.

## Project Overview

The OpenTelemetry Operator is a Kubernetes operator for managing [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) deployments and auto-instrumentation of workloads. It automates deployment, configuration, and lifecycle management of OpenTelemetry components in Kubernetes environments including OpenShift.

**Primary Languages:** Go 1.24+
**Framework:** controller-runtime (Kubernetes operator framework)
**Testing:** Go testing + envtest for unit tests, Chainsaw for e2e tests

## Architecture

**Core Custom Resources:**
- **OpenTelemetryCollector (v1beta1)**: Manages collector deployments in multiple modes (Deployment, DaemonSet, StatefulSet, Sidecar)
- **Instrumentation (v1alpha1)**: Configures auto-instrumentation for workloads (Java, NodeJS, Python, .NET, Go, Apache HTTPD, Nginx)
- **OpAMPBridge (v1alpha1)**: Enables remote configuration management via OpAMP protocol
- **TargetAllocator (v1alpha1)**: Standalone CR for Prometheus target allocation, or embedded in OpenTelemetryCollector spec

**Core Components:**
- **Operator**: Main controller managing all CRDs and orchestrating deployments
- **Target Allocator**: Distributes Prometheus targets among collectors (supports consistent-hashing, least-weighted, per-node strategies)
- **OpAMP Bridge**: Connects collectors to OpAMP management servers for remote config
- **Auto-Instrumentation**: Injects language-specific OpenTelemetry SDKs into pods via mutation webhooks

**Directory Structure:**
- `apis/`: CRD definitions (`v1alpha1` experimental, `v1beta1` stable)
- `internal/controllers/`: Kubernetes reconciliation controllers
- `internal/manifests/`: Resource generation logic (Deployments, Services, ConfigMaps)
- `internal/instrumentation/`: Auto-instrumentation injection logic per language
- `internal/webhook/podmutation/`: Pod mutation webhooks for sidecar and auto-instrumentation injection
- `internal/config/`: Collector configuration parsing and manipulation
- `cmd/otel-allocator/`: Target Allocator source code
- `cmd/operator-opamp-bridge/`: OpAMP Bridge source code
- `autoinstrumentation/`: Dockerfiles for instrumentation images
- `config/`: Kubernetes deployment configurations and overlays
- `tests/`: End-to-end test suites (Chainsaw framework)
- `bundle/`: OLM bundle manifests (community and openshift variants)

**Architecture Patterns:**

*Controller Reconciliation:*
```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch the resource
    // 2. Handle deletion (check for deletion timestamp)
    // 3. Add finalizer if not present
    // 4. Build desired state using manifests package
    // 5. Reconcile actual state with desired state
    // 6. Update status
}
```

*Manifest Generation:* Keep generation pure (no k8s API calls), put in `internal/manifests/`, use `controllerutil.SetControllerReference()` for ownership

*Webhooks:* Mutation webhooks modify pods/resources before admission, validation webhooks reject invalid resources

## Development Environment Setup

### Prerequisites and First-time Setup
```bash
# Install required tools
make install-tools

# Install dependencies
go mod download

# Generate code and manifests
make update

# Run tests to verify setup
make test

# Install cert-manager (required for webhooks)
make cert-manager
```

### Common Development Commands
```bash
# Build and test
make manager             # Build operator binary
make test                # Run unit tests
make lint                # Run golangci-lint
make fmt                 # Format Go code and auto-fix issues
make vet                 # Run go vet
make precommit           # Run fmt, vet, lint, test, ensure-update-is-noop

# Code generation (CRITICAL: run after changing API types)
make generate            # Generate DeepCopy methods for API types
make manifests           # Generate CRDs, RBAC, webhooks
make bundle              # Generate OLM bundles (community + openshift)
make api-docs            # Generate API documentation from CRDs
make update              # Run all: generate, manifests, bundle, api-docs

# Local development (webhooks disabled)
make install run

# Deploy to cluster with custom image
IMG=quay.io/${USER}/opentelemetry-operator:dev make container container-push deploy

# E2E testing with kind
make prepare-e2e         # Creates kind cluster, builds and loads all images
make e2e                 # Run all e2e tests
make e2e-instrumentation # Auto-instrumentation tests
make e2e-targetallocator # Target Allocator tests
make e2e-opampbridge     # OpAMP Bridge tests
make stop-kind           # Delete kind cluster
```

### Development Workflow
```bash
# After making changes to API types (apis/**/*_types.go):
make update              # CRITICAL: Regenerates manifests, bundle, API docs

# Before committing:
make precommit           # Runs fmt, vet, lint, test, ensure-update-is-noop
```

## Code Style and Conventions

### Go Code Style
- Follow standard Go conventions (gofmt, go vet)
- Use golangci-lint configuration in `.golangci.yaml`
- Run `make fmt` before committing - it auto-fixes many issues
- Add kubebuilder markers (comments starting with `//+kubebuilder:`) above API struct fields for CRD generation
- Never edit `zz_generated.*.go` files - they're auto-generated

### Kubebuilder Markers
When adding fields to CRD structs in `apis/`:
```go
type MySpec struct {
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    RequiredField string `json:"requiredField"`

    // +optional
    OptionalField *string `json:"optionalField,omitempty"`
}
```

### Naming Conventions
- Use functions in `internal/naming/` for consistent resource naming
- Follow Kubernetes naming: lowercase, hyphens, DNS-compatible
- Label keys use dots: `app.kubernetes.io/name`
- Annotation keys use slashes: `sidecar.opentelemetry.io/inject`

## Testing Instructions

### Running Tests
```bash
# Unit tests
make test
go test ./internal/controllers/...                              # Specific package
go test ./internal/controllers -run TestReconcile_DeletedCollector  # Single test
go test -race -v ./...                                          # With race detector

# E2E tests (requires kind cluster via make prepare-e2e)
make e2e                 # All e2e tests
make e2e-instrumentation # Auto-instrumentation tests
make e2e-log-operator    # View operator logs after test
```

### Writing Tests
**Unit Tests:**
- Use `envtest` for controllers (provides real Kubernetes API)
- Mock external dependencies
- Table-driven tests preferred
- Place tests next to code: `foo.go` → `foo_test.go`
- Set `KUBEBUILDER_ASSETS` when running from IDE: `KUBEBUILDER_ASSETS=$(./bin/setup-envtest use -p path 1.34) go test ./...`

**E2E Tests:**
- Use Chainsaw framework (YAML-based)
- Place in `tests/e2e-*/` directories
- Files run in order: `00-install.yaml`, `01-assert.yaml`, etc.
- Each test should clean up after itself

**Test Coverage Expectations:**
- All new controller logic should have unit tests
- Bug fixes must include regression tests
- Public API changes need both unit and e2e tests
- Webhook changes require e2e tests (webhooks don't work in `make run`)

## Common Tasks and Workflows

### Adding a new field to OpenTelemetryCollector CRD
```bash
# 1. Edit apis/v1beta1/opentelemetrycollector_types.go
# 2. Add kubebuilder markers for validation
# 3. Regenerate everything
make update
# 4. Update controller to handle new field in internal/controllers/
# 5. Add tests
# 6. Verify CRD output
kubectl explain opentelemetrycollector.spec.yourNewField
```

### Changing API types (CRDs)
1. Edit structs in `apis/v1alpha1/` or `apis/v1beta1/`
2. Add/update kubebuilder markers for validation
3. Run `make update` - generates manifests, bundles, docs
4. Run `make precommit` - ensures everything is valid
5. Verify CRD changes: `git diff config/crd/bases/`

**CRITICAL:** Always run `make update` after API changes. CI will fail with "ensure-update-is-noop" if you forget.

### Changing controller logic
1. Modify reconciliation logic in `internal/controllers/`
2. Update corresponding manifest generation in `internal/manifests/` if needed
3. Add/update unit tests
4. Run `make test` to verify
5. For complex changes, add e2e test in appropriate `tests/e2e-*/` directory

### Changing webhooks
1. Modify webhook logic in `internal/webhook/`
2. Webhooks cannot be tested with `make run` - must use `make deploy`
3. Set up kind cluster: `make prepare-e2e`
4. Add e2e test to verify webhook behavior
5. Test both mutation and validation webhooks

### Adding new auto-instrumentation language
```bash
# 1. Add Dockerfile in autoinstrumentation/{language}/
# 2. Add injection logic in internal/instrumentation/{language}.go
# 3. Add spec field to apis/v1alpha1/instrumentation_types.go
# 4. Update webhook to handle new annotation
# 5. Add feature gate flag in main.go
# 6. Add e2e test in tests/e2e-instrumentation/
# 7. Update versions.txt
# 8. Run make update
```

### Debugging locally
```bash
# Run with detailed logging
ENABLE_WEBHOOKS=false go run -ldflags "${OPERATOR_LDFLAGS}" ./main.go --zap-devel --zap-log-level=debug

# Customize log output format
go run ./main.go --zap-encoder=json --zap-log-level=debug

# Available logging flags:
# --zap-devel: Development mode (console encoder, debug level)
# --zap-encoder: json or console
# --zap-log-level: debug, info, error, or integer > 0
# --zap-stacktrace-level: info, error, panic
# --zap-time-encoding: epoch, millis, nano, iso8601, rfc3339, rfc3339nano

# Or use delve debugger
dlv debug ./main.go -- --zap-devel
```

## Important Technical Details

**Versioning:** Component versions are managed in `versions.txt`
- Operator, target allocator, opamp-bridge versions should match release version
- OpenTelemetry Collector should match latest collector (major.minor typically match)
- Auto-instrumentation versions: some in `versions.txt`, some in `autoinstrumentation/*/version.txt`

**Critical Version Constraints:**
> **⚠️ WARNING**: DO NOT bump Java auto-instrumentation past `1.x.x` and .NET past `1.2.0`
> These introduce breaking HTTP semantic convention changes.

**Supported Versions:**
- Kubernetes: v1.25 to v1.34 (always supports versions maintained by upstream Kubernetes)
- Go: Follows [Go's release policy](https://go.dev/doc/devel/release#policy)
- Cert-Manager: v1.x required for webhooks
- Prometheus-Operator: v0.81.0 (for Target Allocator ServiceMonitor/PodMonitor support)

**Webhook Configuration:** Admission webhooks are automatically configured when deployed via manifests (disabled in `make run`)

**Testing Framework:**
- Unit tests: Go testing + envtest (provides real Kubernetes API)
- E2E tests: Chainsaw test runner (YAML-based test definitions)

**Bundle Variants:**
- `community`: Standard Kubernetes deployment
- `openshift`: OpenShift-specific features (Routes, SCCs, additional RBAC)

**Dependencies:**
- Kubernetes or OpenShift
- cert-manager (for TLS certificate management)
- Object storage (for some collector configurations)

## Configuration

**Operator Runtime Flags:**
```bash
# Set custom images
--target-allocator-image=custom:latest
--collector-image=custom:latest
--auto-instrumentation-java-image=custom:latest

# Enable experimental features
--enable-go-instrumentation=true
--enable-nginx-instrumentation=true
--enable-multi-instrumentation=true

# RBAC and webhook settings
--enable-webhooks=true
--create-rbac-permissions=true
```

**Target Allocator Strategies:**
- `consistent-hashing` (default) - Consistently assigns targets, rebalances when collector count changes
- `least-weighted` - Assigns to collector with fewest targets, more stable during scale changes
- `per-node` - Assigns targets to collector on same node (DaemonSet only, ignores control plane targets)

**Important**: Target Allocator only works with `statefulset` and `daemonset` deployment modes.

**OpAMP Bridge Labels:**
Collectors must be labeled to interact with OpAMP Bridge:
- `opentelemetry.io/opamp-reporting: "true"` - Reporting only
- `opentelemetry.io/opamp-managed: "true"` - Reporting and management

## Important Constraints

**Do NOT:**
- Edit any `zz_generated.*.go` files (they're auto-generated)
- Commit changes without running `make update` after API changes
- Modify bundle files directly (they're generated by `make bundle`)
- Fully validate collector configs (operator intentionally doesn't do this)
- Use `//go:generate` directives (use Makefile targets instead)
- Add dependencies on non-standard Kubernetes distributions without discussion

**DO:**
- Run `make precommit` before pushing
- Add tests for all new functionality
- Use existing patterns from similar controllers/webhooks
- Check CI requirements in `.github/workflows/`
- Update `versions.txt` when changing component versions
- Add changelog entries for user-facing changes
- Follow semantic versioning for API changes

## Commit and PR Guidelines

**Changelog entries:**
```bash
make chlog-new           # Create new changelog entry
# Edit .chloggen/{branch-name}.yaml with:
# - change_type: breaking, deprecation, new_component, enhancement, bug_fix
# - component: operator, target-allocator, auto-instrumentation, opamp
# - note: Description of change
# - issues: [1234]
make chlog-validate      # Validate entry
```

Skip changelog by adding `[chore]` to PR title or "Skip Changelog" label.

**Commit message format:** `<type>: <description>` (types: feat, fix, docs, style, refactor, test, chore)

**PR requirements:**
- All CI checks must pass
- `make ensure-update-is-noop` must succeed
- New features need documentation and tests
- Bug fixes need regression tests
- API changes need approval from maintainers

## Debugging Common Issues

**"ensure-update-is-noop" fails in CI:**
- Run `make update` locally and commit the generated files

**Unit tests fail with "unable to find api-server":**
- Set KUBEBUILDER_ASSETS: `KUBEBUILDER_ASSETS=$(./bin/setup-envtest use -p path 1.34) go test ./...`

**E2E tests fail:**
- Check operator logs: `make e2e-log-operator`
- Verify images loaded: `docker exec -it otel-operator-control-plane crictl images`
- Ensure prepare-e2e ran successfully

**Webhooks not working:**
- Can't test webhooks with `make run` - use `make deploy` instead
- Verify cert-manager is installed: `kubectl get pods -n cert-manager`
- Check webhook configuration: `kubectl get validatingwebhookconfigurations`

**Generated code out of sync:**
- Run `make update` - regenerates all derived files
- Check `git status` to see what changed
- Common causes: changed API types, changed kubebuilder markers

## Additional Resources

- [Kubernetes Operator development](https://sdk.operatorframework.io/docs/)
- [Kubebuilder book](https://book.kubebuilder.io/)
- [Controller-runtime GoDoc](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [OpenTelemetry Collector docs](https://opentelemetry.io/docs/collector/)
- Project docs in `docs/` directory
- API reference in `docs/api/`
