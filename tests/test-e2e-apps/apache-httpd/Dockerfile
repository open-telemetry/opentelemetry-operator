
FROM docker.io/library/httpd:2.4@sha256:2a7deaaaf357261a1dffcb8fc725f4aaba2af95fbde1a40a68bca7cb0f03594e

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/