FROM docker.io/library/golang:1.27-alpine@sha256:4cb7ac979db5fcc41cae44b2227ba5ab8a51e8807f40d9ba4dee20a0ad960b5b AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -o otlp-sink main.go

FROM scratch
COPY --from=builder /app/otlp-sink .
ENTRYPOINT ["./otlp-sink"]
