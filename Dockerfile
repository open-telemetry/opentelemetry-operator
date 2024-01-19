# Get CA certificates from alpine package repo
FROM alpine:3.19 as certificates

RUN apk --no-cache add ca-certificates

######## Start a new stage from scratch #######
FROM scratch

ARG TARGETARCH
ARG BINARY_NAME=manager

WORKDIR /

# Copy the certs from Alpine
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary built on the host
COPY bin/${BINARY_NAME}_${TARGETARCH} main

USER 65532:65532

ENTRYPOINT ["/main"]
