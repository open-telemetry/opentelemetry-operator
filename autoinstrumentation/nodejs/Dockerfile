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
#   used and must be availabe in the image.
FROM node:20 AS build

WORKDIR /operator-build
COPY . .

RUN npm install

FROM busybox

COPY --from=build /operator-build/build/workspace /autoinstrumentation

RUN chmod -R go+r /autoinstrumentation
