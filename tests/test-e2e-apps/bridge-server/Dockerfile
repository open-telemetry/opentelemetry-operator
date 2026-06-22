FROM docker.io/library/golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -o bridge-server main.go

FROM scratch
COPY --from=builder /app/bridge-server .
ENTRYPOINT ["./bridge-server"]
