# How to Contribute to the OpenTelemetry Operator

We'd love your help!

This project is [Apache 2.0 licensed](LICENSE) and accepts contributions via GitHub pull requests. This document outlines some of the conventions on development workflow, contact points and other resources to make it easier to get your contribution accepted.

We gratefully welcome improvements to documentation as well as to code.

## Getting Started

### Workflow

It is recommended to follow the ["GitHub Workflow"](https://guides.github.com/introduction/flow/). When using [GitHub's CLI](https://github.com/cli/cli), here's how it typically looks like:

```bash
gh repo fork github.com/open-telemetry/opentelemetry-operator
git checkout -b your-feature-branch
# do your changes
git commit -sam "Add feature X"
gh pr create
```

#### Make changes to the project manifests

The following command should be run to make sure the project manifests are up-to-date:

```bash
make generate manifests bundle api-docs reset
```

The local changes after running the command should be added to the pull request:

The following `make` target is run on CI to verify the project structure:

```bash
make ensure-generate-is-noop
```

### Pre-requisites
* Install [Go](https://golang.org/doc/install).
* Install [Kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/).
* Install [Operator SDK](https://sdk.operatorframework.io/docs/installation/).
* Have a Kubernetes cluster ready for development. We recommend `minikube` or `kind`.
* Docker version 23.0.0 or greater.

### Adding new components - webhook, API

The repository structure MUST be compliant with `operator-sdk` scaffolding, which uses `kubebuilder` behind the scenes. This is to ensure a valid bundle generation and it makes it easy to maintain the project and add new components.

Refer to the [Operator SDK documentation](https://sdk.operatorframework.io/docs/building-operators/golang/) how to generate new APIs, Webhook and other parts of the project.

### Local run

Build the manifests, install the CRD and run the operator as a local process:
```bash
make install run
```

### Deployment with webhooks

When running `make run`, the webhooks aren't effective as it starts the manager in the local machine instead of in-cluster. To test the webhooks, you'll need to:

1. configure a proxy between the Kubernetes API server and your host, so that it can contact the webhook in your local machine
1. create the TLS certificates and place them, by default, on `/tmp/k8s-webhook-server/serving-certs/tls.crt`. The Kubernetes API server has also to be configured to trust the CA used to generate those certs.

In general, it's just easier to deploy the operator in a Kubernetes cluster instead. For that, you'll need the `cert-manager` installed. You can install it by running:

```bash
make cert-manager
```

The environment variable `CERTMANAGER_VERSION` can be used to override the cert-manager version:
```bash
CERTMANAGER_VERSION=1.60 make cert-manager
```

When deploying the operator into the cluster using `make deploy`, an image in the format `ghcr.io/${DOCKER_USER}/opentelemetry-operator` is generated. If this format isn't suitable, it can be overridden by:

* `IMG_PREFIX`, to override the registry, namespace and image name
* `DOCKER_USER`, to override the namespace
* `IMG_REPO`, to override the repository (`opentelemetry-operator`)
* `VERSION`, to override only the version part
* `IMG`, to override the entire image specification

```bash
IMG=docker.io/${DOCKER_USER}/opentelemetry-operator:dev-$(git rev-parse --short HEAD)-$(date +%s) make generate container container-push deploy
```

Your operator will be available in the `opentelemetry-operator-system` namespace.

#### Using a private container registry

Ensure the secret [regcred](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/) has been created to enable opentelemetry-operator-controller-manager deployment to pull images from your private registry.

```bash
kubectl create secret docker-registry regcred --docker-server=<registry> --docker-username=${USER} --docker-password=${PASSWORD}  -n opentelemetry-operator-system
```

## Testing

With an existing cluster (such as `minikube`), run:
```bash
USE_EXISTING_CLUSTER=true make test
```

Tests can also be run without an existing cluster. For that, install [`kubebuilder`](https://book.kubebuilder.io/quick-start.html#installation). In this case, the tests will bootstrap `etcd` and `kubernetes-api-server` for the tests. Run against an existing cluster whenever possible, though.

### Unit tests

Some unit tests use [envtest](https://book.kubebuilder.io/reference/envtest.html) which requires Kubernetes binaries (e.g. `api-server`, `etcd` and `kubectl`) to be present on the host filesystem. Makefile takes care of installing all dependent binaries, however running the tests from IDE or via `go test` might not work out-of-the-box. The `envtest` uses env variable `KUBEBUILDER_ASSETS` that points to a directory with these binaries. To make the test work in IDE or `go test` the environment variable has to be correctly set.

Example how to run test that use `envtest`:

```bash
make envtest
KUBEBUILDER_ASSETS=$(./bin/setup-envtest use -p path 1.23) go test ./pkg...
```

### End to end tests

To run the end-to-end tests, you'll need [`kind`](https://kind.sigs.k8s.io) and [`kuttl`](https://kuttl.dev). Refer to their documentation for installation instructions.

Once they are installed, the tests can be executed with `make prepare-e2e`, which will build an image to use with the tests, followed by `make e2e`. Each call to the `e2e` target will setup a fresh `kind` cluster, making it safe to be executed multiple times with a single `prepare-e2e` step.

The tests are located under `tests/e2e` and are written to be used with `kuttl`. Refer to their documentation to understand how tests are written.

To evert the changes made by the `make prepare-e2e` run `make reset`.

### OpenShift End to End tests
To run the end-to-end tests written for OpenShift, you'll need a OpenShift cluster. 

To install the OpenTelemetry operator, please follow the instructions in  [Operator Lifecycle Manager (OLM)](https://github.com/open-telemetry/opentelemetry-operator/blob/main/CONTRIBUTING.md#operator-lifecycle-manager-olm)

Once the operator is installed, the tests can be executed using `make e2e-openshift`, which will call to the `e2e-openshift` target. Note that `kind` is disabled for the TestSuite as the requirement is to use an OpenShift cluster for these test cases. 

The tests are located under `tests/e2e-openshift` and are written to be used with `kuttl`.

### Undeploying the operator from the local cluster

```bash
make undeploy
```

## Project Structure

For a general overview of the directories from this operator and what to expect in each one of them, please check out the [official GoDoc](https://godoc.org/github.com/open-telemetry/opentelemetry-operator) or the [locally-hosted GoDoc](http://localhost:6060/pkg/github.com/open-telemetry/opentelemetry-operator/)

## Contributing

Your contribution is welcome! For it to be accepted, we have a few standards that must be followed.

If you are contributing to sync the receivers from [otel-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib), note that the operator only synchronizes receivers that aren't scrapers, as there's no need to open ports in services for this case. In general, receivers would open a UDP/TCP port and the operator should be adding an entry in the Kubernetes Service resource accordingly.

### New features

Before starting the development of a new feature, please create an issue and discuss it with the project maintainers. Features should come with documentation and enough tests (unit and/or end-to-end).

### Bug fixes

Every bug fix should be accompanied with a unit test, so that we can prevent regressions.

### Documentation, typos, ...

They are mostly welcome!

### Adding a Changelog Entry

The [CHANGELOG.md](./CHANGELOG.md) file in this repo is autogenerated from `.yaml` files in the `./.chloggen` directory.

Your pull-request should add a new `.yaml` file to this directory. The name of your file must be unique since the last release.

During the collector release process, all `./.chloggen/*.yaml` files are transcribed into `CHANGELOG.md` and then deleted.

If a changelog entry is not required, add either `[chore]` to the title of the pull request or add the `"Skip Changelog"` label to disable this action.

**Recommended Steps**
1. Create an entry file using `make chlog-new`. This generates a file based on your current branch (e.g. `./.chloggen/my-branch.yaml`)
2. Fill in all fields in the new file
3. Run `make chlog-validate` to ensure the new file is valid
4. Commit and push the file

Alternately, copy `./.chloggen/TEMPLATE.yaml`, or just create your file from scratch.

## Operator Lifecycle Manager (OLM)

For production environments, it is recommended to use the [Operator Lifecycle Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager) to provision and update the OpenTelemetry Operator. Our operator is available in the [Operator Hub](https://operatorhub.io/operator/opentelemetry-operator), and when making changes involving those manifests the following steps can be used for testing. Refer to the [OLM documentation](https://sdk.operatorframework.io/docs/olm-integration/quickstart-bundle/) for more complete information.

### Setup OLM

When using Kubernetes, install OLM following the [official instructions](https://sdk.operatorframework.io/docs/olm-integration/). At the moment of this writing, it involves the following:

```bash
operator-sdk olm install
```

When using OpenShift, the OLM is already installed.

### Create the bundle and related images

The following commands will generate a bundle under `bundle/`, build an image with its contents, build and publish the operator image.

```bash
BUNDLE_IMG=docker.io/${USER}/opentelemetry-operator-bundle:latest IMG=docker.io/${USER}/opentelemetry-operator:latest make bundle container container-push bundle-build bundle-push
```

### Install the operator

```bash
operator-sdk run bundle docker.io/${DOCKER_USER}/opentelemetry-operator-bundle:latest
```

### Uninstall the operator

The operator can be uninstalled by deleting `subscriptions.operators.coreos.com` and `clusterserviceversion.operators.coreos.com` objects from the current namespace.

```bash
kubectl delete clusterserviceversion.operators.coreos.com --all
kubectl delete subscriptions.operators.coreos.com --all
```
