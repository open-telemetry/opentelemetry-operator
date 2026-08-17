FROM docker.io/library/golang:1.26-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -o bridge-server main.go

FROM scratch
COPY --from=builder /app/bridge-server .
ENTRYPOINT ["./bridge-server"]
