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
FROM node:26@sha256:a723b54c35a76e947095a20a67d39585bb09c862e6b1adeb8a9f518f95e34fb0 AS build

WORKDIR /operator-build
COPY . .

RUN npm install

FROM busybox@sha256:fd7dc98638c8e305f4dc34e979f1c0fdfdcaeb0fbf8fcff77ae834b6da3d7e6e

COPY --from=build /operator-build/build/workspace /autoinstrumentation

RUN chmod -R go+r /autoinstrumentation

# Backward compatibility with old operator version
RUN cp -r /autoinstrumentation/node_modules/@opentelemetry/auto-instrumentations-node/build/src/. /autoinstrumentation
RUN ln /autoinstrumentation/register.js /autoinstrumentation/autoinstrumentation.js
