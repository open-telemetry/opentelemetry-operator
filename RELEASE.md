# Release instructions

Steps to release a new version of the OpenTelemetry Operator:

1. Create a `Prepare release x.y.z` pull request with the following content:
   1. Set the version you're releasing as an environment variable for convenience: `export VERSION=0.n+1.0`
   1. Update `versions.txt`
      - Operator, target allocator and opamp-bridge should be `$VERSION`.
      - OpenTelemetry Collector should be the latest collector version. The `major.minor` should typically match, with the patch portion being possibly different.
      - The `autoinstrumentation-*` versions should match the latest supported versions in `autoinstrumentation/`.
        > [!WARNING]
        > DO NOT BUMP JAVA PAST `1.X.X` AND DO NOT BUMP .NET PAST `1.2.0`. Upgrades past these versions will introduce breaking HTTP semantic convention changes.
   1. Check if the compatible OpenShift versions are updated in the `Makefile`.
   1. Update the bundle by running `make bundle reset VERSION=$VERSION`.
   1. Change the compatibility matrix in the [compatibility doc](./docs/compatibility.md) file, using the OpenTelemetry Operator version to be released and the current latest Kubernetes version as the latest supported version. Remove the oldest entry.
   1. Update release schedule table, by moving the current release manager to the end of the table with updated release version.
   1. Add the changes to the changelog by running `make chlog-update VERSION=$VERSION`.
   1. Check the OpenTelemetry Collector's changelog and ensure migration steps are present in `pkg/collector/upgrade`
1. Once the changes above are merged and available in `main`, a draft release will be automatically created. Publish it once the release workflows all complete.
1. Update the operator version in the Helm Chart, as per the [release guide](https://github.com/open-telemetry/opentelemetry-helm-charts/blob/main/charts/opentelemetry-operator/CONTRIBUTING.md)
1. The GitHub Workflow, submits two pull requests to the Operator hub repositories. Make sure the pull requests are approved and merged.
    - `community-operators-prod` is used by OLM on OpenShift. Example: [`operator-framework/community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/494)
    - `community-operators` is used by Operatorhub.io. Example: [`operator-framework/community-operators`](https://github.com/k8s-operatorhub/community-operators/pull/461)

## Generating the changelog

We now use the chloggen to generate the changelog, simply run the following to generate the Changelog:

```bash
make chlog-update VERSION=$VERSION
```

This will delete all entries (other than the template) in the `.chloggen` directory and create a populated Changelog.md entry. Make sure that the PR you are generating for the release has the `[chore]` prefix, otherwise CI will not pass.

## Release managers

A release manager is the person responsible for a specific release. While the manager might request help from other folks, they are ultimately responsible for the success of a release.

In order to have more people comfortable with the release process, and in order to decrease the burden on a small number of volunteers, all approvers and maintainers are release managers from time to time, listed under the Release Schedule section. That table is updated at every release, with the current manager adding themselves to the bottom of the table, removing themselves from the top of the table.

## Release schedule

The operator should be released within a week after the [OpenTelemetry collector release](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/release.md#release-schedule).

| Version  | Release manager |
|----------|-----------------|
| v0.119.0 | @frzifus        |
| v0.120.0 | @yuriolisa      |
| v0.121.0 | @pavolloffay    |
| v0.122.0 | @TylerHelmuth   |
| v0.123.0 | @jaronoff97     |
| v0.124.0 | @swiatekm       |
| v0.125.0 | @iblancasa      |
