FROM ruby:3-slim AS build

WORKDIR /operator-build

COPY . .

RUN bundle install --path vendor/bundle

FROM busybox

COPY --from=build /operator-build /autoinstrumentation

