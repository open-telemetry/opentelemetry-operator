FROM docker.io/library/ruby:4.0.8-slim@sha256:55d1710ff7d36fc88ce792c45b5e04145590c29ba220fb5f1e412b90766b19bf

WORKDIR /app

COPY Gemfile ./
RUN bundle install

COPY app.rb ./

ENV PORT=8080
# Outside production mode Sinatra rejects Host headers other than localhost, .test and IPs.
ENV APP_ENV=production

CMD ["bundle", "exec", "ruby", "app.rb"]
