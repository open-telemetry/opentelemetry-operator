# To build one auto-instrumentation image for php, please:
#  - Download your php auto-instrumentation artefacts to `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#    one init container will be created to copy the files to your app's container.
#  - Grant the necessary access to the files in the `/autoinstrumentation` directory.
#  - Following environment variables are injected to the application container to enable the auto-instrumentation.
#       PHP_INI_SCAN_DIR=:/otel-auto-instrumentation-php/php_ini_scan_dir
#       OTEL_PHP_AUTOLOAD_ENABLED=true
#  - For auto-instrumentation by container injection, the Linux command cp is
#    used and must be availabe in the image.

FROM php-cli as staging

ARG auto_instrumentation_version

WORKDIR /autoinstrumentation

RUN pecl install opentelemetry-$auto_instrumentation_version
RUN mkdir lib
RUN cp /usr/local/lib/php/extensions/no-debug-non-zts-20230831/opentelemetry.so lib/
RUN mkdir php_ini_scan_dir
COPY php_config_opentelemetry_extension.ini php_ini_scan_dir/php_config_opentelemetry_extension.ini

COPY --from=composer:latest /usr/bin/composer /usr/bin/composer
ADD composer.json .
RUN composer install

FROM busybox

COPY --from=staging /autoinstrumentation /autoinstrumentation

RUN chmod -R go+r /autoinstrumentation
