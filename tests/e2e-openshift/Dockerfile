# The Dockerfile's resulting image is purpose-built for executing OpenTelemetry Operator e2e tests within the OpenShift release (https://github.com/openshift/release) using Prow CI. 

FROM golang:1.20

# Copy the repository files
COPY . /tmp/opentelemetry-operator

WORKDIR /tmp

# Install kuttl
RUN curl -L -o kuttl https://github.com/kudobuilder/kuttl/releases/download/v0.15.0/kubectl-kuttl_0.15.0_linux_x86_64 \
    && chmod +x kuttl \
    && mv kuttl /usr/local/bin/kuttl

# Install kubectl and oc
RUN curl -L -o oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest/openshift-client-linux.tar.gz \
    && tar -xvzf oc.tar.gz \
    && chmod +x kubectl oc \
    && mv oc kubectl /usr/local/bin/

# Set the working directory
WORKDIR /tmp/opentelemetry-operator
