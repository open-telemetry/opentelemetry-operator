FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:692953368d8e630f40a3c0a6135163f8824fdafc26e0400b9a6c8d7fac850366

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
