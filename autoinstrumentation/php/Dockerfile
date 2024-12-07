# To build one auto-instrumentation image for php, please:
#  - Download your php auto-instrumentation artefacts to `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#    one init container will be created to copy the files to your app's container.
#  - Grant the necessary access to the files in the `/autoinstrumentation` directory.
#  - Following environment variables are injected to the application container to enable the auto-instrumentation.
#       PHP_INI_SCAN_DIR=:/otel-auto-instrumentation-php/php_ini_scan_dir
#       OTEL_PHP_AUTOLOAD_ENABLED=true
#  - For auto-instrumentation by container injection, the Linux command cp is
#    used and must be availabe in the image.

FROM busybox

ARG SUB_DIR_WITH_FILES_FOR_DOCKER_IMAGE

COPY ${SUB_DIR_WITH_FILES_FOR_DOCKER_IMAGE} /autoinstrumentation/
