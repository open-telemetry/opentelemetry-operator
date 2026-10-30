# Get CA certificates from alpine package repo
FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS certificates

RUN apk --no-cache add 'ca-certificates>=20261020-r0'

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
