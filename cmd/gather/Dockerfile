FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:062c52ff973065752b0965787649db2bcf551a6c727a00e95a3eb42cebadbdab

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
