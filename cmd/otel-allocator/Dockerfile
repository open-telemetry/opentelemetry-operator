# Build the otel-allocator binary
FROM golang:1.20-alpine as builder

WORKDIR /app

RUN apk --no-cache add ca-certificates

# Copy go mod and sum files
COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

######## Start a new stage from scratch #######
FROM scratch

WORKDIR /root/

# Copy the certs from the builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/main .

ENTRYPOINT ["./main"]
