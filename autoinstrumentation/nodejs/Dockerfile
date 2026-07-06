# To build one auto-instrumentation image for Node.js, please:
# - Ensure the packages are installed in the `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#   one init container will be created to copy all the content in `/autoinstrumentation` directory to your app's container. Then
#   update the `NODE_OPTIONS` environment variable accordingly. To achieve this, you can mimic the one in `autoinstrumentation/nodejs/Dockerfile`
#   by using multi-stage builds. In the first stage, install all the required packages in one custom directory.
#   Then in the second stage, copy the directory to `/autoinstrumentation`.
# - Ensure you have `@opentelemetry/api`, `@opentelemetry/auto-instrumentations-node`, and `@opentelemetry/sdk-node` or your customized
#   alternatives installed.
# - Grant the necessary access to `/autoinstrumentation` directory. `chmod -R go+r /autoinstrumentation`
# - For auto-instrumentation by container injection, the Linux command cp is
#   used and must be available in the image.
ARG TARGETARCH
ARG injector_version=0.9.2

FROM node:26@sha256:0473e7dc433a1310f436edee02aa79737ec78a4b345433ab0963d4a256f9ad85 AS build

WORKDIR /operator-build
COPY . .

RUN npm install

FROM busybox@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d AS base

COPY --from=build /operator-build/build/workspace /autoinstrumentation

RUN chmod -R go+r /autoinstrumentation

# Backward compatibility with old operator version
RUN cp -r /autoinstrumentation/node_modules/@opentelemetry/auto-instrumentations-node/build/src/. /autoinstrumentation
RUN ln /autoinstrumentation/register.js /autoinstrumentation/autoinstrumentation.js

# The opentelemetry-injector shared object activates the instrumentation via
# LD_PRELOAD when the operator.autoinstrumentation.injector feature gate is
# enabled. It is placed inside /autoinstrumentation so the init container
# copies it together with the instrumentation itself. It is only published
# for amd64 and arm64, so the other architectures this image is built for
# (s390x, ppc64le) ship without it.
FROM base AS injector-amd64
ARG injector_version
ADD https://github.com/open-telemetry/opentelemetry-injector/releases/download/v$injector_version/libotelinject_amd64.so /autoinstrumentation/libotelinject.so
RUN chmod go+r /autoinstrumentation/libotelinject.so

FROM base AS injector-arm64
ARG injector_version
ADD https://github.com/open-telemetry/opentelemetry-injector/releases/download/v$injector_version/libotelinject_arm64.so /autoinstrumentation/libotelinject.so
RUN chmod go+r /autoinstrumentation/libotelinject.so

FROM base AS injector-s390x

FROM base AS injector-ppc64le

FROM injector-$TARGETARCH
