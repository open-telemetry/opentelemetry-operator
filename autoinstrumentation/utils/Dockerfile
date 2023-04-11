# This utils image is intended to be used as a base image for OpenTelemetry Operator
# autoinstrumentation images. The utils will allow autoinstrumentation packages to be
# copied (via a go based cp command) from the init container to the final destination volume.
FROM golang as utils
WORKDIR /utils/
COPY cp.go .

# add a slim helper executable
# -s omits the symbol table and debug information
# -w omits the DWARF symbol table
RUN go build -ldflags "-s -w" -o /utils/bin/cp cp.go
