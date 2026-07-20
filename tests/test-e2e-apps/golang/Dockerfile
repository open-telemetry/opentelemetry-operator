# Stage 1: Build the Go application
FROM docker.io/library/golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

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
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/rolldice main.go

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
