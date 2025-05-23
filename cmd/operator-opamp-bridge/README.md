# OpAMP Bridge

OpAMP Bridge is an optional component of the OpenTelemetry Operator that can be used to report and manage the state of OpenTelemetry Collectors in Kubernetes. It implements the agent-side of the [OpAMP protocol](https://opentelemetry.io/docs/specs/opamp/) and communicates with an OpAMP server.

The OpAMP Bridge is able to:
- Report the status and effective configuration of OpenTelemetryCollector CRD instances in a Kubernetes cluster to an OpAMP server
- Receive OpenTelemetryCollector CRD configurations from an OpAMP server and perform necessary CRUD operations with the Kubernetes API server to modify OpenTelemetry Collector resources
- Emit its own telemetry to an OTLP/HTTP endpoint

Further information and design of the OpAMP Bridge can be found in [OpAMP for OpenTelemetry Operator](https://docs.google.com/document/d/1M8VLNe_sv1MIfu5bUR5OV_vrMBnAI7IJN-7-IAr37JY/edit?usp=sharing).

Examples of OpAMP server implementations that the OpAMP Bridge can interact with include [jaronoff97/opamp-elixir](https://github.com/jaronoff97/opamp-elixir) and [jaronoff97/opamp-operator-server](https://github.com/jaronoff97/opamp-operator-server).

## Installation

There are two main ways to install the OpAMP Bridge:

1. As part of the OpenTelemetry Operator: The OpAMP Bridge is included with the OpenTelemetry Operator installation and can be deployed by creating an OpAMPBridge custom resource.
2. Using the [OpenTelemetry Kube Stack Helm Chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-kube-stack): The OpAMP Bridge is available as a component in the Helm chart, which serves as a quickstart that installs an OpenTelemetry Operator and a suite of collectors.

## Usage

### OpAMPBridge CRD

The [OpAMPBridge](../../docs/api/opampbridges.md) CRD is used to create an OpAMP Bridge instance.

The following example creates an OpAMP Bridge that can report the health and manage the state of OpenTelemetryCollector CRD instances, allowing for a specific set of OpenTelemetry Collector components to be used:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpAMPBridge
metadata:
  name: opamp-bridge
spec:
  endpoint: "<OPAMP_SERVER_ENDPOINT>"
  capabilities:
    AcceptsRemoteConfig: true
    ReportsEffectiveConfig: true
    ReportsHealth: true
    ReportsRemoteConfig: true
  componentsAllowed:
    receivers:
      - otlp
    processors:
      - memory_limiter
      - batch
    exporters:
      - otlphttp
```

### OpenTelemetryCollector CRD

The [OpenTelemetryCollector](../../docs/api/opentelemetrycollectors.md) CRD needs to be annotated with a label to be operated by the OpAMP Bridge:
- `opentelemetry.io/opamp-reporting` for reporting only
- `opentelemetry.io/opamp-managed` for reporting and management

#### OpAMP Reporting

The `opentelemetry.io/opamp-reporting` label is used to enable reporting only:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: opamp-reporting-collector
  labels:
    opentelemetry.io/opamp-reporting: "true"
spec:
...
```

#### OpAMP Managed

The `opentelemetry.io/opamp-managed` label is used to enable reporting and management:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: opamp-managed-collector
  labels:
    opentelemetry.io/opamp-managed: "true"
spec:
...
```

Alternatively, the name of an OpAMP Bridge can be set to be managed by a specific OpAMP Bridge instance:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: opamp-managed-collector
  labels:
    opentelemetry.io/opamp-managed: opamp-bridge
spec:
...
```

### RBAC

For the OpAMP Bridge to be able to report and manage OpenTelemetryCollectors CRD instances, Kubernetes role-based access control (RBAC) needs to be set up with `ServiceAccount`, `ClusterRole` and `ClusterRoleBinding` resources.

To use an existing service account, the `OpAMPBridge.spec.serviceAccount` can be set:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpAMPBridge
metadata:
  name: opamp-bridge
spec:
  serviceAccount: opamp-bridge-sa
...
```

If omitted, the operator automatically creates a new service account for the OpAMP Bridge. Its name will be a concatenation of the OpAMP Bridge's name and the `-opamp-bridge` suffix. By default, this service account has no defined policy, so a cluster role and a cluster role binding need to be created as per below.

The cluster role provides the OpAMP Bridge with permissions to report and manage OpenTelemetry Collector resources:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: opamp-bridge-role
rules:
- apiGroups:
    - opentelemetry.io
  resources:
    - opentelemetrycollectors
  verbs:
    - "*"
- apiGroups:
    - ""
  resources:
    - pods
  verbs:
    - get
    - list
```

The cluster role binding assigns the role above to the OpAMP Bridge service account:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: opamp-bridge-rolebinding
subjects:
- kind: ServiceAccount
  name: opamp-bridge-sa
  namespace: default
roleRef:
  kind: ClusterRole
  name: opamp-bridge-role
  apiGroup: rbac.authorization.k8s.io
```
