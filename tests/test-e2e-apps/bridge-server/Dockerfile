FROM docker.io/library/golang:1.25-alpine@sha256:8d22e29d960bc50cd025d93d5b7c7d220b1ee9aa7a239b3c8f55a57e987e8d45 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -o bridge-server main.go

FROM scratch
COPY --from=builder /app/bridge-server .
ENTRYPOINT ["./bridge-server"]
