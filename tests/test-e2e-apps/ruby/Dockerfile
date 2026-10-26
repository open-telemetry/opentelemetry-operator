FROM docker.io/library/ruby:4.0-slim@sha256:3251898a6b7ff1275a0dc279d401889c6c743c1fb3292fa3ce1a2e12fc2cd31d

WORKDIR /app

COPY Gemfile ./
RUN bundle install

COPY app.rb ./

ENV PORT=8080
# Outside production mode Sinatra rejects Host headers other than localhost, .test and IPs.
ENV APP_ENV=production

CMD ["bundle", "exec", "ruby", "app.rb"]
