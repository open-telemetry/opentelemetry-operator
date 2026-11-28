FROM composer:2.11.3@sha256:5b2137f112f939a92788ceb5e55f54859d9dbf467a44037cf7760bdda949f365 AS composer_build

FROM php:8.6.0@sha256:9bc69093cdff73d2dcf12bc34f4c308b8496a27dc3e6a4d0c332877289c6bf13

COPY --from=composer_build /usr/bin/composer /usr/local/bin/composer

RUN apt-get update && apt-get install -y git zlib1g-dev libzip-dev unzip && docker-php-ext-install zip

WORKDIR /app

COPY composer.json .
COPY index.php .

RUN composer install

CMD ["php", "-S" , "0.0.0.0:8080"]
