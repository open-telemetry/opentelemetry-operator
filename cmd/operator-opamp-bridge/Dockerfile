# Get CA certificates from the Alpine package repo
FROM alpine:3.18 as certificates

RUN apk --no-cache add ca-certificates

# Start a new stage from scratch
FROM scratch

ARG TARGETARCH

WORKDIR /root/

# Copy the certs
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary built on the host
COPY bin/opampbridge_${TARGETARCH} ./main
<<<<<<< HEAD
=======

# "nonroot"
USER 65532:65532
>>>>>>> b353f36e99a60a87d16f451ffda78715e7dcd91e

ENTRYPOINT ["./main"]
