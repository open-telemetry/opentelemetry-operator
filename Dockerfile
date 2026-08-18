# Get CA certificates from alpine package repo
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS certificates

RUN apk --no-cache add ca-certificates

######## Start a new stage from scratch #######
FROM scratch

ARG TARGETARCH

WORKDIR /

# Copy the certs from Alpine
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary built on the host
COPY bin/manager_${TARGETARCH} manager

USER 65532:65532

ENTRYPOINT ["/manager"]
