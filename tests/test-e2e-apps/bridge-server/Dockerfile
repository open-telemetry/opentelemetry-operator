FROM docker.io/library/golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -o bridge-server main.go

FROM scratch
COPY --from=builder /app/bridge-server .
ENTRYPOINT ["./bridge-server"]
