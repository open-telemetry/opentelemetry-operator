FROM docker.io/library/ruby:3.3-slim@sha256:79f7a07363931fde1a5b312dee281fd62ddf56c33bdba0c2622af9b4621182fb

WORKDIR /app

COPY Gemfile ./
RUN bundle install

COPY app.rb ./

ENV PORT=8080

CMD ["bundle", "exec", "ruby", "app.rb"]
