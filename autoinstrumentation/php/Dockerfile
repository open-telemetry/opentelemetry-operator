ARG version=1.2.1

FROM composer:2@sha256:1c62c57bb5228569034b7b4d1415b17ba6b731619f7de226eaa33ad1845785ec AS composer_build
COPY composer.json .
RUN composer install --ignore-platform-reqs

FROM php:8.1@sha256:76e563191d1ade120313a8736df24154d21da5155c0756f147c0b01bd19d9087 AS build-non-zts-81
ARG version
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.1-zts@sha256:113088c9c240ccfb16c45834cb1df50b2bc6f414638cd16a72ab7a5b03681329 AS build-zts-81
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.2@sha256:9277667c0fc298de473509dfed37adf969c97a0372338de990491b39bacf99a5 AS build-non-zts-82
ARG version
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.2-zts@sha256:f1ea309343a079d12536654de261e96e8b33d293270000fbbdf1cc799afebb12 AS build-zts-82
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.3@sha256:22f6151b15f7845352b6e08b85c602f7ea5ac0e52dc8462f2bd69b4d39d587e9 AS build-non-zts-83
ARG version
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.3-zts@sha256:38e3fd85f4551dfa79807f91655d13cc33ac059bcc4141010cc14de1253b7c84 AS build-zts-83
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.4@sha256:966621a53c8e75f062fad4e043ffec507cb793822aee3422110e0127fe53952d AS build-non-zts-84
ARG version
WORKDIR /build
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.4-zts@sha256:c3b3f84aa03f720545cd56741c76b908b5b73392531697676c202bc9fb6540b1 AS build-zts-84
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.5@sha256:1954ff5cd21f222c992b79d25e403b2600cec829678d5bb7076883f3a44c0d6e AS build-non-zts-85
ARG version
WORKDIR /build/glibc/non-zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.5-zts@sha256:53967f4bcf17cb33d82c594dec23e1edb0fd9ed8d3e0fcca10906c170f1ab0ee AS build-zts-85
ARG version
WORKDIR /build/glibc/zts
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.1-alpine@sha256:7949370448b0b4d9787776dc5968e0fd8d48763292344b5fbf21539441228a98 AS build-non-zts-musl-81
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.1-zts-alpine@sha256:f36160602456091e5a8f656a2f5c8e68435c36284856fa4116a46e00f38dc04b AS build-zts-musl-81
ARG version
WORKDIR /build/musl/zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.2-alpine@sha256:2b1502df0ae31b813e58b8eef346c48ec21d743e8d0e42abc40331aa7783778e AS build-non-zts-musl-82
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.2-zts-alpine@sha256:2a085af54283e68ebb87fb01ebf97b4d30b8fe5b3cdefabdf7e51e67217e74ed AS build-zts-musl-82
ARG version
WORKDIR /build/musl/zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.3-alpine@sha256:a1986e9f5180ee8f1bf96aebbf832fa5fe5f077bdc9176c2a2365e32243118f0 AS build-non-zts-musl-83
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.3-zts-alpine@sha256:6aa7082189947f11cb065bee7eee19d9eee0371eb3f357d3fe92d5b3db1f5c94 AS build-zts-musl-83
ARG version
WORKDIR /build/musl/zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.4.1-alpine@sha256:fbaae9d17cbcb784a92be5e6de7b39848e306b221ef0edb218c832418797c8f7 AS build-non-zts-musl-84
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.4.1-zts-alpine@sha256:b3b4e31d0301d1dbb8c920f4efe3cd48df2f0a07cbc532c43c6635fb5ad76c24 AS build-zts-musl-84
ARG version
WORKDIR /build/musl/zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-zts-*/opentelemetry.so .

FROM php:8.5-alpine@sha256:f6975f0b54f3138826ec673961f44375d54f448d3bbfc8a2a8c58228aeeaaba1 AS build-non-zts-musl-85
ARG version
WORKDIR /build/musl/non-zts
RUN apk add autoconf build-base
RUN pecl install opentelemetry-$version
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so .

FROM php:8.5-zts-alpine@sha256:6b2a31dfcf302b0b5fc2f6150671579400dccb21906eb09535251c02fb33d19e AS build-zts-musl-85
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
