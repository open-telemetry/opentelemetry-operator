# The Dockerfile's resulting image is purpose-built for executing OpenTelemetry Operator e2e tests within the OpenShift release (https://github.com/openshift/release) using Prow CI.

FROM docker.io/library/golang:1.26@sha256:68cb6d68bed024785b69195b89af7ac7a444f27791435f98647edff595aa0479

# Copy the repository files
COPY . /tmp/opentelemetry-operator

WORKDIR /tmp

# Set the Go path and Go cache environment variables
ENV GOPATH=/tmp/go
ENV GOBIN=/tmp/go/bin
ENV GOCACHE=/tmp/.cache/go-build
ENV PATH=$PATH:$GOBIN
ENV CHAINSAW_VERSION=v0.2.15

# Create the /tmp/go/bin and build cache directories, and grant read and write permissions to all users
RUN mkdir -p /tmp/go/bin $GOCACHE \
    && chmod -R 777 /tmp/go/bin $GOPATH $GOCACHE

# Install dependencies required by test cases and debugging
RUN apt-get update && apt-get install -y jq vim libreadline-dev unzip

# Install chainsaw
RUN curl -L -o chainsaw.tar.gz https://github.com/kyverno/chainsaw/releases/download/${CHAINSAW_VERSION}/chainsaw_linux_amd64.tar.gz \
    && tar -xvzf chainsaw.tar.gz \
    && chmod +x chainsaw \
    && mv chainsaw /usr/local/bin/

# Install kubectl and oc
RUN curl -LO https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest/openshift-client-linux.tar.gz \
    && tar -xzf openshift-client-linux.tar.gz \
    && chmod +x oc kubectl \
    && mv oc kubectl /usr/local/bin/

# Install the latest version of logcli
RUN curl -LO https://github.com/grafana/loki/releases/latest/download/logcli-linux-amd64.zip \
    && unzip logcli-linux-amd64.zip \
    && chmod +x logcli-linux-amd64 \
    && mv logcli-linux-amd64 /usr/local/bin/logcli

# Install yq (for YAML processing in TLS profile tests)
RUN curl -L -o /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 \
    && chmod +x /usr/local/bin/yq

# Install AWS CLI
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" \
    && unzip awscliv2.zip \
    && ./aws/install

# Install Claude Code CLI via signed apt repository
RUN install -d -m 0755 /etc/apt/keyrings \
    && curl -fsSL https://downloads.claude.ai/keys/claude-code.asc \
         -o /etc/apt/keyrings/claude-code.asc \
    && echo "deb [signed-by=/etc/apt/keyrings/claude-code.asc] https://downloads.claude.ai/claude-code/apt/stable stable main" \
         > /etc/apt/sources.list.d/claude-code.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends claude-code \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /tmp/opentelemetry-operator