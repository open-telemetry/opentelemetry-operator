FROM docker.io/library/ruby:4.0.10-slim@sha256:67fb5dbef860bb2205a3c69799f7f1c4f5b01c943ae089de436bc266ea8c102e

WORKDIR /app

COPY Gemfile ./
RUN bundle install

COPY app.rb ./

ENV PORT=8080
# Outside production mode Sinatra rejects Host headers other than localhost, .test and IPs.
ENV APP_ENV=production

CMD ["bundle", "exec", "ruby", "app.rb"]
