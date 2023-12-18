# Current Operator version
VERSION ?= "$(shell git describe --tags | sed 's/^v//')"
VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
VERSION_PKG ?= "github.com/open-telemetry/opentelemetry-operator/internal/version"
OTELCOL_VERSION ?= "$(shell grep -v '\#' versions.txt | grep opentelemetry-collector | awk -F= '{print $$2}')"
OPERATOR_VERSION ?= "$(shell grep -v '\#' versions.txt | grep operator= | awk -F= '{print $$2}')"
TARGETALLOCATOR_VERSION ?= "$(shell grep -v '\#' versions.txt | grep targetallocator | awk -F= '{print $$2}')"
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

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true,maxDescLen=200"

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

KUBE_VERSION ?= 1.24
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
	@git diff -s --exit-code bundle.Dockerfile || (echo "Build failed: the bundle.Dockerfile file has been changed. The file should be the same as generated one. Run 'make bundle' and update your PR." && git diff && exit 1)
	@git diff -s --exit-code docs/api.md || (echo "Build failed: the api.md file has been changed but the generated api.md file isn't up to date. Run 'make api-docs' and update your PR." && git diff && exit 1)

.PHONY: all
all: manager
.PHONY: ci
ci: test

# Run tests
# setup-envtest uses KUBEBUILDER_ASSETS which points to a directory with binaries (api-server, etcd and kubectl)
.PHONY: test
test: generate fmt vet ensure-generate-is-noop envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(KUBE_VERSION) -p path)" go test ${GOTEST_OPTS} ./...
	cd cmd/otel-allocator && KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(KUBE_VERSION) -p path)" go test ${GOTEST_OPTS} ./...
	cd cmd/operator-opamp-bridge && KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(KUBE_VERSION) -p path)" go test ${GOTEST_OPTS} ./...

# Build manager binary
.PHONY: manager
manager: generate fmt vet
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -o bin/manager_${ARCH} -ldflags "${COMMON_LDFLAGS} ${OPERATOR_LDFLAGS}" main.go

# Build target allocator binary
targetallocator:
	cd cmd/otel-allocator && CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -a -installsuffix cgo -o bin/targetallocator_${ARCH} -ldflags "${COMMON_LDFLAGS}" .

# Build opamp bridge binary
operator-opamp-bridge:
	cd cmd/operator-opamp-bridge && CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -a -installsuffix cgo -o bin/opampbridge_${ARCH} -ldflags "${COMMON_LDFLAGS}" .

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: generate fmt vet manifests
	ENABLE_WEBHOOKS=$(ENABLE_WEBHOOKS) go run -ldflags ${OPERATOR_LDFLAGS} ./main.go --zap-devel

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

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

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
	cd cmd/otel-allocator && $(GOLANGCI_LINT) run
	cd cmd/operator-opamp-bridge && $(GOLANGCI_LINT) run

# Generate code
.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# end-to-tests
.PHONY: e2e
e2e:
	$(KUTTL) test


# instrumentation end-to-tests
.PHONY: e2e-instrumentation
e2e-instrumentation:
	$(KUTTL) test --config kuttl-test-instrumentation.yaml

# end-to-end-test for PrometheusCR E2E tests
.PHONY: e2e-prometheuscr
e2e-prometheuscr:
	$(KUTTL) test --config kuttl-test-prometheuscr.yaml

# end-to-end-test for testing upgrading
.PHONY: e2e-upgrade
e2e-upgrade: undeploy
	$(KUTTL) test --config kuttl-test-upgrade.yaml

# end-to-end-test for testing autoscale
.PHONY: e2e-autoscale
e2e-autoscale:
	$(KUTTL) test --config kuttl-test-autoscale.yaml

# end-to-end-test for testing pdb support
.PHONY: e2e-pdb
e2e-pdb:
	$(KUTTL) test --config kuttl-test-pdb.yaml

# end-to-end-test for testing OpenShift cases
.PHONY: e2e-openshift
e2e-openshift:
	$(KUTTL) test --config kuttl-test-openshift.yaml

.PHONY: e2e-log-operator
e2e-log-operator:
	kubectl get pod -n opentelemetry-operator-system | grep "opentelemetry-operator" | awk '{print $$1}' | xargs -I {} kubectl logs -n opentelemetry-operator-system {} manager
	kubectl get deploy -A

# end-to-tests for multi-instrumentation
.PHONY: e2e-multi-instrumentation
e2e-multi-instrumentation:
	$(KUTTL) test --config kuttl-test-multi-instr.yaml

# OpAMPBridge CR end-to-tests
.PHONY: e2e-opampbridge
e2e-opampbridge:
	$(KUTTL) test --config kuttl-test-opampbridge.yaml

# Target allocator end-to-tests
.PHONY: e2e-targetallocator
e2e-targetallocator:
	$(KUTTL) test --config kuttl-test-targetallocator.yaml

.PHONY: prepare-e2e
prepare-e2e: kuttl set-image-controller container container-target-allocator container-operator-opamp-bridge start-kind cert-manager install-metrics-server install-targetallocator-prometheus-crds load-image-all deploy
	TARGETALLOCATOR_IMG=$(TARGETALLOCATOR_IMG) OPERATOROPAMPBRIDGE_IMG=$(OPERATOROPAMPBRIDGE_IMG) OPERATOR_IMG=$(IMG) SED_BIN="$(SED)" ./hack/modify-test-images.sh

.PHONY: enable-prometheus-feature-flag
enable-prometheus-feature-flag:
	$(SED) -i "s#--feature-gates=+operator.autoinstrumentation.go#--feature-gates=+operator.autoinstrumentation.go,+operator.observability.prometheus#g" config/default/manager_auth_proxy_patch.yaml


.PHONY: scorecard-tests
scorecard-tests: operator-sdk
	$(OPERATOR_SDK) scorecard -w=5m bundle || (echo "scorecard test failed" && exit 1)


# Build the container image, used only for local dev purposes
# buildx is used to ensure same results for arm based systems (m1/2 chips)
.PHONY: container
container: GOOS = linux
container: manager
	docker build -t ${IMG} .

# Push the container image, used only for local dev purposes
.PHONY: container-push
container-push:
	docker push ${IMG}

.PHONY: container-target-allocator-push
container-target-allocator-push:
	docker push ${TARGETALLOCATOR_IMG}

.PHONY: container-target-allocator
container-target-allocator: GOOS = linux
container-target-allocator: targetallocator
	docker build -t ${TARGETALLOCATOR_IMG} cmd/otel-allocator

.PHONY: container-operator-opamp-bridge
container-operator-opamp-bridge: GOOS = linux
container-operator-opamp-bridge: operator-opamp-bridge
	docker build -t ${OPERATOROPAMPBRIDGE_IMG} cmd/operator-opamp-bridge

.PHONY: start-kind
start-kind:
ifeq (true,$(START_KIND_CLUSTER))
	kind create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG)
endif

.PHONY: install-metrics-server
install-metrics-server:
	./hack/install-metrics-server.sh

.PHONY: install-prometheus-operator
install-prometheus-operator:
	./hack/install-prometheus-operator.sh

# This only installs the CRDs Target Allocator supports
.PHONY: install-targetallocator-prometheus-crds
install-targetallocator-prometheus-crds:
	./hack/install-targetallocator-prometheus-crds.sh

.PHONY: load-image-all
load-image-all: load-image-operator load-image-target-allocator load-image-operator-opamp-bridge

.PHONY: load-image-operator
load-image-operator: container
ifeq (true,$(START_KIND_CLUSTER))
	kind load --name $(KIND_CLUSTER_NAME) docker-image $(IMG)
else
	$(MAKE) container-push
endif


.PHONY: load-image-target-allocator
load-image-target-allocator: container-target-allocator
ifeq (true,$(START_KIND_CLUSTER))
	kind load --name $(KIND_CLUSTER_NAME) docker-image $(TARGETALLOCATOR_IMG)
else
	$(MAKE) container-target-allocator-push
endif


.PHONY: load-image-operator-opamp-bridge
load-image-operator-opamp-bridge:
	kind load --name $(KIND_CLUSTER_NAME) docker-image ${OPERATOROPAMPBRIDGE_IMG}

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
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
CHLOGGEN ?= $(LOCALBIN)/chloggen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

KUSTOMIZE_VERSION ?= v5.0.3
CONTROLLER_TOOLS_VERSION ?= v0.12.0
GOLANGCI_LINT_VERSION ?= v1.54.0


.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

CRDOC = $(shell pwd)/bin/crdoc
.PHONY: crdoc
crdoc: ## Download crdoc locally if necessary.
	$(call go-get-tool,$(CRDOC), fybrik.io/crdoc,v0.5.2)

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

.PHONY: kuttl
kuttl:
ifeq (, $(shell which kubectl-kuttl))
	echo ${PATH}
	ls -l /usr/local/bin
	which kubectl-kuttl

	@{ \
	set -e ;\
	echo "" ;\
	echo "ERROR: kuttl not found." ;\
	echo "Please check https://kuttl.dev/docs/cli.html for installation instructions and try again." ;\
	echo "" ;\
	exit 1 ;\
	}
else
KUTTL=$(shell which kubectl-kuttl)
endif

.PHONY: kind
kind:
ifeq (, $(shell which kind))
	@{ \
	set -e ;\
	echo "" ;\
	echo "ERROR: kind not found." ;\
	echo "Please check https://kind.sigs.k8s.io/docs/user/quick-start/#installation for installation instructions and try again." ;\
	echo "" ;\
	exit 1 ;\
	}
else
KIND=$(shell which kind)
endif

OPERATOR_SDK = $(shell pwd)/bin/operator-sdk
.PHONY: operator-sdk
operator-sdk:
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
.PHONY: bundle
bundle: kustomize operator-sdk manifests set-image-controller
	$(OPERATOR_SDK) generate kustomize manifests -q
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle
	./hack/ignore-createdAt-bundle.sh

.PHONY: reset
reset: kustomize operator-sdk manifests
	$(MAKE) VERSION=${OPERATOR_VERSION} set-image-controller
	$(OPERATOR_SDK)  generate kustomize manifests -q
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version ${OPERATOR_VERSION} $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle
	./hack/ignore-createdAt-bundle.sh
	git checkout config/manager/kustomization.yaml
	OPERATOR_IMG=local/opentelemetry-operator:e2e TARGETALLOCATOR_IMG=local/opentelemetry-operator-targetallocator:e2e OPERATOROPAMPBRIDGE_IMG=local/opentelemetry-operator-opamp-bridge:e2e DEFAULT_OPERATOR_IMG=$(IMG) DEFAULT_TARGETALLOCATOR_IMG=$(TARGETALLOCATOR_IMG) DEFAULT_OPERATOROPAMPBRIDGE_IMG=$(OPERATOROPAMPBRIDGE_IMG) SED_BIN="$(SED)" ./hack/modify-test-images.sh

# Build the bundle image, used only for local dev purposes
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push:
	docker push $(BUNDLE_IMG)

.PHONY: api-docs
api-docs: crdoc kustomize
	@{ \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ; \
	$(KUSTOMIZE) build config/crd -o $$TMP_DIR/crd-output.yaml ;\
	$(CRDOC) --resources $$TMP_DIR/crd-output.yaml --output docs/api.md ;\
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
chlog-update: chlog-install
	$(CHLOGGEN) update --version $(VERSION)


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
