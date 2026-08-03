FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:17fd831ced9434de0a984d60b3fbe61008308261ba98bbc348d6fbdef05fa7c0

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
