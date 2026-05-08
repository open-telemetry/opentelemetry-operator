FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o simple-app ./

FROM alpine:3.18
RUN addgroup -S app && adduser -S -G app app
COPY --from=builder /src/simple-app /usr/local/bin/simple-app
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/simple-app"]
