# Current Operator version
VERSION ?= $(shell git describe --tags | sed 's/^v//')
VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
VERSION_PKG ?= github.com/open-telemetry/opentelemetry-operator/internal/version
OTELCOL_VERSION ?= "$(shell grep -v '\#' versions.txt | grep opentelemetry-collector | awk -F= '{print $$2}')"
OPERATOR_VERSION ?= "$(shell grep -v '\#' versions.txt | grep operator= | awk -F= '{print $$2}')"
TARGETALLOCATOR_VERSION ?= $(shell grep -v '\#' versions.txt | grep targetallocator | awk -F= '{print $$2}')
OPERATOR_OPAMP_BRIDGE_VERSION ?= "$(shell grep -v '\#' versions.txt | grep operator-opamp-bridge | awk -F= '{print $$2}')"
AUTO_INSTRUMENTATION_JAVA_VERSION ?= "$(shell grep -v '\#' versions.txt | grep autoinstrumentation-java | awk -F= '{print $$2}')"
AUTO_INSTRUMENTATION_NODEJS_VERSION ?= "$(shell grep -v '\#' versions.txt | grep autoinstrumentation-nodejs | awk -F= '{print $$2}')"
AUTO_INSTRUMENTATION_PYTHON_VERSION ?= "$(shell grep -v '\#' versions.txt | grep autoinstrumentation-python | awk -F= '{print $$2}')"
AUTO_INSTRUMENTATION_DOTNET_VERSION ?= "$(shell grep -v '\#' versions.txt | grep autoinstrumentation-dotnet | awk -F= '{print $$2}')"
AUTO_INSTRUMENTATION_GO_VERSION ?= "$(shell grep -v '\#' versions.txt | grep autoinstrumentation-go | awk -F= '{print $$2}')"
AUTO_INSTRUMENTATION_APACHE_HTTPD_VERSION ?= "$(shell grep -v '\#' versions.txt | grep autoinstrumentation-apache-httpd | awk -F= '{print $$2}')"
AUTO_INSTRUMENTATION_NGINX_VERSION ?= "$(shell grep -v '\#' versions.txt | grep autoinstrumentation-nginx | awk -F= '{print $$2}')"
COMMON_LDFLAGS ?= -s -w
OPERATOR_LDFLAGS ?= -X ${VERSION_PKG}.version=${VERSION} -X ${VERSION_PKG}.buildDate=${VERSION_DATE} -X ${VERSION_PKG}.otelCol=${OTELCOL_VERSION} -X ${VERSION_PKG}.targetAllocator=${TARGETALLOCATOR_VERSION} -X ${VERSION_PKG}.operatorOpAMPBridge=${OPERATOR_OPAMP_BRIDGE_VERSION} -X ${VERSION_PKG}.autoInstrumentationJava=${AUTO_INSTRUMENTATION_JAVA_VERSION} -X ${VERSION_PKG}.autoInstrumentationNodeJS=${AUTO_INSTRUMENTATION_NODEJS_VERSION} -X ${VERSION_PKG}.autoInstrumentationPython=${AUTO_INSTRUMENTATION_PYTHON_VERSION} -X ${VERSION_PKG}.autoInstrumentationDotNet=${AUTO_INSTRUMENTATION_DOTNET_VERSION} -X ${VERSION_PKG}.autoInstrumentationGo=${AUTO_INSTRUMENTATION_GO_VERSION} -X ${VERSION_PKG}.autoInstrumentationApacheHttpd=${AUTO_INSTRUMENTATION_APACHE_HTTPD_VERSION} -X ${VERSION_PKG}.autoInstrumentationNginx=${AUTO_INSTRUMENTATION_NGINX_VERSION}
ARCH ?= $(shell go env GOARCH)
ifeq ($(shell uname), Darwin)
  SED_INPLACE := sed -i ''
else
  SED_INPLACE := sed -i
endif

# Image URL to use all building/pushing image targets
DOCKER_USER ?= open-telemetry
IMG_PREFIX ?= ghcr.io/${DOCKER_USER}/opentelemetry-operator
IMG_REPO ?= opentelemetry-operator
IMG ?= ${IMG_PREFIX}/${IMG_REPO}:${VERSION}
BUNDLE_IMG ?= ${IMG_PREFIX}/${IMG_REPO}-bundle:${VERSION}

TARGETALLOCATOR_IMG_REPO ?= target-allocator
TARGETALLOCATOR_IMG ?= ${IMG_PREFIX}/${TARGETALLOCATOR_IMG_REPO}:$(addprefix v,${VERSION})

OPERATOROPAMPBRIDGE_IMG_REPO ?= operator-opamp-bridge
OPERATOROPAMPBRIDGE_IMG ?= ${IMG_PREFIX}/${OPERATOROPAMPBRIDGE_IMG_REPO}:$(addprefix v,${VERSION})

BRIDGETESTSERVER_IMG_REPO ?= e2e-test-app-bridge-server
BRIDGETESTSERVER_IMG ?= ${IMG_PREFIX}/${BRIDGETESTSERVER_IMG_REPO}:ve2e

MUSTGATHER_IMG ?= ${IMG_PREFIX}/must-gather

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# by default, do not run the manager with webhooks enabled. This only affects local runs, not the build or in-cluster deployments.
ENABLE_WEBHOOKS ?= false

# If we are running in CI, run go test in verbose mode
ifeq (,$(CI))
GOTEST_OPTS=-race
else
GOTEST_OPTS=-race -v
endif

START_KIND_CLUSTER ?= true

KUBE_VERSION ?= 1.32
KIND_CONFIG ?= kind-$(KUBE_VERSION).yaml
KIND_CLUSTER_NAME ?= "otel-operator"

OPERATOR_SDK_VERSION ?= 1.29.0

CERTMANAGER_VERSION ?= 1.10.0

ifndef ignore-not-found
  ignore-not-found = false
endif

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(OPERATOR_VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif
MANIFEST_DIR ?= config/crd/bases

# kubectl apply does not work on large CRDs.
CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true,maxDescLen=0"

# Choose wich version to generate
BUNDLE_VARIANT ?= community
BUNDLE_DIR = ./bundle/$(BUNDLE_VARIANT)
MANIFESTS_DIR = config/manifests/$(BUNDLE_VARIANT)
BUNDLE_BUILD_GEN_FLAGS ?= $(BUNDLE_GEN_FLAGS) --output-dir . --kustomize-dir ../../$(MANIFESTS_DIR)

MIN_KUBERNETES_VERSION ?= 1.23.0
MIN_OPENSHIFT_VERSION ?= 4.12

## On MacOS, use gsed instead of sed, to make sed behavior
## consistent with Linux.
SED ?= $(shell which gsed 2>/dev/null || which sed)

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: VERSION=$(OPERATOR_VERSION)
ensure-generate-is-noop: DOCKER_USER=open-telemetry
ensure-generate-is-noop: set-image-controller generate bundle
	@# on make bundle config/manager/kustomization.yaml includes changes, which should be ignored for the below check
	@git restore config/manager/kustomization.yaml
	@git diff -s --exit-code apis/v1alpha1/zz_generated.*.go || (echo "Build failed: a model has been changed but the generated resources aren't up to date. Run 'make generate' and update your PR." && exit 1)
	@git diff -s --exit-code bundle config || (echo "Build failed: the bundle, config files has been changed but the generated bundle, config files aren't up to date. Run 'make bundle' and update your PR." && git diff && exit 1)
	@git diff -s --exit-code docs/api.md || (echo "Build failed: the api.md file has been changed but the generated api.md file isn't up to date. Run 'make api-docs' and update your PR." && git diff && exit 1)

.PHONY: all
all: manager targetallocator operator-opamp-bridge

# No lint here, as CI runs it separately
.PHONY: ci
ci: generate fmt vet test ensure-generate-is-noop

# Build manager binary
.PHONY: manager
manager: generate
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -o bin/manager_${ARCH} -ldflags "${COMMON_LDFLAGS} ${OPERATOR_LDFLAGS}" main.go

.PHONY: must-gather
must-gather:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -o bin/must-gather_${ARCH} -ldflags "${COMMON_LDFLAGS}" ./cmd/gather/main.go

# Build target allocator binary
.PHONY: targetallocator
targetallocator:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -o cmd/otel-allocator/bin/targetallocator_${ARCH} -ldflags "${COMMON_LDFLAGS}" ./cmd/otel-allocator

# Build opamp bridge binary
.PHONY: operator-opamp-bridge
operator-opamp-bridge: generate
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -o cmd/operator-opamp-bridge/bin/opampbridge_${ARCH} -ldflags "${COMMON_LDFLAGS}" ./cmd/operator-opamp-bridge

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: generate fmt vet manifests
	ENABLE_WEBHOOKS=$(ENABLE_WEBHOOKS) go run -ldflags "${OPERATOR_LDFLAGS}" ./main.go --zap-devel

# Install CRDs into a cluster
.PHONY: install
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
.PHONY: uninstall
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

# Set the controller image parameters
.PHONY: set-image-controller
set-image-controller: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}

.PHONY: add-operator-arg
add-operator-arg: PATCH = [{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"$(OPERATOR_ARG)"}]
add-operator-arg: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit add patch --kind Deployment --patch '$(PATCH)'

.PHONY: add-image-targetallocator
add-image-targetallocator:
	@$(MAKE) add-operator-arg OPERATOR_ARG=--target-allocator-image=$(TARGETALLOCATOR_IMG)

.PHONY: add-instrumentation-params
add-instrumentation-params:
	@$(MAKE) add-operator-arg OPERATOR_ARG=--enable-go-instrumentation=true

.PHONY: add-multi-instrumentation-params
add-multi-instrumentation-params:
	@$(MAKE) add-operator-arg OPERATOR_ARG=--enable-multi-instrumentation

.PHONY: add-image-opampbridge
add-image-opampbridge:
	@$(MAKE) add-operator-arg OPERATOR_ARG=--operator-opamp-bridge-image=$(OPERATOROPAMPBRIDGE_IMG)

.PHONY: add-rbac-permissions-to-operator
add-rbac-permissions-to-operator: manifests kustomize
	# Kustomize only allows patches in the folder where the kustomization is located
	# This folder is ignored by .gitignore
	mkdir -p config/rbac/extra-permissions-operator
	cp -r tests/e2e-automatic-rbac/extra-permissions-operator/* config/rbac/extra-permissions-operator
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/clusterresourcequotas.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/cronjobs.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/daemonsets.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/events.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/endpoints.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/extensions.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/namespaces.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/namespaces-status.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/nodes.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/nodes-proxy.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/nodes-spec.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/pod-status.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/rbac.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/replicaset.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/replicationcontrollers.yaml
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path extra-permissions-operator/resourcequotas.yaml

.PHONY: enable-targetallocator-cr
enable-targetallocator-cr:
	@$(MAKE) add-operator-arg OPERATOR_ARG='--feature-gates=operator.collector.targetallocatorcr'
	cd config/crd && $(KUSTOMIZE) edit add resource bases/opentelemetry.io_targetallocators.yaml

# Deploy controller in the current Kubernetes context, configured in ~/.kube/config
.PHONY: deploy
deploy: set-image-controller
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	go run hack/check-operator-ready.go 300

# Undeploy controller in the current Kubernetes context, configured in ~/.kube/config
.PHONY: undeploy
undeploy: set-image-controller
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

# Generates the released manifests
.PHONY: release-artifacts
release-artifacts: set-image-controller
	mkdir -p dist
	$(KUSTOMIZE) build config/default -o dist/opentelemetry-operator.yaml
	$(KUSTOMIZE) build config/overlays/openshift -o dist/opentelemetry-operator-openshift.yaml

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=${MANIFEST_DIR}

# Run tests
# setup-envtest uses KUBEBUILDER_ASSETS which points to a directory with binaries (api-server, etcd and kubectl)
.PHONY: test
test: envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(KUBE_VERSION) -p path)" go test ${GOTEST_OPTS} ./...

.PHONY: precommit
precommit: generate fmt vet lint test ensure-generate-is-noop reset

# Run go fmt against code
.PHONY: fmt
fmt:
	go fmt ./...

# Run go vet against code
.PHONY: vet
vet:
	go vet ./...

# Run go lint against code
.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run

# Generate code
.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# end-to-tests
.PHONY: e2e
e2e: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e

# e2e-native-sidecar
# NOTE: make sure the k8s featuregate "SidecarContainers" is set to true.
# NOTE: make sure the operator featuregate "operator.sidecarcontainers.native" is enabled.
.PHONY: e2e-native-sidecar
e2e-native-sidecar: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-native-sidecar

# end-to-end-test for testing automatic RBAC creation
.PHONY: e2e-automatic-rbac
e2e-automatic-rbac: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-automatic-rbac

# end-to-end-test for testing autoscale
.PHONY: e2e-autoscale
e2e-autoscale: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-autoscale

# instrumentation end-to-tests
.PHONY: e2e-instrumentation
e2e-instrumentation: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-instrumentation

.PHONY: e2e-log-operator
e2e-log-operator:
	kubectl get pod -n opentelemetry-operator-system | grep "opentelemetry-operator" | awk '{print $$1}' | xargs -I {} kubectl logs -n opentelemetry-operator-system {} manager
	kubectl get deploy -A

# end-to-tests for multi-instrumentation
.PHONY: e2e-multi-instrumentation
e2e-multi-instrumentation: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-multi-instrumentation

# OpAMPBridge CR end-to-tests
.PHONY: e2e-opampbridge
e2e-opampbridge: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-opampbridge

# end-to-end-test for testing pdb support
.PHONY: e2e-pdb
e2e-pdb: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-pdb

# end-to-end-test for PrometheusCR E2E tests
.PHONY: e2e-prometheuscr
e2e-prometheuscr: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-prometheuscr

# Target allocator end-to-tests
.PHONY: e2e-targetallocator
e2e-targetallocator: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-targetallocator

# Target allocator CR end-to-tests
.PHONY: e2e-targetallocator-cr
e2e-targetallocator-cr: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-targetallocator-cr

.PHONY: add-certmanager-permissions
add-certmanager-permissions: 
	# Kustomize only allows patches in the folder where the kustomization is located
	# This folder is ignored by .gitignore
	cp -r tests/e2e-ta-collector-mtls/certmanager-permissions config/rbac/certmanager-permissions
	cd config/rbac && $(KUSTOMIZE) edit add patch --kind ClusterRole --name manager-role --path certmanager-permissions/certmanager.yaml

# Target allocator collector mTLS end-to-tests
.PHONY: e2e-ta-collector-mtls
e2e-ta-collector-mtls: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-ta-collector-mtls

# end-to-end-test for Annotations/Labels Filters
.PHONY: e2e-metadata-filters
e2e-metadata-filters: chainsaw
	$(CHAINSAW) test --test-dir ./tests/e2e-metadata-filters

# end-to-end-test for testing upgrading
.PHONY: e2e-upgrade
e2e-upgrade: undeploy chainsaw
	kubectl apply -f ./tests/e2e-upgrade/upgrade-test/opentelemetry-operator-v0.86.0.yaml
	go run hack/check-operator-ready.go
	$(CHAINSAW) test --test-dir ./tests/e2e-upgrade

.PHONY: prepare-e2e
prepare-e2e: chainsaw set-image-controller add-image-targetallocator add-image-opampbridge container container-target-allocator container-operator-opamp-bridge container-bridge-test-server start-kind cert-manager install-metrics-server install-targetallocator-prometheus-crds load-image-all deploy

.PHONY: scorecard-tests
scorecard-tests: operator-sdk
	$(OPERATOR_SDK) scorecard -w=5m bundle/community || (echo "scorecard test for community bundle failed" && exit 1)
	$(OPERATOR_SDK) scorecard -w=5m bundle/openshift || (echo "scorecard test for openshift bundle failed" && exit 1)


# Build the container image, used only for local dev purposes
# buildx is used to ensure same results for arm based systems (m1/2 chips)
.PHONY: container
container: GOOS = linux
container: manager
	docker build --load -t ${IMG} .

# Push the container image, used only for local dev purposes
.PHONY: container-push
container-push:
	docker push ${IMG}

.PHONY: container-target-allocator-push
container-target-allocator-push:
	docker push ${TARGETALLOCATOR_IMG}

.PHONY: container-operator-opamp-bridge-push
container-operator-opamp-bridge-push:
	docker push ${OPERATOROPAMPBRIDGE_IMG}

.PHONY: container-target-allocator
container-target-allocator: GOOS = linux
container-target-allocator: targetallocator
	docker build --load -t ${TARGETALLOCATOR_IMG} cmd/otel-allocator

.PHONY: container-operator-opamp-bridge
container-operator-opamp-bridge: GOOS = linux
container-operator-opamp-bridge: operator-opamp-bridge
	docker build --load -t ${OPERATOROPAMPBRIDGE_IMG} cmd/operator-opamp-bridge

.PHONY: container-bridge-test-server
container-bridge-test-server: GOOS = linux
container-bridge-test-server:
	docker build --load -t ${BRIDGETESTSERVER_IMG} tests/test-e2e-apps/bridge-server

.PHONY: container-must-gather
container-must-gather: GOOS = linux
container-must-gather: must-gather
	docker build -f cmd/gather/Dockerfile --load -t ${MUSTGATHER_IMG} .

.PHONY: container-must-gather-push
container-must-gather-push:
	docker push ${MUSTGATHER_IMG}

.PHONY: start-kind
start-kind: kind
ifeq (true,$(START_KIND_CLUSTER))
	$(KIND) create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG) || true
endif

.PHONY: install-metrics-server
install-metrics-server:
	./hack/install-metrics-server.sh

# This only installs the CRDs Target Allocator supports
.PHONY: install-targetallocator-prometheus-crds
install-targetallocator-prometheus-crds:
	./hack/install-targetallocator-prometheus-crds.sh

.PHONY: load-image-all
load-image-all: load-image-operator load-image-target-allocator load-image-operator-opamp-bridge load-image-bridge-test-server

.PHONY: load-image-operator
load-image-operator: container kind
ifeq (true,$(START_KIND_CLUSTER))
	$(KIND) load --name $(KIND_CLUSTER_NAME) docker-image $(IMG)
else
	$(MAKE) container-push
endif

.PHONY: load-image-target-allocator
load-image-target-allocator: container-target-allocator kind
ifeq (true,$(START_KIND_CLUSTER))
	$(KIND) load --name $(KIND_CLUSTER_NAME) docker-image $(TARGETALLOCATOR_IMG)
else
	$(MAKE) container-target-allocator-push
endif

.PHONY: load-image-bridge-test-server
load-image-bridge-test-server: container-bridge-test-server kind
	$(KIND) load --name $(KIND_CLUSTER_NAME) docker-image ${BRIDGETESTSERVER_IMG}

.PHONY: load-image-operator-opamp-bridge
load-image-operator-opamp-bridge: container-operator-opamp-bridge kind
	$(KIND) load --name $(KIND_CLUSTER_NAME) docker-image ${OPERATOROPAMPBRIDGE_IMG}

.PHONY: cert-manager
cert-manager: cmctl
	# Consider using cmctl to install the cert-manager once install command is not experimental
	kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v${CERTMANAGER_VERSION}/cert-manager.yaml
	$(CMCTL) check api --wait=5m

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
CMCTL = $(shell pwd)/bin/cmctl
.PHONY: cmctl
cmctl:
	@{ \
	set -e ;\
	if (`pwd`/bin/cmctl version | grep ${CERTMANAGER_VERSION}) > /dev/null 2>&1 ; then \
		exit 0; \
	fi ;\
	TMP_DIR=$$(mktemp -d) ;\
	curl -L -o $$TMP_DIR/cmctl.tar.gz https://github.com/jetstack/cert-manager/releases/download/v$(CERTMANAGER_VERSION)/cmctl-`go env GOOS`-`go env GOARCH`.tar.gz ;\
	tar xzf $$TMP_DIR/cmctl.tar.gz -C $$TMP_DIR ;\
	[ -d bin ] || mkdir bin ;\
	mv $$TMP_DIR/cmctl $(CMCTL) ;\
	rm -rf $$TMP_DIR ;\
	}

KUSTOMIZE ?= $(LOCALBIN)/kustomize
KIND ?= $(LOCALBIN)/kind
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
CHLOGGEN ?= $(LOCALBIN)/chloggen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
CHAINSAW ?= $(LOCALBIN)/chainsaw

# renovate: datasource=go depName=sigs.k8s.io/kustomize/kustomize/v5
KUSTOMIZE_VERSION ?= v5.6.0
# renovate: datasource=go depName=sigs.k8s.io/controller-tools/cmd/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.17.1
# renovate: datasource=go depName=github.com/golangci/golangci-lint/cmd/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.63.4
# renovate: datasource=go depName=sigs.k8s.io/kind
KIND_VERSION ?= v0.26.0
# renovate: datasource=go depName=github.com/kyverno/chainsaw
CHAINSAW_VERSION ?= v0.2.12

.PHONY: install-tools
install-tools: kustomize golangci-lint kind controller-gen envtest crdoc kind operator-sdk chainsaw

.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: kind
kind: ## Download kind locally if necessary.
	$(call go-get-tool,$(KIND),sigs.k8s.io/kind,$(KIND_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-get-tool,$(CONTROLLER_GEN), sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-get-tool,$(ENVTEST), sigs.k8s.io/controller-runtime/tools/setup-envtest,latest)

CRDOC = $(shell pwd)/bin/crdoc
.PHONY: crdoc
crdoc: ## Download crdoc locally if necessary.
	$(call go-get-tool,$(CRDOC), fybrik.io/crdoc,v0.5.2)

.PHONY: chainsaw
chainsaw: ## Find or download chainsaw
	$(call go-get-tool,$(CHAINSAW), github.com/kyverno/chainsaw,$(CHAINSAW_VERSION))

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
go get -d $(2)@$(3) ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

OPERATOR_SDK = $(shell pwd)/bin/operator-sdk
.PHONY: operator-sdk
operator-sdk: $(LOCALBIN)
	@{ \
	set -e ;\
	if (`pwd`/bin/operator-sdk version | grep ${OPERATOR_SDK_VERSION}) > /dev/null 2>&1 ; then \
		exit 0; \
	fi ;\
	[ -d bin ] || mkdir bin ;\
	curl -L -o $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk_`go env GOOS`_`go env GOARCH`;\
	chmod +x $(OPERATOR_SDK) ;\
	}

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: generate-bundle
generate-bundle: kustomize operator-sdk manifests set-image-controller api-docs
	sed -e 's/minKubeVersion: .*/minKubeVersion: $(MIN_KUBERNETES_VERSION)/' config/manifests/$(BUNDLE_VARIANT)/bases/opentelemetry-operator.clusterserviceversion.yaml

	$(OPERATOR_SDK) generate kustomize manifests -q --input-dir $(MANIFESTS_DIR) --output-dir $(MANIFESTS_DIR)
	cd $(BUNDLE_DIR) && cp ../../PROJECT . && $(KUSTOMIZE) build ../../$(MANIFESTS_DIR) | $(OPERATOR_SDK) generate bundle $(BUNDLE_BUILD_GEN_FLAGS) && rm PROJECT

	# Workaround for https://github.com/operator-framework/operator-sdk/issues/4992
	echo "" >> bundle/$(BUNDLE_VARIANT)/bundle.Dockerfile
	echo "LABEL com.redhat.openshift.versions=v$(MIN_OPENSHIFT_VERSION)" >> bundle/$(BUNDLE_VARIANT)/bundle.Dockerfile
	echo "" >> bundle/$(BUNDLE_VARIANT)/metadata/annotations.yaml
	echo "  com.redhat.openshift.versions: v$(MIN_OPENSHIFT_VERSION)" >> bundle/$(BUNDLE_VARIANT)/metadata/annotations.yaml

	$(OPERATOR_SDK) bundle validate $(BUNDLE_DIR)
	./hack/ignore-createdAt-bundle.sh

.PHONY: bundle
bundle:
	BUNDLE_VARIANT=community VERSION=$(VERSION) $(MAKE) generate-bundle
	BUNDLE_VARIANT=openshift VERSION=$(VERSION) $(MAKE) generate-bundle


.PHONY: reset
reset: kustomize operator-sdk manifests
	$(MAKE) VERSION=${OPERATOR_VERSION} set-image-controller
	$(OPERATOR_SDK)  generate kustomize manifests -q --input-dir config/manifests/community --output-dir config/manifests/community
	$(OPERATOR_SDK)  generate kustomize manifests -q --input-dir config/manifests/openshift --output-dir config/manifests/openshift

	$(KUSTOMIZE) build config/manifests/community | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS) --kustomize-dir config/manifests/community --output-dir bundle/community
	$(KUSTOMIZE) build config/manifests/openshift |$(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS) --kustomize-dir config/manifests/openshift --output-dir bundle/openshift

	# Workaround for https://github.com/operator-framework/operator-sdk/issues/4992
	echo "" >> bundle/community/metadata/annotations.yaml
	echo "  com.redhat.openshift.versions: v$(MIN_OPENSHIFT_VERSION)" >> bundle/community/metadata/annotations.yaml
	echo "" >> bundle/openshift/metadata/annotations.yaml
	echo "  com.redhat.openshift.versions: v$(MIN_OPENSHIFT_VERSION)" >> bundle/openshift/metadata/annotations.yaml

	$(OPERATOR_SDK) bundle validate ./bundle/community
	$(OPERATOR_SDK) bundle validate ./bundle/openshift
	rm bundle.Dockerfile
	git checkout config/manager/kustomization.yaml
	./hack/ignore-createdAt-bundle.sh

# Build the bundle image, used only for local dev purposes
.PHONY: bundle-build
bundle-build:
	docker build --load -f ./bundle/$(BUNDLE_VARIANT)/bundle.Dockerfile -t $(BUNDLE_IMG) ./bundle/$(BUNDLE_VARIANT)

.PHONY: bundle-push
bundle-push:
	docker push $(BUNDLE_IMG)

.PHONY: api-docs
api-docs: crdoc kustomize
	@{ \
	set -e ;\
	TMP_MANIFEST_DIR=$$(mktemp -d) ; \
	cp -r config/crd/* $$TMP_MANIFEST_DIR; \
	$(MAKE) CRD_OPTIONS=$(CRD_OPTIONS),maxDescLen=1200 MANIFEST_DIR=$$TMP_MANIFEST_DIR/bases manifests ;\
	TMP_DIR=$$(mktemp -d) ; \
	$(KUSTOMIZE) build $$TMP_MANIFEST_DIR -o $$TMP_DIR ;\
	mkdir -p docs/api ;\
	for crdmanifest in $$TMP_DIR/*; do \
	  filename="$$(basename -s .opentelemetry.io.yaml $$crdmanifest)" ;\
	  filename="$${filename#apiextensions.k8s.io_v1_customresourcedefinition_}" ;\
	  $(CRDOC) --resources $$crdmanifest --output docs/api/$$filename.md ;\
	done;\
	}


.PHONY: chlog-install
chlog-install: $(CHLOGGEN)
$(CHLOGGEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install go.opentelemetry.io/build-tools/chloggen@v0.11.0

FILENAME?=$(shell git branch --show-current)
.PHONY: chlog-new
chlog-new: chlog-install
	$(CHLOGGEN) new --filename $(FILENAME)

.PHONY: chlog-validate
chlog-validate: chlog-install
	$(CHLOGGEN) validate

.PHONY: chlog-preview
chlog-preview: chlog-install
	$(CHLOGGEN) update --dry

.PHONY: chlog-update
chlog-update: chlog-install chlog-insert-components
	$(CHLOGGEN) update --version $(VERSION)

.PHONY: chlog-insert-components
chlog-insert-components:
	@echo "### Components" > components.md
	@echo "" >>components.md
	@echo "* [OpenTelemetry Collector - v${OTELCOL_VERSION}](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v${OTELCOL_VERSION})" >>components.md
	@echo "* [OpenTelemetry Contrib - v${OTELCOL_VERSION}](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v${OTELCOL_VERSION})" >>components.md
	@echo "* [Java auto-instrumentation - v${AUTO_INSTRUMENTATION_JAVA_VERSION}](https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/tag/v${AUTO_INSTRUMENTATION_JAVA_VERSION})" >>components.md
	@echo "* [.NET auto-instrumentation - v${AUTO_INSTRUMENTATION_DOTNET_VERSION}](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/tag/v${AUTO_INSTRUMENTATION_DOTNET_VERSION})" >>components.md
	@echo "* [Node.JS - v${AUTO_INSTRUMENTATION_NODEJS_VERSION}](https://github.com/open-telemetry/opentelemetry-js/releases/tag/experimental%2Fv${AUTO_INSTRUMENTATION_NODEJS_VERSION})" >>components.md
	@echo "* [Python - v${AUTO_INSTRUMENTATION_PYTHON_VERSION}](https://github.com/open-telemetry/opentelemetry-python-contrib/releases/tag/v${AUTO_INSTRUMENTATION_PYTHON_VERSION})" >>components.md
	@echo "* [Go - ${AUTO_INSTRUMENTATION_GO_VERSION}](https://github.com/open-telemetry/opentelemetry-go-instrumentation/releases/tag/${AUTO_INSTRUMENTATION_GO_VERSION})" >>components.md
	@echo "* [ApacheHTTPD - ${AUTO_INSTRUMENTATION_APACHE_HTTPD_VERSION}](https://github.com/open-telemetry/opentelemetry-cpp-contrib/releases/tag/webserver%2Fv${AUTO_INSTRUMENTATION_APACHE_HTTPD_VERSION})" >>components.md
	@echo "* [Nginx - ${AUTO_INSTRUMENTATION_NGINX_VERSION}](https://github.com/open-telemetry/opentelemetry-cpp-contrib/releases/tag/webserver%2Fv${AUTO_INSTRUMENTATION_NGINX_VERSION})" >>components.md
	@$(SED_INPLACE) '/<!-- next version -->/r ./components.md' CHANGELOG.md
	@$(SED_INPLACE) '/<!-- next version -->/G' CHANGELOG.md
	@rm components.md

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.28.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= ${IMG_PREFIX}/${IMG_REPO}-catalog:${VERSION}

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm bundle-build bundle-push ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	docker push $(CATALOG_IMG)
