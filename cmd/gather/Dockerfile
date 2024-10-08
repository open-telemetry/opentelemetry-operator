FROM registry.access.redhat.com/ubi9-minimal:9.2

RUN INSTALL_PKGS=" \
  rsync \
  tar \
  " && \
  microdnf install -y $INSTALL_PKGS && \
  microdnf clean all
WORKDIR /

ARG TARGETARCH
COPY bin/must-gather_${TARGETARCH} /usr/bin/must-gather

USER 65532:65532

ENTRYPOINT ["/usr/bin/must-gather"]
