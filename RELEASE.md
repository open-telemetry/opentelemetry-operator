# Release instructions

Steps to release a new version of the OpenTelemetry Operator:

1. Change the `versions.txt`, so that it lists the target version of the OpenTelemetry Collector (operand), and the desired version for the target allocator and the operator. The `major.minor` should typically match, with the patch portion being possibly different.
2. Change the `autoinstrumentation-*` versions in `versions.txt` as per the latest supported versions in `auto-instrumentation/`. 
3. Run `make bundle USER=open-telemetry VERSION=0.38.0`, using the version that will be released.
4. Change the compatibility matrix in the readme file, using the OpenTelemetry Operator version to be released and the current latest Kubernetes version as the latest supported version.
5. Add the changes to the changelog. Manually add versions of all operator components.
6. Check the OpenTelemetry Collector's changelog and ensure migration steps are present in `pkg/collector/upgrade`
7. Once the changes above are merged and available in `main`, tag it with the desired version, prefixed with `v`: `v0.38.0`
8. The GitHub Workflow will take it from here, creating a GitHub release with the generated artifacts (manifests) and publishing the images
9. After the release, generate a new OLM bundle (`make bundle`) and create two PRs against the `Community Operators repositories`:
   1. one for the `community-operators-prod`, used by OLM on Kubernetes. Example: [`operator-framework/community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/494)
   1. one for the `community-operators` directory, used by Operatorhub.io. Example: [`operator-framework/community-operators`](https://github.com/k8s-operatorhub/community-operators/pull/461)
10. Update release schedule table, by moving the current release manager to the end of the table with updated release version.

## Generating the changelog

We now use the chloggen to generate the changelog, simply run the following to generate the Changelog:

```bash
make chlog-update
```

This will delete all entries (other than the template) in the `.chloggen` directory and create a populated Changelog.md entry. Make sure that the PR you are generating for the release has the `[chore]` prefix, otherwise CI will not pass.


## Release managers

A release manager is the person responsible for a specific release. While the manager might request help from other folks, they are ultimately responsible for the success of a release.

In order to have more people comfortable with the release process, and in order to decrease the burden on a small number of volunteers, all approvers and maintainers are release managers from time to time, listed under the Release Schedule section. That table is updated at every release, with the current manager adding themselves to the bottom of the table, removing themselves from the top of the table.

## Release schedule

The operator should be released within a week after the [OpenTelemetry collector release](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/release.md#release-schedule).

| Version | Release manager |
|---------|-----------------|
| v0.73.0 | @jaronoff97     |
| v0.74.0 | @pavolloffay    |
| v0.75.0 | @VineethReddy02 |
| v0.76.0 | @frzifus        |
| v0.77.0 | @yuriolisa      |
