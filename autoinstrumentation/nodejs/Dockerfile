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
FROM node:26@sha256:f9a1756160a9e1c3dca456bc0b185bf8f2f112a6771c694df87288046f4306f1 AS build

WORKDIR /operator-build
COPY . .

RUN npm install

FROM busybox@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d

COPY --from=build /operator-build/build/workspace /autoinstrumentation

RUN chmod -R go+r /autoinstrumentation

# Backward compatibility with old operator version
RUN cp -r /autoinstrumentation/node_modules/@opentelemetry/auto-instrumentations-node/build/src/. /autoinstrumentation
RUN ln /autoinstrumentation/register.js /autoinstrumentation/autoinstrumentation.js
