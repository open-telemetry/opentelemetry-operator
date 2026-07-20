
FROM docker.io/library/httpd:2.4@sha256:305fd8326a27a137fedb8900f26375699ab233cc37793f35cf5d7c65671fb252

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/