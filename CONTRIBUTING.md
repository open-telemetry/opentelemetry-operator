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
$ make manifests install run
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
make manifests container container-push deploy
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

## Project Structure

For a general overview of the directories from this operator and what to expect in each one of them, please check out the [official GoDoc](https://godoc.org/github.com/open-telemetry/opentelemetry-operator) or the [locally-hosted GoDoc](http://localhost:6060/pkg/github.com/open-telemetry/opentelemetry-operator/)

## Contributing

Your contribution is welcome! For it to be accepted, we have a few standards that must be followed.

### New features

Before starting the development of a new feature, please create an issue and discuss it with the project maintainers. Features should come with documentation and enough tests (unit and/or end-to-end).

### Bug fixes

Every bug fix should be accompanied with a unit test, so that we can prevent regressions.

### Documentation, typos, ...

They are mostly welcome!

## Operator SDK

Coming soon, but for the moment:

Build the operator
```
BUNDLE_VERSION=0.16.0
make set-image-controller IMG=quay.io/jpkroehling/opentelemetry-operator
make bundle
make bundle-build BUNDLE_IMG=quay.io/jpkroehling/opentelemetry-operator-bundle:${BUNDLE_VERSION}
podman push quay.io/jpkroehling/opentelemetry-operator-bundle:${BUNDLE_VERSION}
opm index add --bundles quay.io/jpkroehling/opentelemetry-operator-bundle:${BUNDLE_VERSION} --tag quay.io/jpkroehling/opentelemetry-operator-index:${BUNDLE_VERSION}
podman push quay.io/jpkroehling/opentelemetry-operator-index:${BUNDLE_VERSION}
```

Setup OLM (not needed on OpenShift):
```
cd ${OLM_HOME}
kubectl create -f deploy/upstream/quickstart/crds.yaml
kubectl create -f deploy/upstream/quickstart/olm.yaml
kubectl wait --for=condition=available deployment packageserver -n olm
kubectl wait --for=condition=available deployment olm-operator -n olm
kubectl wait --for=condition=available deployment catalog-operator -n olm
```

Install the operator
```
kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: opentelemetry-operator-manifests
  namespace: openshift-operators # or olm for Kubernetes
spec:
  sourceType: grpc
  image: quay.io/jpkroehling/opentelemetry-operator-index:${BUNDLE_VERSION}
EOF
kubectl wait --for=condition=ready pod -l olm.catalogSource=opentelemetry-operator-manifests -n olm

kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: opentelemetry-operator-subscription
  namespace: openshift-operators # or just operators on Kubernetes
spec:
  channel: "alpha"
  installPlanApproval: Automatic
  name: opentelemetry-operator
  source: opentelemetry-operator-manifests
  sourceNamespace: openshift-operators
EOF

```