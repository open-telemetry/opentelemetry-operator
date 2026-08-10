
FROM docker.io/library/httpd:2.4@sha256:2920ed8587277d6aa8ea785e143e970835057123dc7bf1199d102c60c80a73bb

RUN sed -i "s#Listen 80#Listen 8080#g" /usr/local/apache2/conf/httpd.conf
RUN chmod 777 -R /usr/local/apache2/