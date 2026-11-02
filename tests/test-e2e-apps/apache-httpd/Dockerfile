
FROM docker.io/library/httpd:2.4@sha256:c6b7745aa371570cc8460a9f08c39989583d4aa7bc8e3ed8bd77a492803ea704

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/