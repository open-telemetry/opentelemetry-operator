FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:1d7c5517a4a1a8e2688620b39ee980e82505ca1ab7ae5541b5463120ae9b3897

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
