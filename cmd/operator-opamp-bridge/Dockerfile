# Get CA certificates from the Alpine package repo
FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS certificates

RUN apk --no-cache add ca-certificates

# Start a new stage from scratch
FROM scratch

ARG TARGETARCH

WORKDIR /

# Copy the certs
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary built on the host
COPY bin/opampbridge_${TARGETARCH} /opampbridge

# "nonroot"
USER 65532:65532
ENTRYPOINT ["/opampbridge"]
