FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:48fa5d8cda7fc00d270d8747c3eaa54ae196f0820d8540074a9c8c61d5e3056f

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
