# OpenTelemetry Operator Must-Gather

The OpenTelemetry Operator `must-gather` tool is designed to collect comprehensive information about OpenTelemetry components within an OpenShift cluster. This utility extends the functionality of [OpenShift must-gather](https://github.com/openshift/must-gather) by specifically targeting and retrieving data related to the OpenTelemetry Operator, helping in diagnostics and troubleshooting.

Note that you can use this utility too to gather information about the objects deployed by the OpenTelemetry Operator if you don't use OpenShift.

## What is a Must-Gather?

The `must-gather` tool is a utility that collects logs, cluster information, and resource configurations related to a specific operator or application in an OpenShift cluster. It helps cluster administrators and developers diagnose issues by providing a snapshot of the cluster's state related to the targeted component. More information [in the official documentation](https://docs.openshift.com/container-platform/4.16/support/gathering-cluster-data.html).

## Usage

To run the must-gather tool for the OpenTelemetry Operator, use one of the following commands, depending on how you want to source the image and the namespace where the operator is deployed.

### Using the image from the Operator deployment

If you want to use the image directly from the existing OpenTelemetry Operator deployment, run the following command:

```sh
oc adm must-gather --image=$(oc -n opentelemetry-operator-system get deployment.apps/opentelemetry-operator-controller-manager -o jsonpath='{.spec.template.spec.containers[?(@.name == "manager")].image}') -- /must-gather --namespace opentelemetry-operator-system
```

### Using the image from a local machine

You can use the image in your machine with the following command:
```sh
docker run --entrypoint=/must-gather <operator-image> --help
```

This is the recommended way to do it if you are not using OpenShift.