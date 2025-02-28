# To build one auto-instrumentation image for Ruby, please:
# - Ensure the packages are installed in the `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#   one init container will be created to copy all the content in `/autoinstrumentation` directory to your app's container. Then
#   update the `RUBYOPT` environment variable accordingly. To achieve this, you can mimic the one in `autoinstrumentation/ruby/Dockerfile`
#   by using multi-stage builds. In the first stage, install all the required packages in one custom directory.
#   Then in the second stage, copy the directory to `/autoinstrumentation`.
# - Ensure you have `opentelemetry-sdk`, `opentelemetry-instrumentation-all`, and `opentelemetry-exporter-otlp` or your customized
#   alternatives installed.
# - Grant the necessary access to `/autoinstrumentation` directory. `chmod -R go+r /autoinstrumentation`
# - For auto-instrumentation by container injection, the Linux command cp is
#   used and must be availabe in the image.
FROM ruby:3.3-slim AS build

ARG PACKAGES="\
    build-essential \
    "

# Install packages
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends ${PACKAGES} && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /ruby/instrumentation

COPY . .

RUN gem install opentelemetry-api -v '1.4.0' --install-dir ./ && \
    gem install opentelemetry-exporter-otlp -v '0.29.1' --install-dir ./ && \
    gem install opentelemetry-helpers-mysql -v '0.2.0' --install-dir ./ && \
    gem install opentelemetry-helpers-sql-obfuscation -v '0.3.0' --install-dir ./ && \
    gem install opentelemetry-instrumentation-all -v '0.72.0' --install-dir ./ && \
    gem install opentelemetry-resource-detector-azure -v '0.2.0' --install-dir ./ && \
    gem install opentelemetry-resource-detector-container -v '0.2.0' --install-dir ./ && \
    gem install opentelemetry-resource-detector-google_cloud_platform -v '0.2.0' --install-dir ./ && \
    gem install opentelemetry-sdk -v '1.6.0' --install-dir ./

FROM busybox

# copy all gems from global install to autoinstrumentation
COPY --from=build /ruby/instrumentation/ /autoinstrumentation

RUN chmod -R go+r /autoinstrumentation
