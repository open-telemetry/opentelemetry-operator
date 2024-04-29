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

# Image URL to use all building/pushing image targets
DOCKER_USER ?= open-telemetry
IMG_PREFIX ?= ghcr.io/${DOCKER_USER}/opentelemetry-operator
IMG_REPO ?= opentelemetry-operator
IMG ?= manager:both # playground. real val --> ${IMG_PREFIX}/${IMG_REPO}:${VERSION}
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

MANIFEST_DIR ?= config/crd/bases
# kubectl apply does not work on large CRDs.
CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true,maxDescLen=0"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# by default, do not run the manager with webhooks enabled. This only affects local runs, not the build or in-cluster deployments.
ENABLE_WEBHOOKS ?= true # Playground. real val --> false

# If we are running in CI, run go test in verbose mode
ifeq (,$(CI))
GOTEST_OPTS=-race
else
GOTEST_OPTS=-race -v
endif

START_KIND_CLUSTER ?= true

KUBE_VERSION ?= 1.29
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
all: manager targetallocator operator-opamp-bridge

# No lint here, as CI runs it separately
.PHONY: ci
ci: generate fmt vet test ensure-generate-is-noop

# Build manager binary
.PHONY: manager
manager: generate
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -o bin/manager_${ARCH} -ldflags "${COMMON_LDFLAGS} ${OPERATOR_LDFLAGS}" main.go

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

.PHONY: add-image-opampbridge
add-image-opampbridge:
	@$(MAKE) add-operator-arg OPERATOR_ARG=--operator-opamp-bridge-image=$(OPERATOROPAMPBRIDGE_IMG)

.PHONY: enable-operator-featuregates
enable-operator-featuregates: OPERATOR_ARG = --feature-gates=$(FEATUREGATES)
enable-operator-featuregates: add-operator-arg

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
prepare-e2e: chainsaw set-image-controller add-image-targetallocator add-image-opampbridge container container-target-allocator container-operator-opamp-bridge start-kind cert-manager install-metrics-server install-targetallocator-prometheus-crds load-image-all deploy

.PHONY: prepare-e2e-with-featuregates
prepare-e2e-with-featuregates: chainsaw enable-operator-featuregates prepare-e2e

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

.PHONY: container-operator-opamp-bridge-push
container-operator-opamp-bridge-push:
	docker push ${OPERATOROPAMPBRIDGE_IMG}

.PHONY: container-target-allocator
container-target-allocator: GOOS = linux
container-target-allocator: targetallocator
	docker build -t ${TARGETALLOCATOR_IMG} cmd/otel-allocator

.PHONY: container-operator-opamp-bridge
container-operator-opamp-bridge: GOOS = linux
container-operator-opamp-bridge: operator-opamp-bridge
	docker build -t ${OPERATOROPAMPBRIDGE_IMG} cmd/operator-opamp-bridge

.PHONY: start-kind
start-kind: kind
ifeq (true,$(START_KIND_CLUSTER))
	$(KIND) create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG) || true
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

KUSTOMIZE_VERSION ?= v5.0.3
CONTROLLER_TOOLS_VERSION ?= v0.14.0
GOLANGCI_LINT_VERSION ?= v1.57.2
KIND_VERSION ?= v0.20.0
CHAINSAW_VERSION ?= v0.1.7

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
	@test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	@test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

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
.PHONY: bundle
bundle: kustomize operator-sdk manifests set-image-controller api-docs
	$(OPERATOR_SDK) generate kustomize manifests -q
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	./hack/ignore-createdAt-bundle.sh
	./hack/add-openshift-annotations.sh
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: reset
reset: kustomize operator-sdk manifests
	$(MAKE) VERSION=${OPERATOR_VERSION} set-image-controller
	$(OPERATOR_SDK)  generate kustomize manifests -q
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version ${OPERATOR_VERSION} $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle
	./hack/ignore-createdAt-bundle.sh
	./hack/add-openshift-annotations.sh
	git checkout config/manager/kustomization.yaml

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
	TMP_MANIFEST_DIR=$$(mktemp -d) ; \
	cp -r config/crd/* $$TMP_MANIFEST_DIR; \
	$(MAKE) CRD_OPTIONS=$(CRD_OPTIONS),maxDescLen=1200 MANIFEST_DIR=$$TMP_MANIFEST_DIR/bases manifests ;\
	TMP_DIR=$$(mktemp -d) ; \
	$(KUSTOMIZE) build $$TMP_MANIFEST_DIR -o $$TMP_DIR/crd-output.yaml ;\
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
chlog-update: chlog-install chlog-insert-components
	$(CHLOGGEN) update --version $(VERSION)

.PHONY: chlog-insert-components
chlog-insert-components:
	@echo "### Components" > components.md
	@echo "" >>components.md
	@echo "* [OpenTelemetry Collector - v${OTELCOL_VERSION}](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v${OTELCOL_VERSION})" >>components.md
	@echo "* [OpenTelemetry Contrib - v${OTELCOL_VERSION}](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v${OTELCOL_VERSION})" >>components.md
	@echo "* [Java auto-instrumentation - v${AUTO_INSTRUMENTATION_JAVA_VERSION}](https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/tag/v${AUTO_INSTRUMENTATION_JAVA_VERSION})" >>components.md
	@echo "* [.NET auto-instrumentation - v${AUTO_INSTRUMENTATION_DOTNET_VERSION}](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/tag/$v{AUTO_INSTRUMENTATION_DOTNET_VERSION})" >>components.md
	@echo "* [Node.JS - v${AUTO_INSTRUMENTATION_NODEJS_VERSION}](https://github.com/open-telemetry/opentelemetry-js/releases/tag/experimental%2Fv${AUTO_INSTRUMENTATION_NODEJS_VERSION})" >>components.md
	@echo "* [Python - v${AUTO_INSTRUMENTATION_PYTHON_VERSION}](https://github.com/open-telemetry/opentelemetry-python-contrib/releases/tag/v${AUTO_INSTRUMENTATION_PYTHON_VERSION})" >>components.md
	@echo "* [Go - ${AUTO_INSTRUMENTATION_GO_VERSION}](https://github.com/open-telemetry/opentelemetry-go-instrumentation/releases/tag/${AUTO_INSTRUMENTATION_GO_VERSION})" >>components.md
	@echo "* [ApacheHTTPD - ${AUTO_INSTRUMENTATION_APACHE_HTTPD_VERSION}](https://github.com/open-telemetry/opentelemetry-cpp-contrib/releases/tag/webserver%2Fv${AUTO_INSTRUMENTATION_APACHE_HTTPD_VERSION})" >>components.md
	@echo "* [Nginx - ${AUTO_INSTRUMENTATION_NGINX_VERSION}](https://github.com/open-telemetry/opentelemetry-cpp-contrib/releases/tag/webserver%2Fv${AUTO_INSTRUMENTATION_NGINX_VERSION})" >>components.md
	@sed -i '/<!-- next version -->/r ./components.md' CHANGELOG.md
	@sed -i '/<!-- next version -->/G' CHANGELOG.md
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
