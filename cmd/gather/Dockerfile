FROM registry.access.redhat.com/ubi9-minimal:9.8@sha256:44bc70ef6e6ea9a70e353be97f4722e10358d09fbb9494ca943b2a641049690e

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
