FROM registry.access.redhat.com/ubi9-minimal:9.9-1794467051@sha256:baafce2d34783a28b4a472779f304e97e00f9d35c99553bc53b068b14344d9a9

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
