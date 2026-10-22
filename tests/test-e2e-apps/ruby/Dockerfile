FROM docker.io/library/ruby:4.0-slim@sha256:ba6a00593739f9cb66de80ad2e395a93a52a13635867ee32918ba0ec083b1ff6

WORKDIR /app

COPY Gemfile ./
RUN bundle install

COPY app.rb ./

ENV PORT=8080

CMD ["bundle", "exec", "ruby", "app.rb"]
