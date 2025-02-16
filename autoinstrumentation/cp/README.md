# cp

This project allows you to take a file and copy to a new location on disk with the 0400 permission mask (read-only).

# Install

## As binary

```bash
go install github.com/otel-warez/cp@latest
```

## As a docker image

```
docker pull ghcr.io/otel-warez/cp:latest
```

This image is built from scratch and will not be useful on its own, but you can use it as a layer. Here is an example:

```
FROM ghcr.io/otel-warez/cp:latest AS cp

FROM scratch AS final
COPY --from=cp /cp /usr/bin/cp
...
```