FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:8eb2830d0936237fc13a1f2f7e45aecf90d69043380ad167fad0343632937f41

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
