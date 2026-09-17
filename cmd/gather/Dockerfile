FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:186a94b76e386782f576c9c49813b16dceb2ba63102af5a28405dcefec2806d0

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
