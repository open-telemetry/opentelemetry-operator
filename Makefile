OPERATOR_NAME ?= opentelemetry-operator
OPERATOR_VERSION ?= "$(shell git describe --tags)"
VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

GO_FLAGS ?= GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on
KUBERNETES_CONFIG ?= "$(HOME)/.kube/config"
WATCH_NAMESPACE ?= ""
BIN_DIR ?= "build/_output/bin"

NAMESPACE ?= "$(USER)"
BUILD_IMAGE ?= "$(NAMESPACE)/$(OPERATOR_NAME):latest"
OUTPUT_BINARY ?= "$(BIN_DIR)/$(OPERATOR_NAME)"
VERSION_PKG ?= "github.com/open-telemetry/opentelemetry-operator/version"
LD_FLAGS ?= "-X=$(VERSION_PKG).Version=$(OPERATOR_VERSION)"

PACKAGES := $(shell go list ./cmd/... ./pkg/... ./version/...)

.DEFAULT_GOAL := build

.PHONY: discard-go-mod-changes
discard-go-mod-changes:
	@# 'go list' will update go.mod/go.sum and there's no way to prevent it (not even with -mod=readonly)
	@git checkout -- go.mod go.sum

.PHONY: check
check: format ensure-generate-is-noop discard-go-mod-changes
	@echo Checking...
	@git diff -s --exit-code . || (echo "Build failed: one or more source files aren't properly formatted. Run 'make format' and update your PR." && exit 1)

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: generate
	@git diff -s --exit-code pkg/apis/opentelemetry/v1alpha1/zz_generated.*.go || (echo "Build failed: a model has been changed but the deep copy functions aren't up to date. Run 'make generate' and update your PR." && exit 1)

.PHONY: format
format:
	@echo Formatting code...
	@goimports -w -local "github.com/open-telemetry/opentelemetry-operator" .

.PHONY: security
security:
	@echo Security...
	@gosec -quiet $(PACKAGES) 2>/dev/null

.PHONY: build
build: format
	@echo Building...
	@operator-sdk build ${BUILD_IMAGE} --go-build-args "-ldflags ${LD_FLAGS}"

.PHONY: run
run: crd
	@OPERATOR_NAME=$(OPERATOR_NAME) operator-sdk up local --go-ldflags "-X $(VERSION_PKG).Version=$(OPERATOR_VERSION) -X $(VERSION_PKG).BuildDate=$(VERSION_DATE)"

.PHONY: run-debug
run-debug: crd
	@OPERATOR_NAME=$(OPERATOR_NAME) operator-sdk up local --operator-flags "--zap-level=2"

.PHONY: clean
clean:
	@echo Cleaning...

.PHONY: crd
crd:
	@kubectl create -f deploy/crds/opentelemetry_v1alpha1_opentelemetryservice_crd.yaml 2>&1 | grep -v "already exists" || true

.PHONY: generate
generate:
	@operator-sdk generate k8s
	@operator-sdk generate openapi

.PHONY: test
test: unit-tests

.PHONY: unit-tests
unit-tests:
	@echo Running unit tests...
	@go test $(PACKAGES) -cover -coverprofile=coverage.txt -covermode=atomic -race

.PHONY: all
all: check format security build test

.PHONY: ci
ci: install-tools ensure-generate-is-noop all

.PHONY: install-tools
install-tools:
	@go get -u \
		github.com/securego/gosec/cmd/gosec \
		golang.org/x/tools/cmd/goimports
