# To build one auto-instrumentation image for Java, please:
#  - Download your customized `javaagent.jar` to `/javaagent.jar`. This is required as when instrumenting the pod,
#    one init container will be created to copy the jar to your app's container.
#  - Grant the necessary access to the jar. `chmod -R go+r /javaagent.jar`
#  - For auto-instrumentation by container injection, the Linux command cp is
#    used and must be availabe in the image.

# Stage 1: Build the cp-utility binary
FROM rust:1.69 as builder

WORKDIR /usr/src/cp-utility
COPY ./tools/cp-utility .

## TARGETARCH is defined by buildx
# https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETARCH

# Run validations and audit only on amd64 bacause it is faster and those two steps
# are only used to validate the source code and don't require anything that is
# architecture specific.

# Validations
## Validate formatting
RUN if [ $TARGETARCH = "amd64" ]; then rustup component add rustfmt && cargo fmt --check ; fi

## Audit dependencies
RUN if [ $TARGETARCH = "amd64" ]; then cargo install cargo-audit && cargo audit ; fi


# Cross-compile based on the target platform.
RUN if [ $TARGETARCH = "amd64" ]; then export ARCH="x86_64" ; \
    elif [ $TARGETARCH = "arm64" ]; then export ARCH="aarch64" ; \
    else false; \
    fi \
    && rustup target add ${ARCH}-unknown-linux-musl \
    && cargo test  --target ${ARCH}-unknown-linux-musl \
    && cargo install --target ${ARCH}-unknown-linux-musl --path . --root .

# Stage 2: Create distribution
FROM scratch

ARG version

COPY --from=builder /usr/src/cp-utility/bin/cp-utility /bin/cp

ADD --chmod=go+r https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v$version/opentelemetry-javaagent.jar /javaagent.jar
