
FROM docker.io/library/httpd:2.4.71@sha256:eef26a140e2817b6d9fa9b9eba883eb79b5a40032ca988503c8f7e40f90d0959

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/