FROM golang:1.23-alpine as builder

COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -o app main.go

FROM scratch
COPY --from=builder /go/app .
ENTRYPOINT ["./app"]
