FROM composer:2@sha256:b6dcd72ca98fa19d0e5b8476de179859704891fad06dd9e0b96833d4c430a5fc AS composer_build

FROM php:8.5@sha256:635a4b998025ab269194639a5549c6fdcd9b4f88ec8cd761b092de8653d40663

COPY --from=composer_build /usr/bin/composer /usr/local/bin/composer

RUN apt-get update && apt-get install -y git zlib1g-dev libzip-dev unzip && docker-php-ext-install zip

WORKDIR /app

COPY composer.json .
COPY index.php .

RUN composer install

CMD ["php", "-S" , "0.0.0.0:8080"]
