# Current Operator version
VERSION ?= "$(shell git describe --tags | grep -Po "([\d\.]+)")" ## version, without the 'v' prefix
VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
VERSION_PKG ?= "github.com/open-telemetry/opentelemetry-operator/internal/version"
OTELCOL_VERSION ?= "$(shell grep -v '\#' versions.txt | grep opentelemetry-collector | awk -F= '{print $$2}')"
LD_FLAGS ?= "-X ${VERSION_PKG}.version=${VERSION} -X ${VERSION_PKG}.buildDate=${VERSION_DATE} -X ${VERSION_PKG}.otelCol=${OTELCOL_VERSION}"

# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG_PREFIX ?= quay.io/${USER}
IMG ?= ${IMG_PREFIX}:${VERSION}

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# by default, do not run the manager with webhooks enabled. This only affects local runs, not the build or in-cluster deployments.
ENABLE_WEBHOOKS ?= false

# If we are running in CI, run ginkgo with the recommended CI settings
ifeq (,$(CI))
GINKGO_OPTS=-r
else
GINKGO_OPTS=-r --randomizeAllSpecs --randomizeSuites --failOnPending --cover --trace --race --progress --compilers=2 -v
endif

all: manager
ci: test

# Run tests
test: ginkgo generate fmt vet manifests
	$(GINKGO) $(GINKGO_OPTS)

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	ENABLE_WEBHOOKS=$(ENABLE_WEBHOOKS) go run -ldflags ${LD_FLAGS} ./main.go --zap-devel

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Set the controller image parameters
set-image-controller: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: set-image-controller
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
release-artifacts: set-image-controller bundle
	mkdir -p dist
	$(KUSTOMIZE) build config/default -o dist/opentelemetry-operator.yaml
	tar czf dist/bundle.tar.gz bundle

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build -t ${IMG} --build-arg VERSION_PKG=${VERSION_PKG} --build-arg VERSION=${VERSION} --build-arg VERSION_DATE=${VERSION_DATE} --build-arg OTELCOL_VERSION=${OTELCOL_VERSION} .

# Push the docker image
docker-push:
	docker push ${IMG}

cert-manager:
	kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.16.1/cert-manager.yaml

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

# find or download ginkgo
# download ginkgo if necessary
ginkgo:
ifeq (, $(shell which ginkgo))
	@{ \
	set -e ;\
	go get github.com/onsi/ginkgo/ginkgo@v1.14.2 ;\
	}
GINKGO=$(GOBIN)/ginkgo
else
GINKGO=$(shell which ginkgo)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

operator-sdk:
ifeq (, $(shell which operator-sdk))
	@{ \
	set -e ;\
	echo "" ;\
	echo "ERROR: operator-sdk not found." ;\
	echo "Please check https://sdk.operatorframework.io for installation instructions and try again." ;\
	echo "" ;\
	exit 1 ;\
	}
else
OPERATOR_SDK=$(shell which operator-sdk)
endif

# Generate bundle manifests and metadata, then validate generated files.
bundle: operator-sdk manifests
	$(OPERATOR_SDK) generate kustomize manifests -q
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

# Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

tools: ginkgo kustomize controller-gen operator-sdk
