# Stage 1: Build the Go application
FROM docker.io/library/golang:1.27.1-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS builder

# Set the working directory inside the container for the build stage.
WORKDIR /app

# Copy the Go source file into the working directory inside the container.
COPY main.go .

# Build the Go application.
# - CGO_ENABLED=0: Disables CGO, which is necessary to build a statically linked binary
#   that can run in a minimal 'scratch' image without external C libraries.
# - GOOS=linux: Explicitly sets the target operating system to Linux, which is standard for containers.
# -o /app/rolldice: Specifies the output path and name for the compiled executable inside this builder stage.
# main.go: The source file to compile.
# Do not strip the binary (no -ldflags="-w -s"): the Go auto-instrumentation agent
# needs the ELF symbol table to locate functions in Go >= 1.26 binaries and falls
# back to DWARF for struct offsets when the Go version is newer than its cached
# offsets. Stripping either makes the instrumentation-go e2e test fail.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/rolldice main.go

# Stage 2: Create the final minimal production image
FROM scratch

# Set the working directory for the final image.
WORKDIR /

# Copy only the compiled binary from the 'builder' stage (/app/rolldice)
# into the root directory of the final 'scratch' image.
COPY --from=builder /app/rolldice /rolldice

# Set the command to run when the container starts.
# This executes the compiled Go binary we copied into the image.
ENTRYPOINT ["/rolldice"]
