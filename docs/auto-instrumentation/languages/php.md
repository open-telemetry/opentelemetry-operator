# PHP auto-instrumentation

PHP auto-instrumentation also honors the following annotations.

```bash
instrumentation.opentelemetry.io/inject-php: "true"
instrumentation.opentelemetry.io/otel-php-auto-detect: "true" # for auto-detecting the platform, version and thread safety by cloning of user PHP container during detection.
instrumentation.opentelemetry.io/otel-php-auto-detect: "false" # for using the following annotation values for platform, version and thread safety, this is the default value and can be omitted
instrumentation.opentelemetry.io/otel-php-platform: "glibc" # for Linux glibc based PHP, this is the default value and can be omitted
instrumentation.opentelemetry.io/otel-php-platform: "musl" # for Linux musl based PHP
instrumentation.opentelemetry.io/otel-php-api-version: "20250925" # API version for PHP 8.5.x
instrumentation.opentelemetry.io/otel-php-api-version: "20240924" # API version for PHP 8.4.x
instrumentation.opentelemetry.io/otel-php-api-version: "20230831" # API version for PHP 8.3.x
instrumentation.opentelemetry.io/otel-php-api-version: "20220829" # API version for PHP 8.2.x
instrumentation.opentelemetry.io/otel-php-api-version: "20210902" # API version for PHP 8.1.x
instrumentation.opentelemetry.io/otel-php-thread-safety: "true" # for ZTS (Zend Thread Safety) PHP
instrumentation.opentelemetry.io/otel-php-thread-safety: "false" # for non-ZTS (non Zend Thread Safety) PHP, this is the default value and can be omitted
```
