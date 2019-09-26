# How to Contribute to the OpenTelemetry Operator for Kubernetes

We'd love your help!

This project is [Apache 2.0 licensed](LICENSE) and accepts contributions via GitHub pull requests. This document outlines some of the conventions on development workflow, commit message formatting, contact points and other resources to make it easier to get your contribution accepted.

We gratefully welcome improvements to documentation as well as to code.

## Certificate of Origin

By contributing to this project you agree to the [Developer Certificate of Origin](https://developercertificate.org/) (DCO). This document was created by the Linux Kernel community and is a simple statement that you, as a contributor, have the legal right to make the contribution. See the [DCO](DCO) file for details.

## Getting Started

This project is a regular [Kubernetes Operator](https://coreos.com/operators/)  built using the Operator SDK. Refer to the Operator SDK documentation to understand the basic architecture of this operator.

### Installing the Operator SDK command line tool

Follow the installation guidelines from [Operator SDK GitHub page](https://github.com/operator-framework/operator-sdk).

### Developing

As usual for operators following the Operator SDK in recent versions, the dependencies are managed using [`go modules`](https://golang.org/doc/go1.11#modules). Refer to that project's documentation for instructions on how to add or update dependencies.

The first step is to get a local Kubernetes instance up and running. The recommended approach is using `minikube`. Refer to the Kubernetes'  [documentation](https://kubernetes.io/docs/tasks/tools/install-minikube/) for instructions on how to install it.

Once `minikube` is installed, it can be started with:

```
minikube start
```

NOTE: Make sure to read the documentation to learn the performance switches that can be applied to your platform.

Once minikube has finished starting, get the Operator running:

```
make run
```

At this point, an OpenTelemetry Collector instance can be installed:

```
kubectl apply -f deploy/examples/simplest.yaml
kubectl get otelcols
kubectl get pods
```

To remove the instance:

```
kubectl delete -f deploy/examples/simplest.yaml
```

#### Model changes

The Operator SDK generates the `pkg/apis/opentelemetry/v1alpha1//zz_generated.*.go` files via the command `make generate`. This should be executed whenever there's a model change (`pkg/apis/opentelemetry/v1alpha1//opentelemetrycollector_types.go`)

#### Tests

Right now, there are only unit tests in this repository. They can be executed via `make unit-tests`. End-to-end tests are planned, and should be executed via `make e2e-tests`. All tests, including unit and end-to-end, will be executed when `make test` is called.
