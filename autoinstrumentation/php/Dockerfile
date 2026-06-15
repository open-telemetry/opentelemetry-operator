ARG version=1.2.1

FROM composer:2 AS composer_build
COPY composer.json .
RUN composer install --ignore-platform-reqs

FROM php:8.1 AS build-non-zts-81
ARG version
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.1-zts AS build-zts-81
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.2 AS build-non-zts-82
ARG version
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.2-zts AS build-zts-82
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.3 AS build-non-zts-83
ARG version
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.3-zts AS build-zts-83
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.4 AS build-non-zts-84
ARG version
WORKDIR /build
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.4-zts AS build-zts-84
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.5 AS build-non-zts-85
ARG version
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.5-zts AS build-zts-85
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.1-alpine AS build-non-zts-musl-81
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.1-zts-alpine AS build-zts-musl-81
ARG version
WORKDIR /build/musl/zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.2-alpine AS build-non-zts-musl-82
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.2-zts-alpine AS build-zts-musl-82
ARG version
WORKDIR /build/musl/zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.3-alpine AS build-non-zts-musl-83
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.3-zts-alpine AS build-zts-musl-83
ARG version
WORKDIR /build/musl/zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.4.1-alpine AS build-non-zts-musl-84
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.4.1-zts-alpine AS build-zts-musl-84
ARG version
WORKDIR /build/musl/zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.5-alpine AS build-non-zts-musl-85
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.5-zts-alpine AS build-zts-musl-85
ARG version
WORKDIR /build/musl/zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .


FROM busybox@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d

COPY --from=build-non-zts-81 /build /autoinstrumentation/20210902
COPY --from=build-zts-81 /build /autoinstrumentation/20210902
COPY --from=build-non-zts-82 /build /autoinstrumentation/20220829
COPY --from=build-zts-82 /build /autoinstrumentation/20220829
COPY --from=build-non-zts-83 /build /autoinstrumentation/20230831
COPY --from=build-zts-83 /build /autoinstrumentation/20230831
COPY --from=build-non-zts-84 /build /autoinstrumentation/20240924
COPY --from=build-zts-84 /build /autoinstrumentation/20240924
COPY --from=build-non-zts-85 /build /autoinstrumentation/20250925
COPY --from=build-zts-85 /build /autoinstrumentation/20250925
COPY --from=build-non-zts-musl-81 /build /autoinstrumentation/20210902
COPY --from=build-zts-musl-81 /build /autoinstrumentation/20210902
COPY --from=build-non-zts-musl-82 /build /autoinstrumentation/20220829
COPY --from=build-zts-musl-82 /build /autoinstrumentation/20220829
COPY --from=build-non-zts-musl-83 /build /autoinstrumentation/20230831
COPY --from=build-zts-musl-83 /build /autoinstrumentation/20230831
COPY --from=build-non-zts-musl-84 /build /autoinstrumentation/20240924
COPY --from=build-zts-musl-84 /build /autoinstrumentation/20240924
COPY --from=build-non-zts-musl-85 /build /autoinstrumentation/20250925
COPY --from=build-zts-musl-85 /build /autoinstrumentation/20250925

COPY --from=composer_build /app/vendor /autoinstrumentation

COPY opentelemetry.ini /autoinstrumentation/opentelemetry.ini
COPY version.txt /autoinstrumentation/version.txt

RUN chmod -R go+r /autoinstrumentation
