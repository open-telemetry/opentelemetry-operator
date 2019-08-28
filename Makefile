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

PACKAGES := $(shell go list ./cmd/... ./pkg/...)

.DEFAULT_GOAL := build

.PHONY: check
check:
	@echo Checking...
	@go fmt $(PACKAGES) > $(FMT_LOG)

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: generate
	@git diff -s --exit-code pkg/apis/opentelemetry/v1alpha1/zz_generated.*.go || (echo "Build failed: a model has been changed but the deep copy functions aren't up to date. Run 'make generate' and update your PR." && exit 1)

.PHONY: format
format:
	@echo Formatting code...
	@go fmt $(PACKAGES)

.PHONY: lint
lint:
	@echo Linting...
	@$(GOPATH)/bin/golint -set_exit_status=1 $(PACKAGES)

.PHONY: security
security:
	@echo Security...
	@$(GOPATH)/bin/gosec -quiet $(PACKAGES) 2>/dev/null

.PHONY: build
build: format
	@echo Building...
	@operator-sdk build ${BUILD_IMAGE} --go-build-args "-ldflags ${LD_FLAGS}"

.PHONY: run
run: crd
	@operator-sdk up local OPERATOR_NAME=$(OPERATOR_NAME) --go-ldflags "-X $(VERSION_PKG).Version=$(OPERATOR_VERSION) -X $(VERSION_PKG).BuildDate=$(VERSION_DATE)"

.PHONY: run-debug
run-debug: crd
	@operator-sdk up local --operator-flags "--log-level=debug" OPERATOR_NAME=$(OPERATOR_NAME)

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
test:
	@echo Running tests...

.PHONY: all
all: check format lint security build test

.PHONY: ci
ci: ensure-generate-is-noop all

.PHONY: scorecard
scorecard:
	@operator-sdk scorecard --cr-manifest deploy/examples/simplest.yaml --csv-path deploy/olm-catalog/jaeger.clusterserviceversion.yaml --init-timeout 30

.PHONY: install-tools
install-tools:
	@go get -u \
		golang.org/x/lint/golint \
		github.com/securego/gosec/cmd/gosec
