# Get CA certificates from the Alpine package repo
FROM alpine:3.24@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4 AS certificates

RUN apk --no-cache add ca-certificates

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
