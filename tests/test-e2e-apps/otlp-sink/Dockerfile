FROM docker.io/library/golang:1.27-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -o otlp-sink main.go

FROM scratch
COPY --from=builder /app/otlp-sink .
ENTRYPOINT ["./otlp-sink"]
