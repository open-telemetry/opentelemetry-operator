# To build one auto-instrumentation image for Python, please:
# - Ensure the packages are installed in the `/autoinstrumentation,{-musl}` directory. This is required as when instrumenting the pod,
#   one init container will be created to copy all the content in `/autoinstrumentation{,-musl}` directory to your app's container. Then
#   update the `PYTHONPATH` environment variable accordingly. To achieve this, you can mimic the one in `autoinstrumentation/python/Dockerfile`
#   by using multi-stage builds. In the first stage, install all the required packages in one custom directory with `pip install --target`.
#   Then in the second stage, copy the directory to `/autoinstrumentation{,-musl}`.
# - Ensure you have `opentelemetry-distro` and `opentelemetry-instrumentation` or your customized alternatives installed.
#   Those two packages are essential to Python auto-instrumentation.
# - Grant the necessary access to `/autoinstrumentation{,-musl}` directory. `chmod -R go+r /autoinstrumentation`
# - For auto-instrumentation by container injection, the Linux command cp is
#   used and must be available in the image.
ARG TARGETARCH
# Version of the opentelemetry-injector release to include, without the "v"
# prefix. There is no default; the canonical value lives in
# autoinstrumentation/injector_version.txt and must be passed in as a build-arg:
#   --build-arg injector_version=$(cat ../injector_version.txt)
ARG injector_version

FROM python:3.14@sha256:311ea5bb79f1a238ee9e38f8d5f09cb3b4b244575cf49e27cf365ea7e60f11d4 AS build

WORKDIR /operator-build

ADD requirements.txt .

RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM python:3.14-alpine@sha256:26730869004e2b9c4b9ad09cab8625e81d256d1ce97e72df5520e806b1709f92 AS build-musl

WORKDIR /operator-build

ADD requirements.txt .

RUN apk add gcc python3-dev musl-dev linux-headers
RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM busybox@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d AS base

COPY --from=build /operator-build/workspace /autoinstrumentation
COPY --from=build-musl /operator-build/workspace /autoinstrumentation-musl

RUN chmod -R go+r /autoinstrumentation
RUN chmod -R go+r /autoinstrumentation-musl

# The opentelemetry-injector activates the instrumentation by prepending a
# single directory to PYTHONPATH, so sitecustomize.py must be importable from
# the root of each installation.
RUN ln -s opentelemetry/instrumentation/auto_instrumentation/sitecustomize.py /autoinstrumentation/sitecustomize.py &&\
    ln -s opentelemetry/instrumentation/auto_instrumentation/sitecustomize.py /autoinstrumentation-musl/sitecustomize.py

# The opentelemetry-injector shared object activates the instrumentation via
# LD_PRELOAD when the operator.autoinstrumentation.injector feature gate is
# enabled. It is only published for amd64 and arm64, so the other
# architectures this image is built for (s390x, ppc64le) ship without it.
FROM base AS injector-amd64
ARG injector_version
ADD https://github.com/open-telemetry/opentelemetry-injector/releases/download/v$injector_version/libotelinject_amd64.so /libotelinject.so
RUN chmod go+r /libotelinject.so

FROM base AS injector-arm64
ARG injector_version
ADD https://github.com/open-telemetry/opentelemetry-injector/releases/download/v$injector_version/libotelinject_arm64.so /libotelinject.so
RUN chmod go+r /libotelinject.so

FROM base AS injector-s390x

FROM base AS injector-ppc64le

FROM injector-$TARGETARCH
