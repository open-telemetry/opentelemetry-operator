
FROM docker.io/library/httpd:2.4.71@sha256:19310e146a08911733b7a7703ddff97b6d8d510d4bc1e38a252db934e7437447

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/