FROM composer:2.10.2@sha256:d020706319701a44468968321dccd0fce6620190159a7a9ec195d78e6e971c71 AS composer_build

FROM php:8.5.12@sha256:acaa95199b53a1d7c346dddc7bcfc367ef3bd21ab832bd178eac7f043ed5396c

COPY --from=composer_build /usr/bin/composer /usr/local/bin/composer

RUN apt-get update && apt-get install -y git zlib1g-dev libzip-dev unzip && docker-php-ext-install zip

WORKDIR /app

COPY composer.json .
COPY index.php .

RUN composer install

CMD ["php", "-S" , "0.0.0.0:8080"]
