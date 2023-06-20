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
LD_FLAGS ?= "-X ${VERSION_PKG}.version=${VERSION} -X ${VERSION_PKG}.buildDate=${VERSION_DATE} -X ${VERSION_PKG}.otelCol=${OTELCOL_VERSION} -X ${VERSION_PKG}.targetAllocator=${TARGETALLOCATOR_VERSION} -X ${VERSION_PKG}.operatorOpAMPBridge=${OPERATOR_OPAMP_BRIDGE_VERSION} -X ${VERSION_PKG}.autoInstrumentationJava=${AUTO_INSTRUMENTATION_JAVA_VERSION} -X ${VERSION_PKG}.autoInstrumentationNodeJS=${AUTO_INSTRUMENTATION_NODEJS_VERSION} -X ${VERSION_PKG}.autoInstrumentationPython=${AUTO_INSTRUMENTATION_PYTHON_VERSION} -X ${VERSION_PKG}.autoInstrumentationDotNet=${AUTO_INSTRUMENTATION_DOTNET_VERSION} -X ${VERSION_PKG}.autoInstrumentationGo=${AUTO_INSTRUMENTATION_GO_VERSION}  -X ${VERSION_PKG}.autoInstrumentationDotNet=${AUTO_INSTRUMENTATION_APACHE_HTTPD_VERSION}"
ARCH ?= $(shell go env GOARCH)

# Image URL to use all building/pushing image targets
IMG_PREFIX ?= ghcr.io/${USER}/opentelemetry-operator
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

CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true"

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
ensure-generate-is-noop: USER=open-telemetry
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
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: generate fmt vet manifests
	ENABLE_WEBHOOKS=$(ENABLE_WEBHOOKS) go run -ldflags ${LD_FLAGS} ./main.go --zap-devel

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
lint:
	golangci-lint run
	cd cmd/otel-allocator && golangci-lint run
	cd cmd/operator-opamp-bridge && golangci-lint run

# Generate code
.PHONY: generate
generate: controller-gen api-docs
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# end-to-tests
.PHONY: e2e
e2e:
	$(KUTTL) test

# end-to-end-test for testing upgrading
.PHONY: e2e-upgrade
e2e-upgrade: undeploy
	$(KUTTL) test --config kuttl-test-upgrade.yaml

# end-to-end-test for testing autoscale
.PHONY: e2e-autoscale
e2e-autoscale:
	$(KUTTL) test --config kuttl-test-autoscale.yaml

.PHONY: e2e-log-operator
e2e-log-operator:
	kubectl get pod -n opentelemetry-operator-system | grep "opentelemetry-operator" | awk '{print $$1}' | xargs -I {} kubectl logs -n opentelemetry-operator-system {} manager
	kubectl get deploy -A

.PHONY: prepare-e2e
prepare-e2e: kuttl set-image-controller container container-target-allocator container-operator-opamp-bridge start-kind cert-manager install-metrics-server install-openshift-routes load-image-all deploy
	TARGETALLOCATOR_IMG=$(TARGETALLOCATOR_IMG) SED_BIN="$(SED)" ./hack/modify-test-images.sh

.PHONY: scorecard-tests
scorecard-tests: operator-sdk
	$(OPERATOR_SDK) scorecard -w=5m bundle || (echo "scorecard test failed" && exit 1)


# Build the container image, used only for local dev purposes
# buildx is used to ensure same results for arm based systems (m1/2 chips)
.PHONY: container
container:
	docker buildx build --load --platform linux/${ARCH} -t ${IMG} --build-arg VERSION_PKG=${VERSION_PKG} --build-arg VERSION=${VERSION} --build-arg VERSION_DATE=${VERSION_DATE} --build-arg OTELCOL_VERSION=${OTELCOL_VERSION} --build-arg TARGETALLOCATOR_VERSION=${TARGETALLOCATOR_VERSION} --build-arg OPERATOR_OPAMP_BRIDGE_VERSION=${OPERATOR_OPAMP_BRIDGE_VERSION} --build-arg AUTO_INSTRUMENTATION_JAVA_VERSION=${AUTO_INSTRUMENTATION_JAVA_VERSION}  --build-arg AUTO_INSTRUMENTATION_NODEJS_VERSION=${AUTO_INSTRUMENTATION_NODEJS_VERSION} --build-arg AUTO_INSTRUMENTATION_PYTHON_VERSION=${AUTO_INSTRUMENTATION_PYTHON_VERSION} --build-arg AUTO_INSTRUMENTATION_DOTNET_VERSION=${AUTO_INSTRUMENTATION_DOTNET_VERSION} --build-arg AUTO_INSTRUMENTATION_GO_VERSION=${AUTO_INSTRUMENTATION_GO_VERSION} --build-arg AUTO_INSTRUMENTATION_APACHE_HTTPD_VERSION=${AUTO_INSTRUMENTATION_APACHE_HTTPD_VERSION} .

# Push the container image, used only for local dev purposes
.PHONY: container-push
container-push:
	docker push ${IMG}

.PHONY: container-target-allocator-push
container-target-allocator-push:
	docker push ${TARGETALLOCATOR_IMG}

.PHONY: container-target-allocator
container-target-allocator:
	docker buildx build  --load --platform linux/${ARCH} -t ${TARGETALLOCATOR_IMG} cmd/otel-allocator

.PHONY: container-operator-opamp-bridge
container-operator-opamp-bridge:
	docker buildx build --platform linux/${ARCH} -t ${OPERATOROPAMPBRIDGE_IMG} cmd/operator-opamp-bridge

.PHONY: start-kind
start-kind:
ifeq (true,$(START_KIND_CLUSTER))
	kind create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG)
endif

.PHONY: install-metrics-server
install-metrics-server:
	./hack/install-metrics-server.sh

.PHONY: install-openshift-routes
install-openshift-routes:
	./hack/install-openshift-routes.sh

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

KUSTOMIZE_VERSION ?= v5.0.3
CONTROLLER_TOOLS_VERSION ?= v0.12.0


.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

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
	GOBIN=$(LOCALBIN) go install go.opentelemetry.io/build-tools/chloggen@v0.3.0

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

