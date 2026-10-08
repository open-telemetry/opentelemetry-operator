FROM composer:2@sha256:9715c7f69044da2a212a5fbde29ee7da24e364d426560ae6367b060236f847d7 AS composer_build

FROM php:8.4@sha256:1d2cf2b7d715b5b3aa959f0150ccbcc7b71e0181b637e443abf40a7fb56bf79a

COPY --from=composer_build /usr/bin/composer /usr/local/bin/composer

RUN apt-get update && apt-get install -y git zlib1g-dev libzip-dev unzip && docker-php-ext-install zip

WORKDIR /app

COPY composer.json .
COPY index.php .

RUN composer install

CMD ["php", "-S" , "0.0.0.0:8080"]
