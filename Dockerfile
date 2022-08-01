# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY apis/ apis/
COPY controllers/ controllers/
COPY internal/ internal/
COPY pkg/ pkg/
COPY versions.txt versions.txt

ARG VERSION_PKG
ARG VERSION
ARG VERSION_DATE
ARG OTELCOL_VERSION
ARG TARGETALLOCATOR_VERSION
ARG AUTO_INSTRUMENTATION_JAVA_VERSION
ARG AUTO_INSTRUMENTATION_NODEJS_VERSION
ARG AUTO_INSTRUMENTATION_PYTHON_VERSION
ARG AUTO_INSTRUMENTATION_DOTNET_VERSION

# Build
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -ldflags="-X ${VERSION_PKG}.version=${VERSION} -X ${VERSION_PKG}.buildDate=${VERSION_DATE} -X ${VERSION_PKG}.otelCol=${OTELCOL_VERSION} -X ${VERSION_PKG}.targetAllocator=${TARGETALLOCATOR_VERSION} -X ${VERSION_PKG}.autoInstrumentationJava=${AUTO_INSTRUMENTATION_JAVA_VERSION} -X ${VERSION_PKG}.autoInstrumentationNodeJS=${AUTO_INSTRUMENTATION_NODEJS_VERSION} -X ${VERSION_PKG}.autoInstrumentationPython=${AUTO_INSTRUMENTATION_PYTHON_VERSION} -X ${VERSION_PKG}.autoInstrumentationDotNet=${AUTO_INSTRUMENTATION_DOTNET_VERSION}" -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
