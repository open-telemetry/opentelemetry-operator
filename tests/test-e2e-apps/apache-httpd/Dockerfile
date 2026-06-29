
FROM docker.io/library/httpd:2.4@sha256:393435ee1a31437adeb1f03c134224c9ce5fa5f527e8c0cb9bea576b0d6fc742

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/