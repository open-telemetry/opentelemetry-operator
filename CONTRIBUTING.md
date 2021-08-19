# How to Contribute to the OpenTelemetry Operator

We'd love your help!

This project is [Apache 2.0 licensed](LICENSE) and accepts contributions via GitHub pull requests. This document outlines some of the conventions on development workflow, contact points and other resources to make it easier to get your contribution accepted.

We gratefully welcome improvements to documentation as well as to code.

## Getting Started

### Workflow

It is recommended to follow the ["GitHub Workflow"](https://guides.github.com/introduction/flow/). When using [GitHub's CLI](https://github.com/cli/cli), here's how it typically looks like:

```
$ gh repo fork github.com/open-telemetry/opentelemetry-operator
$ git checkout -b your-feature-branch
# do your changes
$ git commit -sam "Add feature X"
$ gh pr create
```

### Pre-requisites
* Install [Go](https://golang.org/doc/install).
* Have a Kubernetes cluster ready for development. We recommend `minikube` or `kind`.

### Local run

Build the manifests, install the CRD and run the operator as a local process:
```
$ make bundle install run
```

### Deployment with webhooks

When running `make run`, the webhooks aren't effective as it starts the manager in the local machine instead of in-cluster. To test the webhooks, you'll need to:

1. configure a proxy between the Kubernetes API server and your host, so that it can contact the webhook in your local machine
1. create the TLS certificates and place them, by default, on `/tmp/k8s-webhook-server/serving-certs/tls.crt`. The Kubernetes API server has also to be configured to trust the CA used to generate those certs.

In general, it's just easier to deploy the manager in a Kubernetes cluster instead. For that, you'll need the `cert-manager` installed. You can install it by running:

```console
make cert-manager
```

Once it's ready, the following can be used to build and deploy a manager, along with the required webhook configuration:

```
make bundle container container-push deploy
```

By default, it will generate an image following the format `quay.io/${USER}/opentelemetry-operator:${VERSION}`. You can set the following env vars in front of the `make` command to override parts or the entirety of the image:

* `IMG_PREFIX`, to override the registry, namespace and image name (`quay.io`)
* `USER`, to override the namespace
* `IMG_REPO`, to override the repository (`opentelemetry-operator`)
* `VERSION`, to override only the version part
* `IMG`, to override the entire image specification

Your operator will be available in the `opentelemetry-operator-system` namespace.

## Testing

With an existing cluster (such as `minikube`), run:
```
USE_EXISTING_CLUSTER=true make test
```

Tests can also be run without an existing cluster. For that, install [`kubebuilder`](https://book.kubebuilder.io/quick-start.html#installation). In this case, the tests will bootstrap `etcd` and `kubernetes-api-server` for the tests. Run against an existing cluster whenever possible, though.

### End to end tests

To run the end-to-end tests, you'll need [`kind`](https://kind.sigs.k8s.io) and [`kuttl`](https://kuttl.dev). Refer to their documentation for installation instructions.

Once they are installed, the tests can be executed with `make prepare-e2e`, which will build an image to use with the tests, followed by `make e2e`. Each call to the `e2e` target will setup a fresh `kind` cluster, making it safe to be executed multiple times with a single `prepare-e2e` step.

The tests are located under `tests/e2e` and are written to be used with `kuttl`. Refer to their documentation to understand how tests are written.

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

## Operator Lifecycle Manager (OLM)

For production environments, it is recommended to use the [Operator Lifecycle Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager) to provision and update the OpenTelemetry Operator. Our operator is available in the [Operator Hub](https://operatorhub.io/operator/opentelemetry-operator), and when making changes involving those manifests the following steps can be used for testing. Refer to the [OLM documentation](https://sdk.operatorframework.io/docs/olm-integration/quickstart-bundle/) for more complete information.

### Setup OLM

When using Kubernetes, install OLM following the [official instructions](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md). At the moment of this writing, it involves the following:

```
kubectl create -f https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/quickstart/crds.yaml
kubectl create -f https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/quickstart/olm.yaml
kubectl wait --for=condition=available deployment packageserver -n olm
kubectl wait --for=condition=available deployment olm-operator -n olm
kubectl wait --for=condition=available deployment catalog-operator -n olm
```

When using OpenShift, OLM is already installed.

### Create the bundle and related images

The following commands will generate a bundle under `bundle/` and build an image with its contents. It will then generate and publish an index image with the [Operator Package Manager (OPM)](https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md#opm)

```
export VERSION=x.y.z
make set-image-controller bundle bundle-build
podman push quay.io/${USER}/opentelemetry-operator-bundle:${VERSION}
opm index add --bundles quay.io/${USER}/opentelemetry-operator-bundle:${VERSION} --tag quay.io/${USER}/opentelemetry-operator-index:${VERSION}
podman push quay.io/${USER}/opentelemetry-operator-index:${VERSION}
```

### Install the operator

To install our operator, create a `CatalogSource` for our index image, wait for OLM to synchronize and finally create a `Subscription`. Make sure to replace `${USER}` with your username and `${VERSION}` with the version used in the previous step. The namespace for both should be `operators` on Kubernetes, while `openshift-operators` should be used for OpenShift.

```
kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: opentelemetry-operator-manifests
  namespace: operators
spec:
  sourceType: grpc
  image: quay.io/${USER}/opentelemetry-operator-index:${VERSION}
EOF
kubectl wait --for=condition=ready pod -l olm.catalogSource=opentelemetry-operator-manifests -n operators

kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: opentelemetry-operator-subscription
  namespace: operators
spec:
  channel: "alpha"
  installPlanApproval: Automatic
  name: opentelemetry-operator
  source: opentelemetry-operator-manifests
  sourceNamespace: operators
EOF
kubectl wait --for=condition=available deployment opentelemetry-operator-controller-manager -n operators
```