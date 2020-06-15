FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/opentelemetry-operator \
    USER_UID=1001 \
    USER_NAME=opentelemetry-operator

# install operator binary
COPY build/_output/bin/opentelemetry-operator ${OPERATOR}

ENTRYPOINT [${OPERATOR}]

USER ${USER_UID}
