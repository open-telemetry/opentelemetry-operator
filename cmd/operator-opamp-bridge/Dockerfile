# Get CA certificates from the Alpine package repo
FROM alpine:3.19 as certificates

RUN apk --no-cache add ca-certificates

# Start a new stage from scratch
FROM scratch

ARG TARGETARCH

WORKDIR /root/

# Copy the certs
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary built on the host
COPY bin/opampbridge_${TARGETARCH} ./main

# "nonroot"
USER 65532:65532
ENTRYPOINT ["./main"]
