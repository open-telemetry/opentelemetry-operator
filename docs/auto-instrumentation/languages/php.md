# PHP auto-instrumentation

PHP auto-instrumentation also honors the following annotations.

| Annotation                                                | Valid Values                                               | Default   |
|-----------------------------------------------------------|------------------------------------------------------------|-----------|
| `instrumentation.opentelemetry.io/inject-php`             | `true`, `false`                                            | —         |
| `instrumentation.opentelemetry.io/otel-php-api-version`   | `20250925`, `20240924`, `20230831`, `20220829`, `20210902` | —         |
| `instrumentation.opentelemetry.io/otel-php-platform`      | `glibc`, `musl`                                            | `glibc`   |
| `instrumentation.opentelemetry.io/otel-php-thread-safety` | `zts`, `non-zts`                                           | `non-zts` |

```bash
# Required annotation to enable PHP auto-instrumentation
instrumentation.opentelemetry.io/inject-php: "true"
instrumentation.opentelemetry.io/otel-php-api-version: "20250925" # API version for PHP 8.5.x
instrumentation.opentelemetry.io/otel-php-api-version: "20240924" # API version for PHP 8.4.x
instrumentation.opentelemetry.io/otel-php-api-version: "20230831" # API version for PHP 8.3.x
instrumentation.opentelemetry.io/otel-php-api-version: "20220829" # API version for PHP 8.2.x
instrumentation.opentelemetry.io/otel-php-api-version: "20210902" # API version for PHP 8.1.x
# Optional annotations to specify the platform and thread safety of the PHP runtime. If not specified, the default values are used.
instrumentation.opentelemetry.io/otel-php-platform: "glibc" # for Linux glibc based PHP, this is the default value and can be omitted
instrumentation.opentelemetry.io/otel-php-platform: "musl" # for Linux musl based PHP
instrumentation.opentelemetry.io/otel-php-thread-safety: "zts" # for ZTS (Zend Thread Safety) PHP
instrumentation.opentelemetry.io/otel-php-thread-safety: "non-zts" # for non-ZTS (non Zend Thread Safety) PHP, this is the default value and can be omitted
```
