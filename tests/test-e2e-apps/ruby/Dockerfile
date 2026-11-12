FROM docker.io/library/ruby:4.0.9-slim@sha256:575fcf0c168d396a6fa9fba87ea1e390ceb4904af6c3fe2bbfdf4824224628c3

WORKDIR /app

COPY Gemfile ./
RUN bundle install

COPY app.rb ./

ENV PORT=8080
# Outside production mode Sinatra rejects Host headers other than localhost, .test and IPs.
ENV APP_ENV=production

CMD ["bundle", "exec", "ruby", "app.rb"]
