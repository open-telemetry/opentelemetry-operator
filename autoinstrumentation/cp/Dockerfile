# This cp image is intended to be used as a base image for OpenTelemetry Operator
# autoinstrumentation images. The cp will allow autoinstrumentation packages to be
# copied (via a go based cp command) from the init container to the final destination volume.
FROM golang as builder
WORKDIR /builder/
COPY cp.go .

# Create a slim cp executable
# -s omits the symbol table and debug information
# -w omits the DWARF symbol table
RUN go build -ldflags "-s -w" -o /builder/bin/cp cp.go

FROM scratch AS final
COPY --from=builder /builder/ /
