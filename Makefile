OPERATOR_NAME ?= opentelemetry-operator
OPERATOR_VERSION ?= "$(shell git describe --tags)"
VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

GO_FLAGS ?= GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on
GOPATH ?= $(shell go env GOPATH)
KUBERNETES_CONFIG ?= "${HOME}/.kube/config"
WATCH_NAMESPACE ?= ""
BIN_DIR ?= "build/_output/bin"

NAMESPACE ?= "quay.io/${USER}"
BUILD_IMAGE ?= "${NAMESPACE}/${OPERATOR_NAME}:latest"
OUTPUT_BINARY ?= "${BIN_DIR}/${OPERATOR_NAME}"
VERSION_PKG ?= "github.com/open-telemetry/opentelemetry-operator/pkg/version"
LD_FLAGS ?= "-X ${VERSION_PKG}.version=${OPERATOR_VERSION} -X ${VERSION_PKG}.buildDate=${VERSION_DATE} -X ${VERSION_PKG}.otelCol=${OTELSVC_VERSION}"

OTELSVC_VERSION ?= "$(shell grep -v '\#' opentelemetry.version | grep opentelemetry-collector | awk -F= '{print $$2}')"

.DEFAULT_GOAL := build

.PHONY: discard-go-mod-changes
discard-go-mod-changes:
	@# 'go list' will update go.mod/go.sum and there's no way to prevent it (not even with -mod=readonly)
	@git checkout -- go.mod go.sum

.PHONY: check
check: ensure-generate-is-noop discard-go-mod-changes
	@echo Checking...
	@git diff -s --exit-code . || (echo "Build failed: one or more source files aren't properly formatted. Run 'make format' and update your PR." && exit 1)

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: generate format
	@git diff -s --exit-code pkg/apis/opentelemetry/v1alpha1/zz_generated.*.go || (echo "Build failed: a model has been changed but the deep copy functions aren't up to date. Run 'make generate' and update your PR." && exit 1)
	@git diff -s --exit-code pkg/client/versioned || (echo "Build failed: the versioned clients aren't up to date. Run 'make generate'." && exit 1)

.PHONY: format
format:
	@echo Formatting code...
	@GOPATH=${GOPATH} .ci/format.sh

.PHONY: security
security:
	@echo Security...
	@gosec -quiet ./... 2>/dev/null

.PHONY: build
build: format
	@echo Building operator binary...
	@${GO_FLAGS} go build -o ${OUTPUT_BINARY} -ldflags ${LD_FLAGS} ./cmd/manager/main.go

.PHONY: container
container:
	@echo Building container ${BUILD_IMAGE}...
	@mkdir -p build/_output
	@BUILD_IMAGE=${BUILD_IMAGE} ./.ci/build-container.sh

.PHONY: run
run: crd
	@OPERATOR_NAME=${OPERATOR_NAME} operator-sdk run local --watch-namespace="${WATCH_NAMESPACE}" --operator-flags "--zap-devel" --go-ldflags ${LD_FLAGS}

.PHONY: clean
clean:
	@echo Cleaning...

.PHONY: crd
crd:
	@kubectl create -f deploy/crds/opentelemetry.io_opentelemetrycollectors_crd.yaml 2>&1 | grep -v "already exists" || true

.PHONY: generate
generate: internal-generate format

.PHONY: internal-generate
internal-generate:
	@GOPATH=${GOPATH} ./.ci/generate.sh

.PHONY: lint
lint:
	@echo Linting...
	@GOPATH=${GOPATH} ./.ci/lint.sh

.PHONY: test
test: unit-tests

.PHONY: unit-tests
unit-tests:
	@echo Running unit tests...
	@go test ./... -cover -coverprofile=coverage.txt -covermode=atomic -race

.PHONY: all
all: check format lint security build test

.PHONY: ci
ci: install-tools ensure-generate-is-noop all

.PHONY: install-tools
install-tools:
	@curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b ${GOPATH}/bin 2.0.0
	@go install \
		golang.org/x/tools/cmd/goimports \
		k8s.io/code-generator/cmd/client-gen \
		k8s.io/kube-openapi/cmd/openapi-gen

.PHONY: install-prometheus-operator
install-prometheus-operator:
	@echo Installing Prometheus Operator bundle
	@kubectl apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/bundle.yaml
