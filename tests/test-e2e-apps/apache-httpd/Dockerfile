
FROM docker.io/library/httpd:2.4@sha256:03f858efb82c25cb0f9962946615cdcdaf26927cb971722149b563197cfd0fdd

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/