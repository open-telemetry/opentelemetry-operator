
FROM docker.io/library/httpd:2.4@sha256:454942557d44332cd7e4ae415e7ec9b4dbbf01b4a042491f423fcaf10a32ce9f

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/