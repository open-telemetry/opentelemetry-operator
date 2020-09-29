# Release instructions

Steps to release a new version of the OpenTelemetry Operator:

1. Change the `versions.txt`, so that it lists the target version of the OpenTelemetry Collector (operand), and the desired version for the operator. The `major.minor` should typically match, with the patch portion being possibly different.
1. Add the changes to the changelog
1. Check the OpenTelemetry Collector's changelog and ensure migration steps are present in `pkg/collector/upgrade`
1. Once the changes above are merged and available in `master`, tag it with the desired version, prefixed with `v`: `v0.3.0`
1. The GitHub Workflow will take it from here, creating a GitHub release with the generated artifacts (manifests) and publishing the images

## Generating the changelog

Run this generator:
```console
$ docker run --rm  -v "${PWD}:/app" pavolloffay/gch:latest --oauth-token ${GH_WRITE_TOKEN} --owner open-telemetry --repo opentelemetry-operator
```

Remove the commits that are not relevant to users, like:
* CI or testing-specific commits (e2e, unit test, ...)
* bug fixes for problems that are not part of a release yet
