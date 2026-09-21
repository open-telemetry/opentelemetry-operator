FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:984df0a2b8d9011d419b0ad260b25f6b74b7ce69ab0595b75522e30f8849f27f

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
