
FROM docker.io/library/httpd:2.4@sha256:a30cd4cd57f3f3c3be9b2d16384d2488839996bb14c67df2b51f1ecbb3e584a1

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/