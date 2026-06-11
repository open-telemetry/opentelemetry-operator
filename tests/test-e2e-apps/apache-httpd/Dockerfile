
FROM docker.io/library/httpd:2.4@sha256:bda12def1dc3b718a5314b4b26ca84927fdb3905091f11200d581186f240e4f6

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/