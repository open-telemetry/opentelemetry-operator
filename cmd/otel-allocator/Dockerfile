# Build the target allocator binary
FROM golang:1.19 as builder

WORKDIR /app

# copy operator and allocator code
COPY . .

WORKDIR ./cmd/otel-allocator

RUN go mod download

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

######## Start a new stage from scratch #######
FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/cmd/otel-allocator/main .

ENTRYPOINT ["./main"]
