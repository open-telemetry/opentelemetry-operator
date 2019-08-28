FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/opentelemetry-operator \
    USER_UID=1001 \
    USER_NAME=opentelemetry-operator

# install operator binary
COPY build/_output/bin/opentelemetry-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
