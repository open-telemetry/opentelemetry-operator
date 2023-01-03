# Build the target allocator binary
FROM golang:1.19 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY allocation/ allocation/
COPY collector/ collector/
COPY config/ config/
COPY diff/ diff/
COPY prehook/ prehook/
COPY server/ server/
COPY target/ target/
COPY watcher/ watcher/

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
# Copy the pre-built binary file from the previous stage
COPY --from=builder /workspace/main .
USER 65532:65532

ENTRYPOINT ["/main"]
