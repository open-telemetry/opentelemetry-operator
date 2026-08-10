FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:57c8151c51445a07e503dab9dc9211dc3cdeac9d45ed81a10954b7d770659b3b

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
