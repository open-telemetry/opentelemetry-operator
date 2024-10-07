# OpenTelemetry Operator Must-Gather

The OpenTelemetry Operator `must-gather` tool is designed to collect comprehensive information about OpenTelemetry components within an OpenShift cluster. This utility extends the functionality of [OpenShift must-gather](https://github.com/openshift/must-gather) by specifically targeting and retrieving data related to the OpenTelemetry Operator, helping in diagnostics and troubleshooting.

Note that you can use this utility too to gather information about the objects deployed by the OpenTelemetry Operator if you don't use OpenShift.

## What is a Must-Gather?

The `must-gather` tool is a utility that collects logs, cluster information, and resource configurations related to a specific operator or application in an OpenShift cluster. It helps cluster administrators and developers diagnose issues by providing a snapshot of the cluster's state related to the targeted component. More information [in the official documentation](https://docs.openshift.com/container-platform/4.16/support/gathering-cluster-data.html).

## Usage

First, you will need to build and push the image:
```sh
make container-must-gather container-must-gather-push
```

To run the must-gather tool for the OpenTelemetry Operator, use one of the following commands, depending on how you want to source the image and the namespace where the operator is deployed.

### Using the image from the Operator deployment

This is the recommended way to do it if you are not using OpenShift.

If you want to use the image in a running cluster, you need to run the following command:

```sh
oc adm must-gather --image=ghcr.io/open-telemetry/opentelemetry-operator/must-gather -- /usr/bin/must-gather --operator-namespace opentelemetry-operator-system
```

### Using it as a CLI

You only need to build and run:
```sh
make must-gather
./bin/must-gather_$(go env GOARCH) --help
```
