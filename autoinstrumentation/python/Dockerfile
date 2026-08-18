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
FROM python:3.14@sha256:4fad23465a06cc5149a541fbec6f87e234a64dc0550f6bfdd2d290d8f03240df AS build

WORKDIR /operator-build

ADD requirements.txt .

RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM python:3.14-alpine@sha256:05b2b8b732ecd268fee8727a369f936f022d1321b59befd13c30ede22769dcdc AS build-musl

WORKDIR /operator-build

ADD requirements.txt .

RUN apk add gcc python3-dev musl-dev linux-headers
RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM busybox@sha256:dc2d74b28e4cf8984fa52af1f39bc7c3d9c73760b41a74d629f5d11b1ab28616

COPY --from=build /operator-build/workspace /autoinstrumentation
COPY --from=build-musl /operator-build/workspace /autoinstrumentation-musl

RUN chmod -R go+r /autoinstrumentation
RUN chmod -R go+r /autoinstrumentation-musl
