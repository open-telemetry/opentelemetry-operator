FROM composer:2.11.1@sha256:5190d9b445add2fa26090108273cb1ef1e1af98175201c737b85b3c4f90fbf7c AS composer_build

FROM php:8.5.12@sha256:9236fd373d3bcda0659cf94838a744bd79791419292958963f8c6b34a0071554

COPY --from=composer_build /usr/bin/composer /usr/local/bin/composer

RUN apt-get update && apt-get install -y git zlib1g-dev libzip-dev unzip && docker-php-ext-install zip

WORKDIR /app

COPY composer.json .
COPY index.php .

RUN composer install

CMD ["php", "-S" , "0.0.0.0:8080"]
