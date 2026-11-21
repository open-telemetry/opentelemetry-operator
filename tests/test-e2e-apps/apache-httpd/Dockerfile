
FROM docker.io/library/httpd:2.4.71@sha256:32e9403360ab2f1047523e72f256527fb9aee68b342abf6ad03f0d629157f3d9

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/