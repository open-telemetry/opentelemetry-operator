FROM composer:2@sha256:1c62c57bb5228569034b7b4d1415b17ba6b731619f7de226eaa33ad1845785ec AS composer_build

FROM php:8.4@sha256:966621a53c8e75f062fad4e043ffec507cb793822aee3422110e0127fe53952d

COPY --from=composer_build /usr/bin/composer /usr/local/bin/composer

RUN apt update && apt install -y git libzip-dev zip && docker-php-ext-install zip

WORKDIR /app

COPY composer.json .
COPY index.php .

RUN composer install

CMD ["php", "-S" , "0.0.0.0:8080"]
