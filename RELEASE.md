# Release instructions

Steps to release a new version of the OpenTelemetry Operator:

1. Change the `versions.txt`, so that it lists the target version of the OpenTelemetry Collector (operand), and the desired version for the target allocator and the operator. The `major.minor` should typically match, with the patch portion being possibly different.
2. Change the `autoinstrumentation-*` versions in `versions.txt` as per the latest supported versions in `auto-instrumentation/`. 
3. Run `make bundle USER=open-telemetry VERSION=0.38.0`, using the version that will be released.
4. Change the compatibility matrix in the readme file, using the OpenTelemetry Operator version to be released and the current latest Kubernetes version as the latest supported version, with N-2 being the lower bound. Make sure that the CI is currently testing the latest Kubernetes version!
5. Add the changes to the changelog
6. Check the OpenTelemetry Collector's changelog and ensure migration steps are present in `pkg/collector/upgrade`
7. Once the changes above are merged and available in `main`, tag it with the desired version, prefixed with `v`: `v0.38.0`
8. The GitHub Workflow will take it from here, creating a GitHub release with the generated artifacts (manifests) and publishing the images
9. After the release, generate a new OLM bundle (`make bundle`) and create two PRs against the `Community Operators repositories`:
   1. one for the `community-operators-prod`, used by OLM on Kubernetes. Example: [`operator-framework/community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/494)
   1. one for the `community-operators` directory, used by Operatorhub.io. Example: [`operator-framework/community-operators`](https://github.com/k8s-operatorhub/community-operators/pull/461)

## Generating the changelog

We now use the chloggen to generate the changelog, simply run the following to generate the Changelog:

```bash
make chlog-update
```

This will delete all entries (other than the template) in the `.chloggen` directory and create a populated Changelog.md entry. Make sure that the PR you are generating for the release has the `[chore]` prefix, otherwise CI will not pass.
