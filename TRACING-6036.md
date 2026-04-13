# TRACING-6036: Autoinstrumentation webhook volumeMount corruption

## Bug

When a container's `VolumeMounts` slice has spare backing-array capacity (e.g. after Istio sidecar injection), the Apache HTTPD and Nginx clone init containers get corrupted volume mounts. The `cp` command fails with:

```
cp: target '/opt/opentelemetry-webserver/source-conf': No such file or directory
```

## Root cause

Go slice aliasing via `append`. In `apachehttpd.go:79` and `nginx.go:95`:

```go
VolumeMounts: append(container.VolumeMounts, corev1.VolumeMount{...}),
```

This returns a slice sharing the same backing array as `container.VolumeMounts` when `len < cap`. Later mutations to `container.VolumeMounts` (lines 106-113) overwrite the clone's config mount through the shared array.

Introduced in commit `a50c60ed` which refactored from `container.DeepCopy()` to constructing the container from scratch, but used `append` on the original slice instead of copying first.

## Fix

Replace `append` with `slices.Concat` in both files — always allocates a fresh slice:

```go
VolumeMounts: slices.Concat(container.VolumeMounts, []corev1.VolumeMount{{...}}),
```

## Files changed

- `internal/instrumentation/apachehttpd.go` — `slices.Concat` on clone volumeMounts
- `internal/instrumentation/nginx.go` — same fix
- `internal/instrumentation/apachehttpd_test.go` — regression test with spare-capacity slice
- `internal/instrumentation/nginx_test.go` — same regression test
