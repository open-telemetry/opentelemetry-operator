
FROM docker.io/library/httpd:2.4@sha256:41163d514c02ac205161c1549b499bc11adfdeb6291735b433d706af7e64382a

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/