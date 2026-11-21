FROM docker.io/library/ruby:4.0.10-slim@sha256:a4b3e62bfe56bd267a3aeacc1b89aea96be919faa0c7d77c6346e2ac14f6c175

WORKDIR /app

COPY Gemfile ./
RUN bundle install

COPY app.rb ./

ENV PORT=8080
# Outside production mode Sinatra rejects Host headers other than localhost, .test and IPs.
ENV APP_ENV=production

CMD ["bundle", "exec", "ruby", "app.rb"]
