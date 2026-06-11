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
FROM python:3.14@sha256:250e5c97be05e1eb2272fbdbd810dfd638f9012e1e6f65c99390ad3239943a08 AS build

WORKDIR /operator-build

ADD requirements.txt .

RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM python:3.14-alpine@sha256:5a824eb82cc75361f98611f3cfc5091ea33f10a6ccea4d4ebdabbc523b9a1614 AS build-musl

WORKDIR /operator-build

ADD requirements.txt .

RUN apk add gcc python3-dev musl-dev linux-headers
RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM busybox@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d

COPY --from=build /operator-build/workspace /autoinstrumentation
COPY --from=build-musl /operator-build/workspace /autoinstrumentation-musl

RUN chmod -R go+r /autoinstrumentation
RUN chmod -R go+r /autoinstrumentation-musl
