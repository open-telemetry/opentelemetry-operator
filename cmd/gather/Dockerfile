FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:dfa0eb04abc59221544e74862432496add1f491f915c9b531880f91affc0e12c

RUN INSTALL_PKGS=" \
  rsync \
  tar \
  " && \
  microdnf install -y $INSTALL_PKGS && \
  microdnf clean all
WORKDIR /

ARG TARGETARCH
COPY bin/must-gather_${TARGETARCH} /usr/bin/gather

USER 65532:65532

ENTRYPOINT ["/usr/bin/gather"]
