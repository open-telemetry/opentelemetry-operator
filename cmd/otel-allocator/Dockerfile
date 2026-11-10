# Get CA certificates from the Alpine package repo
FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS certificates

RUN apk --no-cache add 'ca-certificates>=20261020-r0'

# Start a new stage from scratch
FROM scratch

ARG TARGETARCH

WORKDIR /

# Copy the certs
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary built on the host
COPY bin/targetallocator_${TARGETARCH} ./main

USER 65532:65532

ENTRYPOINT ["./main"]
