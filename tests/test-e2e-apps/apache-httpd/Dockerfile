
FROM docker.io/library/httpd:2.4@sha256:b7c88cc3708bab3cc33ed0bc45657165453ba34bc783ba30dd9dc33746d51597

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/