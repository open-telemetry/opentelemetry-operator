FROM docker.io/library/ruby:3.4-slim@sha256:c12225b6a73743f943530d9e610f4ae78e0dbcffdcf797b01522a55f7d25acb3

WORKDIR /app

COPY Gemfile ./
RUN bundle install

COPY app.rb ./

ENV PORT=8080

CMD ["bundle", "exec", "ruby", "app.rb"]
