# API Reference

Packages:

- [opentelemetry.io/v1alpha1](#opentelemetryiov1alpha1)

# opentelemetry.io/v1alpha1

Resource Types:

- [Instrumentation](#instrumentation)

- [OpAMPBridge](#opampbridge)

- [OpenTelemetryCollector](#opentelemetrycollector)




## Instrumentation
<sup><sup>[↩ Parent](#opentelemetryiov1alpha1 )</sup></sup>






Instrumentation is the spec for OpenTelemetry instrumentation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>opentelemetry.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Instrumentation</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#instrumentationspec">spec</a></b></td>
        <td>object</td>
        <td>
          InstrumentationSpec defines the desired state of OpenTelemetry SDK and instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>object</td>
        <td>
          InstrumentationStatus defines status of the instrumentation.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec
<sup><sup>[↩ Parent](#instrumentation)</sup></sup>



InstrumentationSpec defines the desired state of OpenTelemetry SDK and instrumentation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecapachehttpd">apacheHttpd</a></b></td>
        <td>object</td>
        <td>
          ApacheHttpd defines configuration for Apache HTTPD auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecdotnet">dotnet</a></b></td>
        <td>object</td>
        <td>
          DotNet defines configuration for DotNet auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          Env defines common env vars.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecexporter">exporter</a></b></td>
        <td>object</td>
        <td>
          Exporter defines exporter configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecgo">go</a></b></td>
        <td>object</td>
        <td>
          Go defines configuration for Go auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecjava">java</a></b></td>
        <td>object</td>
        <td>
          Java defines configuration for java auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginx">nginx</a></b></td>
        <td>object</td>
        <td>
          Nginx defines configuration for Nginx auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnodejs">nodejs</a></b></td>
        <td>object</td>
        <td>
          NodeJS defines configuration for nodejs auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>propagators</b></td>
        <td>[]enum</td>
        <td>
          Propagators defines inter-process context propagation configuration. Values in this list will be set in the OTEL_PROPAGATORS env var. Enum=tracecontext;baggage;b3;b3multi;jaeger;xray;ottrace;none<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecpython">python</a></b></td>
        <td>object</td>
        <td>
          Python defines configuration for python auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecresource">resource</a></b></td>
        <td>object</td>
        <td>
          Resource defines the configuration for the resource attributes, as defined by the OpenTelemetry specification.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecsampler">sampler</a></b></td>
        <td>object</td>
        <td>
          Sampler defines sampling configuration.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



ApacheHttpd defines configuration for Apache HTTPD auto-instrumentation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecapachehttpdattrsindex">attrs</a></b></td>
        <td>[]object</td>
        <td>
          Attrs defines Apache HTTPD agent specific attributes. The precedence is: `agent default attributes` > `instrument spec attributes` . Attributes are documented at https://github.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>configPath</b></td>
        <td>string</td>
        <td>
          Location of Apache HTTPD server configuration. Needed only if different from default "/usr/local/apache2/conf"<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          Env defines Apache HTTPD specific env vars.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image is a container image with Apache SDK and auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdresourcerequirements">resourceRequirements</a></b></td>
        <td>object</td>
        <td>
          Resources describes the compute resource requirements.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>version</b></td>
        <td>string</td>
        <td>
          Apache HTTPD server version. One of 2.4 or 2.2. Default is 2.4<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeLimitSize</b></td>
        <td>int or string</td>
        <td>
          VolumeSizeLimit defines size limit for volume used for auto-instrumentation. The default size is 200Mi.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.attrs[index]
<sup><sup>[↩ Parent](#instrumentationspecapachehttpd)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdattrsindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.attrs[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdattrsindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecapachehttpdattrsindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdattrsindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdattrsindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdattrsindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.attrs[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdattrsindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.attrs[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdattrsindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.attrs[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdattrsindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.attrs[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdattrsindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.env[index]
<sup><sup>[↩ Parent](#instrumentationspecapachehttpd)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.env[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecapachehttpdenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecapachehttpdenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.resourceRequirements
<sup><sup>[↩ Parent](#instrumentationspecapachehttpd)</sup></sup>



Resources describes the compute resource requirements.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecapachehttpdresourcerequirementsclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.apacheHttpd.resourceRequirements.claims[index]
<sup><sup>[↩ Parent](#instrumentationspecapachehttpdresourcerequirements)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Instrumentation.spec.dotnet
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



DotNet defines configuration for DotNet auto-instrumentation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecdotnetenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          Env defines DotNet specific env vars.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image is a container image with DotNet SDK and auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecdotnetresourcerequirements">resourceRequirements</a></b></td>
        <td>object</td>
        <td>
          Resources describes the compute resource requirements.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeLimitSize</b></td>
        <td>int or string</td>
        <td>
          VolumeSizeLimit defines size limit for volume used for auto-instrumentation. The default size is 200Mi.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.dotnet.env[index]
<sup><sup>[↩ Parent](#instrumentationspecdotnet)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecdotnetenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.dotnet.env[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecdotnetenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecdotnetenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecdotnetenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecdotnetenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecdotnetenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.dotnet.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecdotnetenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.dotnet.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecdotnetenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.dotnet.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecdotnetenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.dotnet.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecdotnetenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.dotnet.resourceRequirements
<sup><sup>[↩ Parent](#instrumentationspecdotnet)</sup></sup>



Resources describes the compute resource requirements.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecdotnetresourcerequirementsclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.dotnet.resourceRequirements.claims[index]
<sup><sup>[↩ Parent](#instrumentationspecdotnetresourcerequirements)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Instrumentation.spec.env[index]
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.env[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.exporter
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



Exporter defines exporter configuration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>endpoint</b></td>
        <td>string</td>
        <td>
          Endpoint is address of the collector with OTLP endpoint.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.go
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



Go defines configuration for Go auto-instrumentation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecgoenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          Env defines Go specific env vars.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image is a container image with Go SDK and auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecgoresourcerequirements">resourceRequirements</a></b></td>
        <td>object</td>
        <td>
          Resources describes the compute resource requirements.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeLimitSize</b></td>
        <td>int or string</td>
        <td>
          VolumeSizeLimit defines size limit for volume used for auto-instrumentation. The default size is 200Mi.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.go.env[index]
<sup><sup>[↩ Parent](#instrumentationspecgo)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecgoenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.go.env[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecgoenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecgoenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecgoenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecgoenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecgoenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.go.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecgoenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.go.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecgoenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.go.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecgoenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.go.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecgoenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.go.resourceRequirements
<sup><sup>[↩ Parent](#instrumentationspecgo)</sup></sup>



Resources describes the compute resource requirements.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecgoresourcerequirementsclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.go.resourceRequirements.claims[index]
<sup><sup>[↩ Parent](#instrumentationspecgoresourcerequirements)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Instrumentation.spec.java
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



Java defines configuration for java auto-instrumentation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecjavaenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          Env defines java specific env vars.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image is a container image with javaagent auto-instrumentation JAR.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecjavaresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources describes the compute resource requirements.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeLimitSize</b></td>
        <td>int or string</td>
        <td>
          VolumeSizeLimit defines size limit for volume used for auto-instrumentation. The default size is 200Mi.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.java.env[index]
<sup><sup>[↩ Parent](#instrumentationspecjava)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecjavaenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.java.env[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecjavaenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecjavaenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecjavaenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecjavaenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecjavaenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.java.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecjavaenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.java.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecjavaenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.java.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecjavaenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.java.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecjavaenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.java.resources
<sup><sup>[↩ Parent](#instrumentationspecjava)</sup></sup>



Resources describes the compute resource requirements.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecjavaresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.java.resources.claims[index]
<sup><sup>[↩ Parent](#instrumentationspecjavaresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



Nginx defines configuration for Nginx auto-instrumentation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecnginxattrsindex">attrs</a></b></td>
        <td>[]object</td>
        <td>
          Attrs defines Nginx agent specific attributes. The precedence order is: `agent default attributes` > `instrument spec attributes` . Attributes are documented at https://github.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>configFile</b></td>
        <td>string</td>
        <td>
          Location of Nginx configuration file. Needed only if different from default "/etx/nginx/nginx.conf"<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          Env defines Nginx specific env vars.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image is a container image with Nginx SDK and auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxresourcerequirements">resourceRequirements</a></b></td>
        <td>object</td>
        <td>
          Resources describes the compute resource requirements.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeLimitSize</b></td>
        <td>int or string</td>
        <td>
          VolumeSizeLimit defines size limit for volume used for auto-instrumentation. The default size is 200Mi.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.attrs[index]
<sup><sup>[↩ Parent](#instrumentationspecnginx)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxattrsindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.attrs[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecnginxattrsindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecnginxattrsindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxattrsindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxattrsindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxattrsindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.attrs[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecnginxattrsindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.attrs[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecnginxattrsindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.attrs[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecnginxattrsindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.attrs[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecnginxattrsindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.env[index]
<sup><sup>[↩ Parent](#instrumentationspecnginx)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.env[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecnginxenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecnginxenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnginxenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecnginxenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecnginxenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecnginxenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecnginxenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.resourceRequirements
<sup><sup>[↩ Parent](#instrumentationspecnginx)</sup></sup>



Resources describes the compute resource requirements.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecnginxresourcerequirementsclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nginx.resourceRequirements.claims[index]
<sup><sup>[↩ Parent](#instrumentationspecnginxresourcerequirements)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nodejs
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



NodeJS defines configuration for nodejs auto-instrumentation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecnodejsenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          Env defines nodejs specific env vars.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image is a container image with NodeJS SDK and auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnodejsresourcerequirements">resourceRequirements</a></b></td>
        <td>object</td>
        <td>
          Resources describes the compute resource requirements.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeLimitSize</b></td>
        <td>int or string</td>
        <td>
          VolumeSizeLimit defines size limit for volume used for auto-instrumentation. The default size is 200Mi.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nodejs.env[index]
<sup><sup>[↩ Parent](#instrumentationspecnodejs)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnodejsenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nodejs.env[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecnodejsenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecnodejsenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnodejsenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnodejsenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecnodejsenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nodejs.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecnodejsenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nodejs.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecnodejsenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nodejs.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecnodejsenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nodejs.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecnodejsenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nodejs.resourceRequirements
<sup><sup>[↩ Parent](#instrumentationspecnodejs)</sup></sup>



Resources describes the compute resource requirements.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecnodejsresourcerequirementsclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.nodejs.resourceRequirements.claims[index]
<sup><sup>[↩ Parent](#instrumentationspecnodejsresourcerequirements)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Instrumentation.spec.python
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



Python defines configuration for python auto-instrumentation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecpythonenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          Env defines python specific env vars.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image is a container image with Python SDK and auto-instrumentation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecpythonresourcerequirements">resourceRequirements</a></b></td>
        <td>object</td>
        <td>
          Resources describes the compute resource requirements.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeLimitSize</b></td>
        <td>int or string</td>
        <td>
          VolumeSizeLimit defines size limit for volume used for auto-instrumentation. The default size is 200Mi.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.python.env[index]
<sup><sup>[↩ Parent](#instrumentationspecpython)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecpythonenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.python.env[index].valueFrom
<sup><sup>[↩ Parent](#instrumentationspecpythonenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecpythonenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecpythonenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecpythonenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#instrumentationspecpythonenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.python.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#instrumentationspecpythonenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.python.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#instrumentationspecpythonenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.python.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#instrumentationspecpythonenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.python.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#instrumentationspecpythonenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.python.resourceRequirements
<sup><sup>[↩ Parent](#instrumentationspecpython)</sup></sup>



Resources describes the compute resource requirements.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#instrumentationspecpythonresourcerequirementsclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.python.resourceRequirements.claims[index]
<sup><sup>[↩ Parent](#instrumentationspecpythonresourcerequirements)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Instrumentation.spec.resource
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



Resource defines the configuration for the resource attributes, as defined by the OpenTelemetry specification.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>addK8sUIDAttributes</b></td>
        <td>boolean</td>
        <td>
          AddK8sUIDAttributes defines whether K8s UID attributes should be collected (e.g. k8s.deployment.uid).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>resourceAttributes</b></td>
        <td>map[string]string</td>
        <td>
          Attributes defines attributes that are added to the resource. For example environment: dev<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Instrumentation.spec.sampler
<sup><sup>[↩ Parent](#instrumentationspec)</sup></sup>



Sampler defines sampling configuration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>argument</b></td>
        <td>string</td>
        <td>
          Argument defines sampler argument. The value depends on the sampler type. For instance for parentbased_traceidratio sampler type it is a number in range [0..1] e.g. 0.25.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type defines sampler type. The value will be set in the OTEL_TRACES_SAMPLER env var. The value can be for instance parentbased_always_on, parentbased_always_off, parentbased_traceidratio...<br/>
          <br/>
            <i>Enum</i>: always_on, always_off, traceidratio, parentbased_always_on, parentbased_always_off, parentbased_traceidratio, jaeger_remote, xray<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## OpAMPBridge
<sup><sup>[↩ Parent](#opentelemetryiov1alpha1 )</sup></sup>






OpAMPBridge is the Schema for the opampbridges API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>opentelemetry.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>OpAMPBridge</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespec">spec</a></b></td>
        <td>object</td>
        <td>
          OpAMPBridgeSpec defines the desired state of OpAMPBridge.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgestatus">status</a></b></td>
        <td>object</td>
        <td>
          OpAMPBridgeStatus defines the observed state of OpAMPBridge.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec
<sup><sup>[↩ Parent](#opampbridge)</sup></sup>



OpAMPBridgeSpec defines the desired state of OpAMPBridge.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>capabilities</b></td>
        <td>map[string]boolean</td>
        <td>
          Capabilities supported by the OpAMP Bridge<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>endpoint</b></td>
        <td>string</td>
        <td>
          OpAMP backend Server endpoint<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinity">affinity</a></b></td>
        <td>object</td>
        <td>
          If specified, indicates the pod's scheduling constraints<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>componentsAllowed</b></td>
        <td>map[string][]string</td>
        <td>
          ComponentsAllowed is a list of allowed OpenTelemetry components for each pipeline type (receiver, processor, etc.)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          ENV vars to set on the OpAMPBridge Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecenvfromindex">envFrom</a></b></td>
        <td>[]object</td>
        <td>
          List of sources to populate environment variables on the OpAMPBridge Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>headers</b></td>
        <td>map[string]string</td>
        <td>
          Headers is an optional map of headers to use when connecting to the OpAMP Server, typically used to set access tokens or other authorization headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostNetwork</b></td>
        <td>boolean</td>
        <td>
          HostNetwork indicates if the pod should run in the host networking namespace.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image indicates the container image to use for the OpAMPBridge.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>imagePullPolicy</b></td>
        <td>string</td>
        <td>
          ImagePullPolicy indicates the pull policy to be used for retrieving the container image (Always, Never, IfNotPresent)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          NodeSelector to schedule OpAMPBridge pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>podAnnotations</b></td>
        <td>map[string]string</td>
        <td>
          PodAnnotations is the set of annotations that will be attached to OpAMPBridge pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecpodsecuritycontext">podSecurityContext</a></b></td>
        <td>object</td>
        <td>
          PodSecurityContext will be set as the pod security context.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecportsindex">ports</a></b></td>
        <td>[]object</td>
        <td>
          Ports allows a set of ports to be exposed by the underlying v1.Service.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>
          If specified, indicates the pod's priority. If not specified, the pod priority will be default or zero if there is no default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas is the number of pod instances for the OpAMPBridge.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Maximum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources to set on the OpAMPBridge pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecsecuritycontext">securityContext</a></b></td>
        <td>object</td>
        <td>
          SecurityContext will be set as the container security context.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>serviceAccount</b></td>
        <td>string</td>
        <td>
          ServiceAccount indicates the name of an existing service account to use with this instance. When set, the operator will not automatically create a ServiceAccount for the OpAMPBridge.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespectolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Toleration to schedule OpAMPBridge pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespectopologyspreadconstraintsindex">topologySpreadConstraints</a></b></td>
        <td>[]object</td>
        <td>
          TopologySpreadConstraints embedded kubernetes pod configuration option, controls how pods are spread across your cluster among failure-domains such as regions, zones, nodes, and other user-defined top<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>upgradeStrategy</b></td>
        <td>enum</td>
        <td>
          UpgradeStrategy represents how the operator will handle upgrades to the CR when a newer version of the operator is deployed<br/>
          <br/>
            <i>Enum</i>: automatic, none<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumemountsindex">volumeMounts</a></b></td>
        <td>[]object</td>
        <td>
          VolumeMounts represents the mount points to use in the underlying OpAMPBridge deployment(s)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindex">volumes</a></b></td>
        <td>[]object</td>
        <td>
          Volumes represents which volumes to use in the underlying OpAMPBridge deployment(s).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



If specified, indicates the pod's scheduling constraints

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes node affinity scheduling rules for the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.nodeAffinity
<sup><sup>[↩ Parent](#opampbridgespecaffinity)</sup></sup>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>
          If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitynodeaffinity)</sup></sup>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>
          A node selector term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
<sup><sup>[↩ Parent](#opampbridgespecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
<sup><sup>[↩ Parent](#opampbridgespecaffinitynodeaffinity)</sup></sup>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>
          Required. A list of node selector terms. The terms are ORed.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitynodeaffinityrequiredduringschedulingignoredduringexecution)</sup></sup>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity
<sup><sup>[↩ Parent](#opampbridgespecaffinity)</sup></sup>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-locate

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity
<sup><sup>[↩ Parent](#opampbridgespecaffinity)</sup></sup>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-locate

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.env[index]
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.env[index].valueFrom
<sup><sup>[↩ Parent](#opampbridgespecenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#opampbridgespecenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#opampbridgespecenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#opampbridgespecenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#opampbridgespecenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.envFrom[index]
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



EnvFromSource represents the source of a set of ConfigMaps

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecenvfromindexconfigmapref">configMapRef</a></b></td>
        <td>object</td>
        <td>
          The ConfigMap to select from<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>prefix</b></td>
        <td>string</td>
        <td>
          An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecenvfromindexsecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          The Secret to select from<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.envFrom[index].configMapRef
<sup><sup>[↩ Parent](#opampbridgespecenvfromindex)</sup></sup>



The ConfigMap to select from

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.envFrom[index].secretRef
<sup><sup>[↩ Parent](#opampbridgespecenvfromindex)</sup></sup>



The Secret to select from

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.podSecurityContext
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



PodSecurityContext will be set as the pod security context.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fsGroup</b></td>
        <td>integer</td>
        <td>
          A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod: 
 1.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>fsGroupChangePolicy</b></td>
        <td>string</td>
        <td>
          fsGroupChangePolicy defines behavior of changing ownership and permission of the volume before being exposed inside Pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsGroup</b></td>
        <td>integer</td>
        <td>
          The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsNonRoot</b></td>
        <td>boolean</td>
        <td>
          Indicates that the container must run as a non-root user.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUser</b></td>
        <td>integer</td>
        <td>
          The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecpodsecuritycontextselinuxoptions">seLinuxOptions</a></b></td>
        <td>object</td>
        <td>
          The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecpodsecuritycontextseccompprofile">seccompProfile</a></b></td>
        <td>object</td>
        <td>
          The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>supplementalGroups</b></td>
        <td>[]integer</td>
        <td>
          A list of groups applied to the first process run in each container, in addition to the container's primary GID, the fsGroup (if specified), and group memberships defined in the container image for th<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecpodsecuritycontextsysctlsindex">sysctls</a></b></td>
        <td>[]object</td>
        <td>
          Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported sysctls (by the container runtime) might fail to launch. Note that this field cannot be set when spec.os.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecpodsecuritycontextwindowsoptions">windowsOptions</a></b></td>
        <td>object</td>
        <td>
          The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.podSecurityContext.seLinuxOptions
<sup><sup>[↩ Parent](#opampbridgespecpodsecuritycontext)</sup></sup>



The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>level</b></td>
        <td>string</td>
        <td>
          Level is SELinux level label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>role</b></td>
        <td>string</td>
        <td>
          Role is a SELinux role label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type is a SELinux type label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          User is a SELinux user label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.podSecurityContext.seccompProfile
<sup><sup>[↩ Parent](#opampbridgespecpodsecuritycontext)</sup></sup>



The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type indicates which kind of seccomp profile will be applied. Valid options are: 
 Localhost - a profile defined in a file on the node should be used.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>localhostProfile</b></td>
        <td>string</td>
        <td>
          localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.podSecurityContext.sysctls[index]
<sup><sup>[↩ Parent](#opampbridgespecpodsecuritycontext)</sup></sup>



Sysctl defines a kernel parameter to be set

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of a property to set<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value of a property to set<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.podSecurityContext.windowsOptions
<sup><sup>[↩ Parent](#opampbridgespecpodsecuritycontext)</sup></sup>



The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>gmsaCredentialSpec</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gmsaCredentialSpecName</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpecName is the name of the GMSA credential spec to use.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostProcess</b></td>
        <td>boolean</td>
        <td>
          HostProcess determines if a container should be run as a 'Host Process' container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUserName</b></td>
        <td>string</td>
        <td>
          The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.ports[index]
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



ServicePort contains information on service's port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          The port that will be exposed by this service.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>appProtocol</b></td>
        <td>string</td>
        <td>
          The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand. This field follows standard Kubernetes label syntax.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The name of this port within the service. This must be a DNS_LABEL. All ports within a ServiceSpec must have unique names.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodePort</b></td>
        <td>integer</td>
        <td>
          The port on each node on which this service is exposed when type is NodePort or LoadBalancer.  Usually assigned by the system.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          The IP protocol for this port. Supports "TCP", "UDP", and "SCTP". Default is TCP.<br/>
          <br/>
            <i>Default</i>: TCP<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetPort</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.resources
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



Resources to set on the OpAMPBridge pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.resources.claims[index]
<sup><sup>[↩ Parent](#opampbridgespecresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.securityContext
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



SecurityContext will be set as the container security context.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>allowPrivilegeEscalation</b></td>
        <td>boolean</td>
        <td>
          AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecsecuritycontextcapabilities">capabilities</a></b></td>
        <td>object</td>
        <td>
          The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>privileged</b></td>
        <td>boolean</td>
        <td>
          Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>procMount</b></td>
        <td>string</td>
        <td>
          procMount denotes the type of proc mount to use for the containers. The default is DefaultProcMount which uses the container runtime defaults for readonly paths and masked paths.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnlyRootFilesystem</b></td>
        <td>boolean</td>
        <td>
          Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsGroup</b></td>
        <td>integer</td>
        <td>
          The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsNonRoot</b></td>
        <td>boolean</td>
        <td>
          Indicates that the container must run as a non-root user.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUser</b></td>
        <td>integer</td>
        <td>
          The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecsecuritycontextselinuxoptions">seLinuxOptions</a></b></td>
        <td>object</td>
        <td>
          The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecsecuritycontextseccompprofile">seccompProfile</a></b></td>
        <td>object</td>
        <td>
          The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecsecuritycontextwindowsoptions">windowsOptions</a></b></td>
        <td>object</td>
        <td>
          The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.securityContext.capabilities
<sup><sup>[↩ Parent](#opampbridgespecsecuritycontext)</sup></sup>



The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>add</b></td>
        <td>[]string</td>
        <td>
          Added capabilities<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>drop</b></td>
        <td>[]string</td>
        <td>
          Removed capabilities<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.securityContext.seLinuxOptions
<sup><sup>[↩ Parent](#opampbridgespecsecuritycontext)</sup></sup>



The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>level</b></td>
        <td>string</td>
        <td>
          Level is SELinux level label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>role</b></td>
        <td>string</td>
        <td>
          Role is a SELinux role label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type is a SELinux type label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          User is a SELinux user label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.securityContext.seccompProfile
<sup><sup>[↩ Parent](#opampbridgespecsecuritycontext)</sup></sup>



The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type indicates which kind of seccomp profile will be applied. Valid options are: 
 Localhost - a profile defined in a file on the node should be used.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>localhostProfile</b></td>
        <td>string</td>
        <td>
          localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.securityContext.windowsOptions
<sup><sup>[↩ Parent](#opampbridgespecsecuritycontext)</sup></sup>



The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>gmsaCredentialSpec</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gmsaCredentialSpecName</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpecName is the name of the GMSA credential spec to use.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostProcess</b></td>
        <td>boolean</td>
        <td>
          HostProcess determines if a container should be run as a 'Host Process' container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUserName</b></td>
        <td>string</td>
        <td>
          The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.tolerations[index]
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.topologySpreadConstraints[index]
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



TopologySpreadConstraint specifies how to spread matching pods among the given topology.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxSkew</b></td>
        <td>integer</td>
        <td>
          MaxSkew describes the degree to which pods may be unevenly distributed.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>whenUnsatisfiable</b></td>
        <td>string</td>
        <td>
          WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespectopologyspreadconstraintsindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select the pods over which spreading will be calculated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minDomains</b></td>
        <td>integer</td>
        <td>
          MinDomains indicates a minimum number of eligible domains.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeAffinityPolicy</b></td>
        <td>string</td>
        <td>
          NodeAffinityPolicy indicates how we will treat Pod's nodeAffinity/nodeSelector when calculating pod topology spread skew.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeTaintsPolicy</b></td>
        <td>string</td>
        <td>
          NodeTaintsPolicy indicates how we will treat node taints when calculating pod topology spread skew.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.topologySpreadConstraints[index].labelSelector
<sup><sup>[↩ Parent](#opampbridgespectopologyspreadconstraintsindex)</sup></sup>



LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespectopologyspreadconstraintsindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.topologySpreadConstraints[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespectopologyspreadconstraintsindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumeMounts[index]
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



VolumeMount describes a mounting of a Volume within a container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mountPath</b></td>
        <td>string</td>
        <td>
          Path within the container at which the volume should be mounted.  Must not contain ':'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          This must match the Name of a Volume.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mountPropagation</b></td>
        <td>string</td>
        <td>
          mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subPath</b></td>
        <td>string</td>
        <td>
          Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subPathExpr</b></td>
        <td>string</td>
        <td>
          Expanded path within the volume from which the container's volume should be mounted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index]
<sup><sup>[↩ Parent](#opampbridgespec)</sup></sup>



Volume represents a named volume in a pod that may be accessed by any container in the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          name of the volume. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexawselasticblockstore">awsElasticBlockStore</a></b></td>
        <td>object</td>
        <td>
          awsElasticBlockStore represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexazuredisk">azureDisk</a></b></td>
        <td>object</td>
        <td>
          azureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexazurefile">azureFile</a></b></td>
        <td>object</td>
        <td>
          azureFile represents an Azure File Service mount on the host and bind mount to the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexcephfs">cephfs</a></b></td>
        <td>object</td>
        <td>
          cephFS represents a Ceph FS mount on the host that shares a pod's lifetime<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexcinder">cinder</a></b></td>
        <td>object</td>
        <td>
          cinder represents a cinder volume attached and mounted on kubelets host machine. More info: https://examples.k8s.io/mysql-cinder-pd/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>
          configMap represents a configMap that should populate this volume<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexcsi">csi</a></b></td>
        <td>object</td>
        <td>
          csi (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers (Beta feature).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>
          downwardAPI represents downward API about the pod that should populate this volume<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexemptydir">emptyDir</a></b></td>
        <td>object</td>
        <td>
          emptyDir represents a temporary directory that shares a pod's lifetime. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeral">ephemeral</a></b></td>
        <td>object</td>
        <td>
          ephemeral represents a volume that is handled by a cluster storage driver.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexfc">fc</a></b></td>
        <td>object</td>
        <td>
          fc represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexflexvolume">flexVolume</a></b></td>
        <td>object</td>
        <td>
          flexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexflocker">flocker</a></b></td>
        <td>object</td>
        <td>
          flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexgcepersistentdisk">gcePersistentDisk</a></b></td>
        <td>object</td>
        <td>
          gcePersistentDisk represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexgitrepo">gitRepo</a></b></td>
        <td>object</td>
        <td>
          gitRepo represents a git repository at a particular revision. DEPRECATED: GitRepo is deprecated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexglusterfs">glusterfs</a></b></td>
        <td>object</td>
        <td>
          glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/glusterfs/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexhostpath">hostPath</a></b></td>
        <td>object</td>
        <td>
          hostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexiscsi">iscsi</a></b></td>
        <td>object</td>
        <td>
          iscsi represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://examples.k8s.io/volumes/iscsi/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexnfs">nfs</a></b></td>
        <td>object</td>
        <td>
          nfs represents an NFS mount on the host that shares a pod's lifetime More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexpersistentvolumeclaim">persistentVolumeClaim</a></b></td>
        <td>object</td>
        <td>
          persistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexphotonpersistentdisk">photonPersistentDisk</a></b></td>
        <td>object</td>
        <td>
          photonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexportworxvolume">portworxVolume</a></b></td>
        <td>object</td>
        <td>
          portworxVolume represents a portworx volume attached and mounted on kubelets host machine<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojected">projected</a></b></td>
        <td>object</td>
        <td>
          projected items for all in one resources secrets, configmaps, and downward API<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexquobyte">quobyte</a></b></td>
        <td>object</td>
        <td>
          quobyte represents a Quobyte mount on the host that shares a pod's lifetime<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexrbd">rbd</a></b></td>
        <td>object</td>
        <td>
          rbd represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/rbd/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexscaleio">scaleIO</a></b></td>
        <td>object</td>
        <td>
          scaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexsecret">secret</a></b></td>
        <td>object</td>
        <td>
          secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexstorageos">storageos</a></b></td>
        <td>object</td>
        <td>
          storageOS represents a StorageOS volume attached and mounted on Kubernetes nodes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexvspherevolume">vsphereVolume</a></b></td>
        <td>object</td>
        <td>
          vsphereVolume represents a vSphere volume attached and mounted on kubelets host machine<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].awsElasticBlockStore
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



awsElasticBlockStore represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>volumeID</b></td>
        <td>string</td>
        <td>
          volumeID is unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>partition</b></td>
        <td>integer</td>
        <td>
          partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1".<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly value true will force the readOnly setting in VolumeMounts. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].azureDisk
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



azureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>diskName</b></td>
        <td>string</td>
        <td>
          diskName is the Name of the data disk in the blob storage<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>diskURI</b></td>
        <td>string</td>
        <td>
          diskURI is the URI of data disk in the blob storage<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>cachingMode</b></td>
        <td>string</td>
        <td>
          cachingMode is the Host Caching mode: None, Read Only, Read Write.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          kind expected values are Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].azureFile
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



azureFile represents an Azure File Service mount on the host and bind mount to the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>secretName</b></td>
        <td>string</td>
        <td>
          secretName is the  name of secret that contains Azure Storage Account Name and Key<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>shareName</b></td>
        <td>string</td>
        <td>
          shareName is the azure share Name<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].cephfs
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



cephFS represents a Ceph FS mount on the host that shares a pod's lifetime

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>monitors</b></td>
        <td>[]string</td>
        <td>
          monitors is Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is Optional: Used as the mounted root, rather than the full Ceph tree, default is /<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secretFile</b></td>
        <td>string</td>
        <td>
          secretFile is Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexcephfssecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          user is optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].cephfs.secretRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexcephfs)</sup></sup>



secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].cinder
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



cinder represents a cinder volume attached and mounted on kubelets host machine. More info: https://examples.k8s.io/mysql-cinder-pd/README.md

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>volumeID</b></td>
        <td>string</td>
        <td>
          volumeID used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexcindersecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is optional: points to a secret object containing parameters used to connect to OpenStack.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].cinder.secretRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexcinder)</sup></sup>



secretRef is optional: points to a secret object containing parameters used to connect to OpenStack.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].configMap
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



configMap represents a configMap that should populate this volume

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultMode</b></td>
        <td>integer</td>
        <td>
          defaultMode is optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          items if unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          optional specify whether the ConfigMap or its keys must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].configMap.items[index]
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexconfigmap)</sup></sup>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the key to project.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].csi
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



csi (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers (Beta feature).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>driver</b></td>
        <td>string</td>
        <td>
          driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType to mount. Ex. "ext4", "xfs", "ntfs". If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexcsinodepublishsecretref">nodePublishSecretRef</a></b></td>
        <td>object</td>
        <td>
          nodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly specifies a read-only configuration for the volume. Defaults to false (read/write).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeAttributes</b></td>
        <td>map[string]string</td>
        <td>
          volumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver's documentation for supported values.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].csi.nodePublishSecretRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexcsi)</sup></sup>



nodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].downwardAPI
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



downwardAPI represents downward API about the pod that should populate this volume

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultMode</b></td>
        <td>integer</td>
        <td>
          Optional: mode bits to use on created files by default. Must be a Optional: mode bits used to set permissions on created files by default.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          Items is a list of downward API volume file<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].downwardAPI.items[index]
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexdownwardapi)</sup></sup>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].downwardAPI.items[index].fieldRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexdownwardapiitemsindex)</sup></sup>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].downwardAPI.items[index].resourceFieldRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexdownwardapiitemsindex)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].emptyDir
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



emptyDir represents a temporary directory that shares a pod's lifetime. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>medium</b></td>
        <td>string</td>
        <td>
          medium represents what type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sizeLimit</b></td>
        <td>int or string</td>
        <td>
          sizeLimit is the total amount of local storage required for this EmptyDir volume. The size limit is also applicable for memory medium.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



ephemeral represents a volume that is handled by a cluster storage driver.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeralvolumeclaimtemplate">volumeClaimTemplate</a></b></td>
        <td>object</td>
        <td>
          Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC, i.e.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral.volumeClaimTemplate
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexephemeral)</sup></sup>



Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC, i.e.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespec">spec</a></b></td>
        <td>object</td>
        <td>
          The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeralvolumeclaimtemplatemetadata">metadata</a></b></td>
        <td>object</td>
        <td>
          May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during validation.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral.volumeClaimTemplate.spec
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexephemeralvolumeclaimtemplate)</sup></sup>



The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>
          accessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespecdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>
          dataSource field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespecdatasourceref">dataSourceRef</a></b></td>
        <td>object</td>
        <td>
          dataSourceRef specifies the object from which to populate the volume with data, if a non-empty volume is desired.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespecresources">resources</a></b></td>
        <td>object</td>
        <td>
          resources represents the minimum resources the volume should have.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespecselector">selector</a></b></td>
        <td>object</td>
        <td>
          selector is a label query over volumes to consider for binding.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>
          storageClassName is the name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>
          volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>
          volumeName is the binding reference to the PersistentVolume backing this claim.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.dataSource
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespec)</sup></sup>



dataSource field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind is the type of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>
          APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.dataSourceRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespec)</sup></sup>



dataSourceRef specifies the object from which to populate the volume with data, if a non-empty volume is desired.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind is the type of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>
          APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace is the namespace of resource being referenced Note that when a namespace is specified, a gateway.networking.k8s.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.resources
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespec)</sup></sup>



resources represents the minimum resources the volume should have.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespecresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.resources.claims[index]
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespecresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.selector
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespec)</sup></sup>



selector is a label query over volumes to consider for binding.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespecselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexephemeralvolumeclaimtemplatespecselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].ephemeral.volumeClaimTemplate.metadata
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexephemeralvolumeclaimtemplate)</sup></sup>



May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during validation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>finalizers</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].fc
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



fc represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lun</b></td>
        <td>integer</td>
        <td>
          lun is Optional: FC target lun number<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetWWNs</b></td>
        <td>[]string</td>
        <td>
          targetWWNs is Optional: FC target worldwide names (WWNs)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>wwids</b></td>
        <td>[]string</td>
        <td>
          wwids Optional: FC volume world wide identifiers (wwids) Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].flexVolume
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



flexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>driver</b></td>
        <td>string</td>
        <td>
          driver is the name of the driver to use for this volume.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>options</b></td>
        <td>map[string]string</td>
        <td>
          options is Optional: this field holds extra command options if any.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly is Optional: defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexflexvolumesecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is Optional: secretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].flexVolume.secretRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexflexvolume)</sup></sup>



secretRef is Optional: secretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].flocker
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>datasetName</b></td>
        <td>string</td>
        <td>
          datasetName is Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>datasetUUID</b></td>
        <td>string</td>
        <td>
          datasetUUID is the UUID of the dataset. This is unique identifier of a Flocker dataset<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].gcePersistentDisk
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



gcePersistentDisk represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>pdName</b></td>
        <td>string</td>
        <td>
          pdName is unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>partition</b></td>
        <td>integer</td>
        <td>
          partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1".<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].gitRepo
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



gitRepo represents a git repository at a particular revision. DEPRECATED: GitRepo is deprecated.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>repository</b></td>
        <td>string</td>
        <td>
          repository is the URL<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>directory</b></td>
        <td>string</td>
        <td>
          directory is the target directory name. Must not contain or start with '..'.  If '.' is supplied, the volume directory will be the git repository.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>revision</b></td>
        <td>string</td>
        <td>
          revision is the commit hash for the specified revision.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].glusterfs
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/glusterfs/README.md

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>endpoints</b></td>
        <td>string</td>
        <td>
          endpoints is the endpoint name that details Glusterfs topology. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the Glusterfs volume path. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].hostPath
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



hostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path of the directory on the host. If the path is a symlink, it will follow the link to the real path. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type for HostPath Volume Defaults to "" More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].iscsi
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



iscsi represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://examples.k8s.io/volumes/iscsi/README.md

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>iqn</b></td>
        <td>string</td>
        <td>
          iqn is the target iSCSI Qualified Name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>lun</b></td>
        <td>integer</td>
        <td>
          lun represents iSCSI Target Lun number.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>targetPortal</b></td>
        <td>string</td>
        <td>
          targetPortal is iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>chapAuthDiscovery</b></td>
        <td>boolean</td>
        <td>
          chapAuthDiscovery defines whether support iSCSI Discovery CHAP authentication<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>chapAuthSession</b></td>
        <td>boolean</td>
        <td>
          chapAuthSession defines whether support iSCSI Session CHAP authentication<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initiatorName</b></td>
        <td>string</td>
        <td>
          initiatorName is the custom iSCSI Initiator Name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>iscsiInterface</b></td>
        <td>string</td>
        <td>
          iscsiInterface is the interface Name that uses an iSCSI transport. Defaults to 'default' (tcp).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>portals</b></td>
        <td>[]string</td>
        <td>
          portals is the iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexiscsisecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is the CHAP Secret for iSCSI target and initiator authentication<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].iscsi.secretRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexiscsi)</sup></sup>



secretRef is the CHAP Secret for iSCSI target and initiator authentication

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].nfs
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



nfs represents an NFS mount on the host that shares a pod's lifetime More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path that is exported by the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>server</b></td>
        <td>string</td>
        <td>
          server is the hostname or IP address of the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the NFS export to be mounted with read-only permissions. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].persistentVolumeClaim
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



persistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. More info: https://kubernetes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>claimName</b></td>
        <td>string</td>
        <td>
          claimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly Will force the ReadOnly setting in VolumeMounts. Default false.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].photonPersistentDisk
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



photonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>pdID</b></td>
        <td>string</td>
        <td>
          pdID is the ID that identifies Photon Controller persistent disk<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].portworxVolume
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



portworxVolume represents a portworx volume attached and mounted on kubelets host machine

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>volumeID</b></td>
        <td>string</td>
        <td>
          volumeID uniquely identifies a Portworx volume<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fSType represents the filesystem type to mount Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



projected items for all in one resources secrets, configmaps, and downward API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultMode</b></td>
        <td>integer</td>
        <td>
          defaultMode are the mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindex">sources</a></b></td>
        <td>[]object</td>
        <td>
          sources is the list of volume projections<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index]
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojected)</sup></sup>



Projection that may be projected along with other supported volume types

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>
          configMap information about the configMap data to project<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>
          downwardAPI information about the downwardAPI data to project<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindexsecret">secret</a></b></td>
        <td>object</td>
        <td>
          secret information about the secret data to project<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindexserviceaccounttoken">serviceAccountToken</a></b></td>
        <td>object</td>
        <td>
          serviceAccountToken is information about the serviceAccountToken data to project<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index].configMap
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojectedsourcesindex)</sup></sup>



configMap information about the configMap data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          items if unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          optional specify whether the ConfigMap or its keys must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index].configMap.items[index]
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojectedsourcesindexconfigmap)</sup></sup>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the key to project.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index].downwardAPI
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojectedsourcesindex)</sup></sup>



downwardAPI information about the downwardAPI data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          Items is a list of DownwardAPIVolume file<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index].downwardAPI.items[index]
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojectedsourcesindexdownwardapi)</sup></sup>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index].downwardAPI.items[index].fieldRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojectedsourcesindexdownwardapiitemsindex)</sup></sup>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index].downwardAPI.items[index].resourceFieldRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojectedsourcesindexdownwardapiitemsindex)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index].secret
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojectedsourcesindex)</sup></sup>



secret information about the secret data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opampbridgespecvolumesindexprojectedsourcesindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          items if unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          optional field specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index].secret.items[index]
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojectedsourcesindexsecret)</sup></sup>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the key to project.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].projected.sources[index].serviceAccountToken
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexprojectedsourcesindex)</sup></sup>



serviceAccountToken is information about the serviceAccountToken data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the path relative to the mount point of the file to project the token into.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>audience</b></td>
        <td>string</td>
        <td>
          audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>expirationSeconds</b></td>
        <td>integer</td>
        <td>
          expirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].quobyte
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



quobyte represents a Quobyte mount on the host that shares a pod's lifetime

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>registry</b></td>
        <td>string</td>
        <td>
          registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>volume</b></td>
        <td>string</td>
        <td>
          volume is a string that references an already created Quobyte volume by name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>group</b></td>
        <td>string</td>
        <td>
          group to map volume access to Default is no group<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tenant</b></td>
        <td>string</td>
        <td>
          tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          user to map volume access to Defaults to serivceaccount user<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].rbd
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



rbd represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/rbd/README.md

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          image is the rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>monitors</b></td>
        <td>[]string</td>
        <td>
          monitors is a collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>keyring</b></td>
        <td>string</td>
        <td>
          keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>pool</b></td>
        <td>string</td>
        <td>
          pool is the rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexrbdsecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          user is the rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].rbd.secretRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexrbd)</sup></sup>



secretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].scaleIO
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



scaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>gateway</b></td>
        <td>string</td>
        <td>
          gateway is the host address of the ScaleIO API Gateway.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexscaleiosecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>system</b></td>
        <td>string</td>
        <td>
          system is the name of the storage system as configured in ScaleIO.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Default is "xfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protectionDomain</b></td>
        <td>string</td>
        <td>
          protectionDomain is the name of the ScaleIO Protection Domain for the configured storage.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sslEnabled</b></td>
        <td>boolean</td>
        <td>
          sslEnabled Flag enable/disable SSL communication with Gateway, default false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageMode</b></td>
        <td>string</td>
        <td>
          storageMode indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storagePool</b></td>
        <td>string</td>
        <td>
          storagePool is the ScaleIO Storage Pool associated with the protection domain.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>
          volumeName is the name of a volume already created in the ScaleIO system that is associated with this volume source.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].scaleIO.secretRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexscaleio)</sup></sup>



secretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].secret
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultMode</b></td>
        <td>integer</td>
        <td>
          defaultMode is Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          items If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          optional field specify whether the Secret or its keys must be defined<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secretName</b></td>
        <td>string</td>
        <td>
          secretName is the name of the secret in the pod's namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].secret.items[index]
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexsecret)</sup></sup>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the key to project.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].storageos
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



storageOS represents a StorageOS volume attached and mounted on Kubernetes nodes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opampbridgespecvolumesindexstorageossecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef specifies the secret to use for obtaining the StorageOS API credentials.  If not specified, default values will be attempted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>
          volumeName is the human-readable name of the StorageOS volume.  Volume names are only unique within a namespace.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeNamespace</b></td>
        <td>string</td>
        <td>
          volumeNamespace specifies the scope of the volume within StorageOS.  If no namespace is specified then the Pod's namespace will be used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].storageos.secretRef
<sup><sup>[↩ Parent](#opampbridgespecvolumesindexstorageos)</sup></sup>



secretRef specifies the secret to use for obtaining the StorageOS API credentials.  If not specified, default values will be attempted.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.spec.volumes[index].vsphereVolume
<sup><sup>[↩ Parent](#opampbridgespecvolumesindex)</sup></sup>



vsphereVolume represents a vSphere volume attached and mounted on kubelets host machine

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>volumePath</b></td>
        <td>string</td>
        <td>
          volumePath is the path that identifies vSphere volume vmdk<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storagePolicyID</b></td>
        <td>string</td>
        <td>
          storagePolicyID is the storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storagePolicyName</b></td>
        <td>string</td>
        <td>
          storagePolicyName is the storage Policy Based Management (SPBM) profile name.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpAMPBridge.status
<sup><sup>[↩ Parent](#opampbridge)</sup></sup>



OpAMPBridgeStatus defines the observed state of OpAMPBridge.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>version</b></td>
        <td>string</td>
        <td>
          Version of the managed OpAMP Bridge (operand)<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## OpenTelemetryCollector
<sup><sup>[↩ Parent](#opentelemetryiov1alpha1 )</sup></sup>






OpenTelemetryCollector is the Schema for the opentelemetrycollectors API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>opentelemetry.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>OpenTelemetryCollector</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspec">spec</a></b></td>
        <td>object</td>
        <td>
          OpenTelemetryCollectorSpec defines the desired state of OpenTelemetryCollector.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorstatus">status</a></b></td>
        <td>object</td>
        <td>
          OpenTelemetryCollectorStatus defines the observed state of OpenTelemetryCollector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec
<sup><sup>[↩ Parent](#opentelemetrycollector)</sup></sup>



OpenTelemetryCollectorSpec defines the desired state of OpenTelemetryCollector.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindex">additionalContainers</a></b></td>
        <td>[]object</td>
        <td>
          AdditionalContainers allows injecting additional containers into the Collector's pod definition.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinity">affinity</a></b></td>
        <td>object</td>
        <td>
          If specified, indicates the pod's scheduling constraints<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>args</b></td>
        <td>map[string]string</td>
        <td>
          Args is the set of arguments to pass to the OpenTelemetry Collector binary<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscaler">autoscaler</a></b></td>
        <td>object</td>
        <td>
          Autoscaler specifies the pod autoscaling configuration to use for the OpenTelemetryCollector workload.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>config</b></td>
        <td>string</td>
        <td>
          Config is the raw JSON to be used as the collector's configuration. Refer to the OpenTelemetry Collector documentation for details.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecconfigmapsindex">configmaps</a></b></td>
        <td>[]object</td>
        <td>
          ConfigMaps is a list of ConfigMaps in the same namespace as the OpenTelemetryCollector object, which shall be mounted into the Collector Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          ENV vars to set on the OpenTelemetry Collector's Pods. These can then in certain cases be consumed in the config file for the Collector.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecenvfromindex">envFrom</a></b></td>
        <td>[]object</td>
        <td>
          List of sources to populate environment variables on the OpenTelemetry Collector's Pods. These can then in certain cases be consumed in the config file for the Collector.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostNetwork</b></td>
        <td>boolean</td>
        <td>
          HostNetwork indicates if the pod should run in the host networking namespace.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image indicates the container image to use for the OpenTelemetry Collector.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>imagePullPolicy</b></td>
        <td>string</td>
        <td>
          ImagePullPolicy indicates the pull policy to be used for retrieving the container image (Always, Never, IfNotPresent)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecingress">ingress</a></b></td>
        <td>object</td>
        <td>
          Ingress is used to specify how OpenTelemetry Collector is exposed. This functionality is only available if one of the valid modes is set. Valid modes are: deployment, daemonset and statefulset.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindex">initContainers</a></b></td>
        <td>[]object</td>
        <td>
          InitContainers allows injecting initContainers to the Collector's pod definition.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecycle">lifecycle</a></b></td>
        <td>object</td>
        <td>
          Actions that the management system should take in response to container lifecycle events. Cannot be updated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspeclivenessprobe">livenessProbe</a></b></td>
        <td>object</td>
        <td>
          Liveness config for the OpenTelemetry Collector except the probe handler which is auto generated from the health extension of the collector.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>managementState</b></td>
        <td>enum</td>
        <td>
          ManagementState defines if the CR should be managed by the operator or not. Default is managed.<br/>
          <br/>
            <i>Enum</i>: managed, unmanaged<br/>
            <i>Default</i>: managed<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxReplicas</b></td>
        <td>integer</td>
        <td>
          MaxReplicas sets an upper bound to the autoscaling feature. If MaxReplicas is set autoscaling is enabled. Deprecated: use "OpenTelemetryCollector.Spec.Autoscaler.MaxReplicas" instead.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minReplicas</b></td>
        <td>integer</td>
        <td>
          MinReplicas sets a lower bound to the autoscaling feature.  Set this if you are using autoscaling. It must be at least 1 Deprecated: use "OpenTelemetryCollector.Spec.Autoscaler.MinReplicas" instead.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          Mode represents how the collector should be deployed (deployment, daemonset, statefulset or sidecar)<br/>
          <br/>
            <i>Enum</i>: daemonset, deployment, sidecar, statefulset<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          NodeSelector to schedule OpenTelemetry Collector pods. This is only relevant to daemonset, statefulset, and deployment mode<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecobservability">observability</a></b></td>
        <td>object</td>
        <td>
          ObservabilitySpec defines how telemetry data gets handled.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>podAnnotations</b></td>
        <td>map[string]string</td>
        <td>
          PodAnnotations is the set of annotations that will be attached to Collector and Target Allocator pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecpoddisruptionbudget">podDisruptionBudget</a></b></td>
        <td>object</td>
        <td>
          PodDisruptionBudget specifies the pod disruption budget configuration to use for the OpenTelemetryCollector workload.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecpodsecuritycontext">podSecurityContext</a></b></td>
        <td>object</td>
        <td>
          PodSecurityContext configures the pod security context for the opentelemetry-collector pod, when running as a deployment, daemonset, or statefulset.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecportsindex">ports</a></b></td>
        <td>[]object</td>
        <td>
          Ports allows a set of ports to be exposed by the underlying v1.Service. By default, the operator will attempt to infer the required ports by parsing the .Spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>
          If specified, indicates the pod's priority. If not specified, the pod priority will be default or zero if there is no default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas is the number of pod instances for the underlying OpenTelemetry Collector. Set this if your are not using autoscaling<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources to set on the OpenTelemetry Collector pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecsecuritycontext">securityContext</a></b></td>
        <td>object</td>
        <td>
          SecurityContext configures the container security context for the opentelemetry-collector container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>serviceAccount</b></td>
        <td>string</td>
        <td>
          ServiceAccount indicates the name of an existing service account to use with this instance. When set, the operator will not automatically create a ServiceAccount for the collector.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocator">targetAllocator</a></b></td>
        <td>object</td>
        <td>
          TargetAllocator indicates a value which determines whether to spawn a target allocation resource or not.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationGracePeriodSeconds</b></td>
        <td>integer</td>
        <td>
          Duration in seconds the pod needs to terminate gracefully upon probe failure.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Toleration to schedule OpenTelemetry Collector pods. This is only relevant to daemonset, statefulset, and deployment mode<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectopologyspreadconstraintsindex">topologySpreadConstraints</a></b></td>
        <td>[]object</td>
        <td>
          TopologySpreadConstraints embedded kubernetes pod configuration option, controls how pods are spread across your cluster among failure-domains such as regions, zones, nodes, and other user-defined top<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecupdatestrategy">updateStrategy</a></b></td>
        <td>object</td>
        <td>
          UpdateStrategy represents the strategy the operator will take replacing existing DaemonSet pods with new pods https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>upgradeStrategy</b></td>
        <td>enum</td>
        <td>
          UpgradeStrategy represents how the operator will handle upgrades to the CR when a newer version of the operator is deployed<br/>
          <br/>
            <i>Enum</i>: automatic, none<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindex">volumeClaimTemplates</a></b></td>
        <td>[]object</td>
        <td>
          VolumeClaimTemplates will provide stable storage using PersistentVolumes. Only available when the mode=statefulset.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumemountsindex">volumeMounts</a></b></td>
        <td>[]object</td>
        <td>
          VolumeMounts represents the mount points to use in the underlying collector deployment(s)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindex">volumes</a></b></td>
        <td>[]object</td>
        <td>
          Volumes represents which volumes to use in the underlying collector deployment(s).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



A single application container that you want to run within a pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>args</b></td>
        <td>[]string</td>
        <td>
          Arguments to the entrypoint. The container image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Entrypoint array. Not executed within a shell. The container image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          List of environment variables to set in the container. Cannot be updated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexenvfromindex">envFrom</a></b></td>
        <td>[]object</td>
        <td>
          List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Container image name. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>imagePullPolicy</b></td>
        <td>string</td>
        <td>
          Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecycle">lifecycle</a></b></td>
        <td>object</td>
        <td>
          Actions that the management system should take in response to container lifecycle events. Cannot be updated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlivenessprobe">livenessProbe</a></b></td>
        <td>object</td>
        <td>
          Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexportsindex">ports</a></b></td>
        <td>[]object</td>
        <td>
          List of ports to expose from the container. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexreadinessprobe">readinessProbe</a></b></td>
        <td>object</td>
        <td>
          Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexresizepolicyindex">resizePolicy</a></b></td>
        <td>[]object</td>
        <td>
          Resources resize policy for the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexresources">resources</a></b></td>
        <td>object</td>
        <td>
          Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>restartPolicy</b></td>
        <td>string</td>
        <td>
          RestartPolicy defines the restart behavior of individual containers in a pod. This field may only be set for init containers, and the only allowed value is "Always".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexsecuritycontext">securityContext</a></b></td>
        <td>object</td>
        <td>
          SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexstartupprobe">startupProbe</a></b></td>
        <td>object</td>
        <td>
          StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>stdin</b></td>
        <td>boolean</td>
        <td>
          Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>stdinOnce</b></td>
        <td>boolean</td>
        <td>
          Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationMessagePath</b></td>
        <td>string</td>
        <td>
          Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationMessagePolicy</b></td>
        <td>string</td>
        <td>
          Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tty</b></td>
        <td>boolean</td>
        <td>
          Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexvolumedevicesindex">volumeDevices</a></b></td>
        <td>[]object</td>
        <td>
          volumeDevices is the list of block devices to be used by the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexvolumemountsindex">volumeMounts</a></b></td>
        <td>[]object</td>
        <td>
          Pod volumes to mount into the container's filesystem. Cannot be updated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>workingDir</b></td>
        <td>string</td>
        <td>
          Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].env[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].env[index].valueFrom
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].envFrom[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



EnvFromSource represents the source of a set of ConfigMaps

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexenvfromindexconfigmapref">configMapRef</a></b></td>
        <td>object</td>
        <td>
          The ConfigMap to select from<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>prefix</b></td>
        <td>string</td>
        <td>
          An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexenvfromindexsecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          The Secret to select from<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].envFrom[index].configMapRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexenvfromindex)</sup></sup>



The ConfigMap to select from

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].envFrom[index].secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexenvfromindex)</sup></sup>



The Secret to select from

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



Actions that the management system should take in response to container lifecycle events. Cannot be updated.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecyclepoststart">postStart</a></b></td>
        <td>object</td>
        <td>
          PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecycleprestop">preStop</a></b></td>
        <td>object</td>
        <td>
          PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.postStart
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecycle)</sup></sup>



PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecyclepoststartexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecyclepoststarthttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecyclepoststarttcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.postStart.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecyclepoststart)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.postStart.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecyclepoststart)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecyclepoststarthttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.postStart.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecyclepoststarthttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.postStart.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecyclepoststart)</sup></sup>



Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.preStop
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecycle)</sup></sup>



PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecycleprestopexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecycleprestophttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecycleprestoptcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.preStop.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecycleprestop)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.preStop.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecycleprestop)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlifecycleprestophttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.preStop.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecycleprestophttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].lifecycle.preStop.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlifecycleprestop)</sup></sup>



Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].livenessProbe
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlivenessprobeexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlivenessprobegrpc">grpc</a></b></td>
        <td>object</td>
        <td>
          GRPC specifies an action involving a GRPC port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlivenessprobehttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initialDelaySeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>periodSeconds</b></td>
        <td>integer</td>
        <td>
          How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>successThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlivenessprobetcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          TCPSocket specifies an action involving a TCP port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationGracePeriodSeconds</b></td>
        <td>integer</td>
        <td>
          Optional duration in seconds the pod needs to terminate gracefully upon probe failure.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeoutSeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].livenessProbe.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlivenessprobe)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].livenessProbe.grpc
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlivenessprobe)</sup></sup>



GRPC specifies an action involving a GRPC port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port number of the gRPC service. Number must be in the range 1 to 65535.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>service</b></td>
        <td>string</td>
        <td>
          Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].livenessProbe.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlivenessprobe)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexlivenessprobehttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].livenessProbe.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlivenessprobehttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].livenessProbe.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexlivenessprobe)</sup></sup>



TCPSocket specifies an action involving a TCP port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].ports[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



ContainerPort represents a network port in a single container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>containerPort</b></td>
        <td>integer</td>
        <td>
          Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>hostIP</b></td>
        <td>string</td>
        <td>
          What host IP to bind the external port to.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostPort</b></td>
        <td>integer</td>
        <td>
          Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          Protocol for port. Must be UDP, TCP, or SCTP. Defaults to "TCP".<br/>
          <br/>
            <i>Default</i>: TCP<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].readinessProbe
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexreadinessprobeexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexreadinessprobegrpc">grpc</a></b></td>
        <td>object</td>
        <td>
          GRPC specifies an action involving a GRPC port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexreadinessprobehttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initialDelaySeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>periodSeconds</b></td>
        <td>integer</td>
        <td>
          How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>successThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexreadinessprobetcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          TCPSocket specifies an action involving a TCP port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationGracePeriodSeconds</b></td>
        <td>integer</td>
        <td>
          Optional duration in seconds the pod needs to terminate gracefully upon probe failure.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeoutSeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].readinessProbe.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexreadinessprobe)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].readinessProbe.grpc
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexreadinessprobe)</sup></sup>



GRPC specifies an action involving a GRPC port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port number of the gRPC service. Number must be in the range 1 to 65535.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>service</b></td>
        <td>string</td>
        <td>
          Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].readinessProbe.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexreadinessprobe)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexreadinessprobehttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].readinessProbe.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexreadinessprobehttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].readinessProbe.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexreadinessprobe)</sup></sup>



TCPSocket specifies an action involving a TCP port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].resizePolicy[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



ContainerResizePolicy represents resource resize policy for the container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resourceName</b></td>
        <td>string</td>
        <td>
          Name of the resource to which this resource resize policy applies. Supported values: cpu, memory.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>restartPolicy</b></td>
        <td>string</td>
        <td>
          Restart policy to apply when specified resource is resized. If not specified, it defaults to NotRequired.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].resources
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].resources.claims[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].securityContext
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>allowPrivilegeEscalation</b></td>
        <td>boolean</td>
        <td>
          AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexsecuritycontextcapabilities">capabilities</a></b></td>
        <td>object</td>
        <td>
          The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>privileged</b></td>
        <td>boolean</td>
        <td>
          Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>procMount</b></td>
        <td>string</td>
        <td>
          procMount denotes the type of proc mount to use for the containers. The default is DefaultProcMount which uses the container runtime defaults for readonly paths and masked paths.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnlyRootFilesystem</b></td>
        <td>boolean</td>
        <td>
          Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsGroup</b></td>
        <td>integer</td>
        <td>
          The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsNonRoot</b></td>
        <td>boolean</td>
        <td>
          Indicates that the container must run as a non-root user.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUser</b></td>
        <td>integer</td>
        <td>
          The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexsecuritycontextselinuxoptions">seLinuxOptions</a></b></td>
        <td>object</td>
        <td>
          The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexsecuritycontextseccompprofile">seccompProfile</a></b></td>
        <td>object</td>
        <td>
          The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexsecuritycontextwindowsoptions">windowsOptions</a></b></td>
        <td>object</td>
        <td>
          The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].securityContext.capabilities
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexsecuritycontext)</sup></sup>



The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>add</b></td>
        <td>[]string</td>
        <td>
          Added capabilities<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>drop</b></td>
        <td>[]string</td>
        <td>
          Removed capabilities<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].securityContext.seLinuxOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexsecuritycontext)</sup></sup>



The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>level</b></td>
        <td>string</td>
        <td>
          Level is SELinux level label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>role</b></td>
        <td>string</td>
        <td>
          Role is a SELinux role label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type is a SELinux type label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          User is a SELinux user label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].securityContext.seccompProfile
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexsecuritycontext)</sup></sup>



The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type indicates which kind of seccomp profile will be applied. Valid options are: 
 Localhost - a profile defined in a file on the node should be used.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>localhostProfile</b></td>
        <td>string</td>
        <td>
          localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].securityContext.windowsOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexsecuritycontext)</sup></sup>



The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>gmsaCredentialSpec</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gmsaCredentialSpecName</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpecName is the name of the GMSA credential spec to use.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostProcess</b></td>
        <td>boolean</td>
        <td>
          HostProcess determines if a container should be run as a 'Host Process' container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUserName</b></td>
        <td>string</td>
        <td>
          The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].startupProbe
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexstartupprobeexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexstartupprobegrpc">grpc</a></b></td>
        <td>object</td>
        <td>
          GRPC specifies an action involving a GRPC port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexstartupprobehttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initialDelaySeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>periodSeconds</b></td>
        <td>integer</td>
        <td>
          How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>successThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexstartupprobetcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          TCPSocket specifies an action involving a TCP port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationGracePeriodSeconds</b></td>
        <td>integer</td>
        <td>
          Optional duration in seconds the pod needs to terminate gracefully upon probe failure.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeoutSeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].startupProbe.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexstartupprobe)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].startupProbe.grpc
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexstartupprobe)</sup></sup>



GRPC specifies an action involving a GRPC port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port number of the gRPC service. Number must be in the range 1 to 65535.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>service</b></td>
        <td>string</td>
        <td>
          Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].startupProbe.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexstartupprobe)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecadditionalcontainersindexstartupprobehttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].startupProbe.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexstartupprobehttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].startupProbe.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindexstartupprobe)</sup></sup>



TCPSocket specifies an action involving a TCP port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].volumeDevices[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



volumeDevice describes a mapping of a raw block device within a container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>devicePath</b></td>
        <td>string</td>
        <td>
          devicePath is the path inside of the container that the device will be mapped to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          name must match the name of a persistentVolumeClaim in the pod<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.additionalContainers[index].volumeMounts[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecadditionalcontainersindex)</sup></sup>



VolumeMount describes a mounting of a Volume within a container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mountPath</b></td>
        <td>string</td>
        <td>
          Path within the container at which the volume should be mounted.  Must not contain ':'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          This must match the Name of a Volume.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mountPropagation</b></td>
        <td>string</td>
        <td>
          mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subPath</b></td>
        <td>string</td>
        <td>
          Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subPathExpr</b></td>
        <td>string</td>
        <td>
          Expanded path within the volume from which the container's volume should be mounted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



If specified, indicates the pod's scheduling constraints

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes node affinity scheduling rules for the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.nodeAffinity
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinity)</sup></sup>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>
          If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitynodeaffinity)</sup></sup>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>
          A node selector term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitynodeaffinity)</sup></sup>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>
          Required. A list of node selector terms. The terms are ORed.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitynodeaffinityrequiredduringschedulingignoredduringexecution)</sup></sup>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinity)</sup></sup>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-locate

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinity)</sup></sup>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-locate

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



Autoscaler specifies the pod autoscaling configuration to use for the OpenTelemetryCollector workload.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalerbehavior">behavior</a></b></td>
        <td>object</td>
        <td>
          HorizontalPodAutoscalerBehavior configures the scaling behavior of the target in both Up and Down directions (scaleUp and scaleDown fields respectively).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxReplicas</b></td>
        <td>integer</td>
        <td>
          MaxReplicas sets an upper bound to the autoscaling feature. If MaxReplicas is set autoscaling is enabled.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalermetricsindex">metrics</a></b></td>
        <td>[]object</td>
        <td>
          Metrics is meant to provide a customizable way to configure HPA metrics. currently the only supported custom metrics is type=Pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minReplicas</b></td>
        <td>integer</td>
        <td>
          MinReplicas sets a lower bound to the autoscaling feature.  Set this if your are using autoscaling. It must be at least 1<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetCPUUtilization</b></td>
        <td>integer</td>
        <td>
          TargetCPUUtilization sets the target average CPU used across all replicas. If average CPU exceeds this value, the HPA will scale up. Defaults to 90 percent.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetMemoryUtilization</b></td>
        <td>integer</td>
        <td>
          TargetMemoryUtilization sets the target average memory utilization across all replicas<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.behavior
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscaler)</sup></sup>



HorizontalPodAutoscalerBehavior configures the scaling behavior of the target in both Up and Down directions (scaleUp and scaleDown fields respectively).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalerbehaviorscaledown">scaleDown</a></b></td>
        <td>object</td>
        <td>
          scaleDown is scaling policy for scaling Down. If not set, the default value is to allow to scale down to minReplicas pods, with a 300 second stabilization window (i.e.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalerbehaviorscaleup">scaleUp</a></b></td>
        <td>object</td>
        <td>
          scaleUp is scaling policy for scaling Up.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.behavior.scaleDown
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscalerbehavior)</sup></sup>



scaleDown is scaling policy for scaling Down. If not set, the default value is to allow to scale down to minReplicas pods, with a 300 second stabilization window (i.e.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalerbehaviorscaledownpoliciesindex">policies</a></b></td>
        <td>[]object</td>
        <td>
          policies is a list of potential scaling polices which can be used during scaling. At least one policy must be specified, otherwise the HPAScalingRules will be discarded as invalid<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>selectPolicy</b></td>
        <td>string</td>
        <td>
          selectPolicy is used to specify which policy should be used. If not set, the default value Max is used.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>stabilizationWindowSeconds</b></td>
        <td>integer</td>
        <td>
          stabilizationWindowSeconds is the number of seconds for which past recommendations should be considered while scaling up or scaling down.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.behavior.scaleDown.policies[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscalerbehaviorscaledown)</sup></sup>



HPAScalingPolicy is a single policy which must hold true for a specified past interval.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>periodSeconds</b></td>
        <td>integer</td>
        <td>
          periodSeconds specifies the window of time for which the policy should hold true. PeriodSeconds must be greater than zero and less than or equal to 1800 (30 min).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type is used to specify the scaling policy.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>integer</td>
        <td>
          value contains the amount of change which is permitted by the policy. It must be greater than zero<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.behavior.scaleUp
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscalerbehavior)</sup></sup>



scaleUp is scaling policy for scaling Up.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalerbehaviorscaleuppoliciesindex">policies</a></b></td>
        <td>[]object</td>
        <td>
          policies is a list of potential scaling polices which can be used during scaling. At least one policy must be specified, otherwise the HPAScalingRules will be discarded as invalid<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>selectPolicy</b></td>
        <td>string</td>
        <td>
          selectPolicy is used to specify which policy should be used. If not set, the default value Max is used.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>stabilizationWindowSeconds</b></td>
        <td>integer</td>
        <td>
          stabilizationWindowSeconds is the number of seconds for which past recommendations should be considered while scaling up or scaling down.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.behavior.scaleUp.policies[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscalerbehaviorscaleup)</sup></sup>



HPAScalingPolicy is a single policy which must hold true for a specified past interval.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>periodSeconds</b></td>
        <td>integer</td>
        <td>
          periodSeconds specifies the window of time for which the policy should hold true. PeriodSeconds must be greater than zero and less than or equal to 1800 (30 min).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type is used to specify the scaling policy.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>integer</td>
        <td>
          value contains the amount of change which is permitted by the policy. It must be greater than zero<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.metrics[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscaler)</sup></sup>



MetricSpec defines a subset of metrics to be defined for the HPA's metric array more metric type can be supported as needed. See https://pkg.go.dev/k8s.io/api/autoscaling/v2#MetricSpec for reference.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          MetricSourceType indicates the type of metric.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalermetricsindexpods">pods</a></b></td>
        <td>object</td>
        <td>
          PodsMetricSource indicates how to scale on a metric describing each pod in the current scale target (for example, transactions-processed-per-second).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.metrics[index].pods
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscalermetricsindex)</sup></sup>



PodsMetricSource indicates how to scale on a metric describing each pod in the current scale target (for example, transactions-processed-per-second).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalermetricsindexpodsmetric">metric</a></b></td>
        <td>object</td>
        <td>
          metric identifies the target metric by name and selector<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalermetricsindexpodstarget">target</a></b></td>
        <td>object</td>
        <td>
          target specifies the target value for the given metric<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.metrics[index].pods.metric
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscalermetricsindexpods)</sup></sup>



metric identifies the target metric by name and selector

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          name is the name of the given metric<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalermetricsindexpodsmetricselector">selector</a></b></td>
        <td>object</td>
        <td>
          selector is the string-encoded form of a standard kubernetes label selector for the given metric When set, it is passed as an additional parameter to the metrics server for more specific metrics scopi<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.metrics[index].pods.metric.selector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscalermetricsindexpodsmetric)</sup></sup>



selector is the string-encoded form of a standard kubernetes label selector for the given metric When set, it is passed as an additional parameter to the metrics server for more specific metrics scopi

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecautoscalermetricsindexpodsmetricselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.metrics[index].pods.metric.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscalermetricsindexpodsmetricselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.autoscaler.metrics[index].pods.target
<sup><sup>[↩ Parent](#opentelemetrycollectorspecautoscalermetricsindexpods)</sup></sup>



target specifies the target value for the given metric

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type represents whether the metric type is Utilization, Value, or AverageValue<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          averageUtilization is the target value of the average of the resource metric across all relevant pods, represented as a percentage of the requested value of the resource for the pods.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          averageValue is the target value of the average of the metric across all relevant pods (as a quantity)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          value is the target value of the metric (as a quantity).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.configmaps[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mountpath</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Configmap defines name and path where the configMaps should be mounted.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.env[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.env[index].valueFrom
<sup><sup>[↩ Parent](#opentelemetrycollectorspecenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.envFrom[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



EnvFromSource represents the source of a set of ConfigMaps

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecenvfromindexconfigmapref">configMapRef</a></b></td>
        <td>object</td>
        <td>
          The ConfigMap to select from<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>prefix</b></td>
        <td>string</td>
        <td>
          An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecenvfromindexsecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          The Secret to select from<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.envFrom[index].configMapRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecenvfromindex)</sup></sup>



The ConfigMap to select from

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.envFrom[index].secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecenvfromindex)</sup></sup>



The Secret to select from

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.ingress
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



Ingress is used to specify how OpenTelemetry Collector is exposed. This functionality is only available if one of the valid modes is set. Valid modes are: deployment, daemonset and statefulset.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td>
          Annotations to add to ingress. e.g. 'cert-manager.io/cluster-issuer: "letsencrypt"'<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostname</b></td>
        <td>string</td>
        <td>
          Hostname by which the ingress proxy can be reached.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ingressClassName</b></td>
        <td>string</td>
        <td>
          IngressClassName is the name of an IngressClass cluster resource. Ingress controller implementations use this field to know whether they should be serving this Ingress resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecingressroute">route</a></b></td>
        <td>object</td>
        <td>
          Route is an OpenShift specific section that is only considered when type "route" is used.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          RuleType defines how Ingress exposes collector receivers. IngressRuleTypePath ("path") exposes each receiver port on a unique path on single domain defined in Hostname.<br/>
          <br/>
            <i>Enum</i>: path, subdomain<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecingresstlsindex">tls</a></b></td>
        <td>[]object</td>
        <td>
          TLS configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type default value is: "" Supported types are: ingress, route<br/>
          <br/>
            <i>Enum</i>: ingress, route<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.ingress.route
<sup><sup>[↩ Parent](#opentelemetrycollectorspecingress)</sup></sup>



Route is an OpenShift specific section that is only considered when type "route" is used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>termination</b></td>
        <td>enum</td>
        <td>
          Termination indicates termination type. By default "edge" is used.<br/>
          <br/>
            <i>Enum</i>: insecure, edge, passthrough, reencrypt<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.ingress.tls[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecingress)</sup></sup>



IngressTLS describes the transport layer security associated with an ingress.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>hosts</b></td>
        <td>[]string</td>
        <td>
          hosts is a list of hosts included in the TLS certificate. The values in this list must match the name/s used in the tlsSecret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secretName</b></td>
        <td>string</td>
        <td>
          secretName is the name of the secret used to terminate TLS traffic on port 443. Field is left optional to allow TLS routing based on SNI hostname alone.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



A single application container that you want to run within a pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>args</b></td>
        <td>[]string</td>
        <td>
          Arguments to the entrypoint. The container image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Entrypoint array. Not executed within a shell. The container image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          List of environment variables to set in the container. Cannot be updated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexenvfromindex">envFrom</a></b></td>
        <td>[]object</td>
        <td>
          List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Container image name. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>imagePullPolicy</b></td>
        <td>string</td>
        <td>
          Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecycle">lifecycle</a></b></td>
        <td>object</td>
        <td>
          Actions that the management system should take in response to container lifecycle events. Cannot be updated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlivenessprobe">livenessProbe</a></b></td>
        <td>object</td>
        <td>
          Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexportsindex">ports</a></b></td>
        <td>[]object</td>
        <td>
          List of ports to expose from the container. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexreadinessprobe">readinessProbe</a></b></td>
        <td>object</td>
        <td>
          Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexresizepolicyindex">resizePolicy</a></b></td>
        <td>[]object</td>
        <td>
          Resources resize policy for the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexresources">resources</a></b></td>
        <td>object</td>
        <td>
          Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>restartPolicy</b></td>
        <td>string</td>
        <td>
          RestartPolicy defines the restart behavior of individual containers in a pod. This field may only be set for init containers, and the only allowed value is "Always".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexsecuritycontext">securityContext</a></b></td>
        <td>object</td>
        <td>
          SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexstartupprobe">startupProbe</a></b></td>
        <td>object</td>
        <td>
          StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>stdin</b></td>
        <td>boolean</td>
        <td>
          Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>stdinOnce</b></td>
        <td>boolean</td>
        <td>
          Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationMessagePath</b></td>
        <td>string</td>
        <td>
          Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationMessagePolicy</b></td>
        <td>string</td>
        <td>
          Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tty</b></td>
        <td>boolean</td>
        <td>
          Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexvolumedevicesindex">volumeDevices</a></b></td>
        <td>[]object</td>
        <td>
          volumeDevices is the list of block devices to be used by the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexvolumemountsindex">volumeMounts</a></b></td>
        <td>[]object</td>
        <td>
          Pod volumes to mount into the container's filesystem. Cannot be updated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>workingDir</b></td>
        <td>string</td>
        <td>
          Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].env[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].env[index].valueFrom
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].envFrom[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



EnvFromSource represents the source of a set of ConfigMaps

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexenvfromindexconfigmapref">configMapRef</a></b></td>
        <td>object</td>
        <td>
          The ConfigMap to select from<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>prefix</b></td>
        <td>string</td>
        <td>
          An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexenvfromindexsecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          The Secret to select from<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].envFrom[index].configMapRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexenvfromindex)</sup></sup>



The ConfigMap to select from

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].envFrom[index].secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexenvfromindex)</sup></sup>



The Secret to select from

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



Actions that the management system should take in response to container lifecycle events. Cannot be updated.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecyclepoststart">postStart</a></b></td>
        <td>object</td>
        <td>
          PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecycleprestop">preStop</a></b></td>
        <td>object</td>
        <td>
          PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.postStart
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecycle)</sup></sup>



PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecyclepoststartexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecyclepoststarthttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecyclepoststarttcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.postStart.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecyclepoststart)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.postStart.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecyclepoststart)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecyclepoststarthttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.postStart.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecyclepoststarthttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.postStart.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecyclepoststart)</sup></sup>



Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.preStop
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecycle)</sup></sup>



PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecycleprestopexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecycleprestophttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecycleprestoptcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.preStop.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecycleprestop)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.preStop.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecycleprestop)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlifecycleprestophttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.preStop.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecycleprestophttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].lifecycle.preStop.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlifecycleprestop)</sup></sup>



Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].livenessProbe
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlivenessprobeexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlivenessprobegrpc">grpc</a></b></td>
        <td>object</td>
        <td>
          GRPC specifies an action involving a GRPC port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlivenessprobehttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initialDelaySeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>periodSeconds</b></td>
        <td>integer</td>
        <td>
          How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>successThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlivenessprobetcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          TCPSocket specifies an action involving a TCP port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationGracePeriodSeconds</b></td>
        <td>integer</td>
        <td>
          Optional duration in seconds the pod needs to terminate gracefully upon probe failure.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeoutSeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].livenessProbe.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlivenessprobe)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].livenessProbe.grpc
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlivenessprobe)</sup></sup>



GRPC specifies an action involving a GRPC port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port number of the gRPC service. Number must be in the range 1 to 65535.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>service</b></td>
        <td>string</td>
        <td>
          Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].livenessProbe.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlivenessprobe)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexlivenessprobehttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].livenessProbe.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlivenessprobehttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].livenessProbe.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexlivenessprobe)</sup></sup>



TCPSocket specifies an action involving a TCP port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].ports[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



ContainerPort represents a network port in a single container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>containerPort</b></td>
        <td>integer</td>
        <td>
          Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>hostIP</b></td>
        <td>string</td>
        <td>
          What host IP to bind the external port to.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostPort</b></td>
        <td>integer</td>
        <td>
          Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          Protocol for port. Must be UDP, TCP, or SCTP. Defaults to "TCP".<br/>
          <br/>
            <i>Default</i>: TCP<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].readinessProbe
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexreadinessprobeexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexreadinessprobegrpc">grpc</a></b></td>
        <td>object</td>
        <td>
          GRPC specifies an action involving a GRPC port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexreadinessprobehttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initialDelaySeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>periodSeconds</b></td>
        <td>integer</td>
        <td>
          How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>successThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexreadinessprobetcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          TCPSocket specifies an action involving a TCP port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationGracePeriodSeconds</b></td>
        <td>integer</td>
        <td>
          Optional duration in seconds the pod needs to terminate gracefully upon probe failure.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeoutSeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].readinessProbe.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexreadinessprobe)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].readinessProbe.grpc
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexreadinessprobe)</sup></sup>



GRPC specifies an action involving a GRPC port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port number of the gRPC service. Number must be in the range 1 to 65535.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>service</b></td>
        <td>string</td>
        <td>
          Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].readinessProbe.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexreadinessprobe)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexreadinessprobehttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].readinessProbe.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexreadinessprobehttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].readinessProbe.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexreadinessprobe)</sup></sup>



TCPSocket specifies an action involving a TCP port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].resizePolicy[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



ContainerResizePolicy represents resource resize policy for the container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resourceName</b></td>
        <td>string</td>
        <td>
          Name of the resource to which this resource resize policy applies. Supported values: cpu, memory.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>restartPolicy</b></td>
        <td>string</td>
        <td>
          Restart policy to apply when specified resource is resized. If not specified, it defaults to NotRequired.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].resources
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].resources.claims[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].securityContext
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>allowPrivilegeEscalation</b></td>
        <td>boolean</td>
        <td>
          AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexsecuritycontextcapabilities">capabilities</a></b></td>
        <td>object</td>
        <td>
          The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>privileged</b></td>
        <td>boolean</td>
        <td>
          Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>procMount</b></td>
        <td>string</td>
        <td>
          procMount denotes the type of proc mount to use for the containers. The default is DefaultProcMount which uses the container runtime defaults for readonly paths and masked paths.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnlyRootFilesystem</b></td>
        <td>boolean</td>
        <td>
          Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsGroup</b></td>
        <td>integer</td>
        <td>
          The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsNonRoot</b></td>
        <td>boolean</td>
        <td>
          Indicates that the container must run as a non-root user.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUser</b></td>
        <td>integer</td>
        <td>
          The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexsecuritycontextselinuxoptions">seLinuxOptions</a></b></td>
        <td>object</td>
        <td>
          The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexsecuritycontextseccompprofile">seccompProfile</a></b></td>
        <td>object</td>
        <td>
          The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexsecuritycontextwindowsoptions">windowsOptions</a></b></td>
        <td>object</td>
        <td>
          The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].securityContext.capabilities
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexsecuritycontext)</sup></sup>



The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>add</b></td>
        <td>[]string</td>
        <td>
          Added capabilities<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>drop</b></td>
        <td>[]string</td>
        <td>
          Removed capabilities<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].securityContext.seLinuxOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexsecuritycontext)</sup></sup>



The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>level</b></td>
        <td>string</td>
        <td>
          Level is SELinux level label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>role</b></td>
        <td>string</td>
        <td>
          Role is a SELinux role label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type is a SELinux type label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          User is a SELinux user label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].securityContext.seccompProfile
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexsecuritycontext)</sup></sup>



The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type indicates which kind of seccomp profile will be applied. Valid options are: 
 Localhost - a profile defined in a file on the node should be used.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>localhostProfile</b></td>
        <td>string</td>
        <td>
          localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].securityContext.windowsOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexsecuritycontext)</sup></sup>



The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>gmsaCredentialSpec</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gmsaCredentialSpecName</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpecName is the name of the GMSA credential spec to use.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostProcess</b></td>
        <td>boolean</td>
        <td>
          HostProcess determines if a container should be run as a 'Host Process' container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUserName</b></td>
        <td>string</td>
        <td>
          The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].startupProbe
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexstartupprobeexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexstartupprobegrpc">grpc</a></b></td>
        <td>object</td>
        <td>
          GRPC specifies an action involving a GRPC port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexstartupprobehttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initialDelaySeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>periodSeconds</b></td>
        <td>integer</td>
        <td>
          How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>successThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexstartupprobetcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          TCPSocket specifies an action involving a TCP port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationGracePeriodSeconds</b></td>
        <td>integer</td>
        <td>
          Optional duration in seconds the pod needs to terminate gracefully upon probe failure.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeoutSeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].startupProbe.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexstartupprobe)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].startupProbe.grpc
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexstartupprobe)</sup></sup>



GRPC specifies an action involving a GRPC port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port number of the gRPC service. Number must be in the range 1 to 65535.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>service</b></td>
        <td>string</td>
        <td>
          Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].startupProbe.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexstartupprobe)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecinitcontainersindexstartupprobehttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].startupProbe.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexstartupprobehttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].startupProbe.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindexstartupprobe)</sup></sup>



TCPSocket specifies an action involving a TCP port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].volumeDevices[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



volumeDevice describes a mapping of a raw block device within a container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>devicePath</b></td>
        <td>string</td>
        <td>
          devicePath is the path inside of the container that the device will be mapped to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          name must match the name of a persistentVolumeClaim in the pod<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.initContainers[index].volumeMounts[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecinitcontainersindex)</sup></sup>



VolumeMount describes a mounting of a Volume within a container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mountPath</b></td>
        <td>string</td>
        <td>
          Path within the container at which the volume should be mounted.  Must not contain ':'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          This must match the Name of a Volume.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mountPropagation</b></td>
        <td>string</td>
        <td>
          mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subPath</b></td>
        <td>string</td>
        <td>
          Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subPathExpr</b></td>
        <td>string</td>
        <td>
          Expanded path within the volume from which the container's volume should be mounted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



Actions that the management system should take in response to container lifecycle events. Cannot be updated.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecyclepoststart">postStart</a></b></td>
        <td>object</td>
        <td>
          PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecycleprestop">preStop</a></b></td>
        <td>object</td>
        <td>
          PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.postStart
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecycle)</sup></sup>



PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecyclepoststartexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecyclepoststarthttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecyclepoststarttcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.postStart.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecyclepoststart)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.postStart.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecyclepoststart)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecyclepoststarthttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.postStart.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecyclepoststarthttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.postStart.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecyclepoststart)</sup></sup>



Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.preStop
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecycle)</sup></sup>



PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecycleprestopexec">exec</a></b></td>
        <td>object</td>
        <td>
          Exec specifies the action to take.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecycleprestophttpget">httpGet</a></b></td>
        <td>object</td>
        <td>
          HTTPGet specifies the http request to perform.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecycleprestoptcpsocket">tcpSocket</a></b></td>
        <td>object</td>
        <td>
          Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.preStop.exec
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecycleprestop)</sup></sup>



Exec specifies the action to take.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>command</b></td>
        <td>[]string</td>
        <td>
          Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.preStop.httpGet
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecycleprestop)</sup></sup>



HTTPGet specifies the http request to perform.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspeclifecycleprestophttpgethttpheadersindex">httpHeaders</a></b></td>
        <td>[]object</td>
        <td>
          Custom headers to set in the request. HTTP allows repeated headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path to access on the HTTP server.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scheme</b></td>
        <td>string</td>
        <td>
          Scheme to use for connecting to the host. Defaults to HTTP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.preStop.httpGet.httpHeaders[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecycleprestophttpget)</sup></sup>



HTTPHeader describes a custom header to be used in HTTP probes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The header field value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.lifecycle.preStop.tcpSocket
<sup><sup>[↩ Parent](#opentelemetrycollectorspeclifecycleprestop)</sup></sup>



Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Optional: Host name to connect to, defaults to the pod IP.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.livenessProbe
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



Liveness config for the OpenTelemetry Collector except the probe handler which is auto generated from the health extension of the collector.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initialDelaySeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after the container has started before liveness probes are initiated. Defaults to 0 seconds. Minimum value is 0. More info: https://kubernetes.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>periodSeconds</b></td>
        <td>integer</td>
        <td>
          How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>successThreshold</b></td>
        <td>integer</td>
        <td>
          Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>terminationGracePeriodSeconds</b></td>
        <td>integer</td>
        <td>
          Optional duration in seconds the pod needs to terminate gracefully upon probe failure.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeoutSeconds</b></td>
        <td>integer</td>
        <td>
          Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.observability
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



ObservabilitySpec defines how telemetry data gets handled.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecobservabilitymetrics">metrics</a></b></td>
        <td>object</td>
        <td>
          Metrics defines the metrics configuration for operands.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.observability.metrics
<sup><sup>[↩ Parent](#opentelemetrycollectorspecobservability)</sup></sup>



Metrics defines the metrics configuration for operands.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enableMetrics</b></td>
        <td>boolean</td>
        <td>
          EnableMetrics specifies if ServiceMonitor or PodMonitor(for sidecar mode) should be created for the OpenTelemetry Collector and Prometheus Exporters. The operator.observability.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.podDisruptionBudget
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



PodDisruptionBudget specifies the pod disruption budget configuration to use for the OpenTelemetryCollector workload.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxUnavailable</b></td>
        <td>int or string</td>
        <td>
          An eviction is allowed if at most "maxUnavailable" pods selected by "selector" are unavailable after the eviction, i.e. even in absence of the evicted pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minAvailable</b></td>
        <td>int or string</td>
        <td>
          An eviction is allowed if at least "minAvailable" pods selected by "selector" will still be available after the eviction, i.e. even in the absence of the evicted pod.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.podSecurityContext
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



PodSecurityContext configures the pod security context for the opentelemetry-collector pod, when running as a deployment, daemonset, or statefulset.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fsGroup</b></td>
        <td>integer</td>
        <td>
          A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod: 
 1.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>fsGroupChangePolicy</b></td>
        <td>string</td>
        <td>
          fsGroupChangePolicy defines behavior of changing ownership and permission of the volume before being exposed inside Pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsGroup</b></td>
        <td>integer</td>
        <td>
          The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsNonRoot</b></td>
        <td>boolean</td>
        <td>
          Indicates that the container must run as a non-root user.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUser</b></td>
        <td>integer</td>
        <td>
          The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecpodsecuritycontextselinuxoptions">seLinuxOptions</a></b></td>
        <td>object</td>
        <td>
          The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecpodsecuritycontextseccompprofile">seccompProfile</a></b></td>
        <td>object</td>
        <td>
          The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>supplementalGroups</b></td>
        <td>[]integer</td>
        <td>
          A list of groups applied to the first process run in each container, in addition to the container's primary GID, the fsGroup (if specified), and group memberships defined in the container image for th<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecpodsecuritycontextsysctlsindex">sysctls</a></b></td>
        <td>[]object</td>
        <td>
          Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported sysctls (by the container runtime) might fail to launch. Note that this field cannot be set when spec.os.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecpodsecuritycontextwindowsoptions">windowsOptions</a></b></td>
        <td>object</td>
        <td>
          The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.podSecurityContext.seLinuxOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspecpodsecuritycontext)</sup></sup>



The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>level</b></td>
        <td>string</td>
        <td>
          Level is SELinux level label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>role</b></td>
        <td>string</td>
        <td>
          Role is a SELinux role label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type is a SELinux type label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          User is a SELinux user label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.podSecurityContext.seccompProfile
<sup><sup>[↩ Parent](#opentelemetrycollectorspecpodsecuritycontext)</sup></sup>



The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type indicates which kind of seccomp profile will be applied. Valid options are: 
 Localhost - a profile defined in a file on the node should be used.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>localhostProfile</b></td>
        <td>string</td>
        <td>
          localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.podSecurityContext.sysctls[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecpodsecuritycontext)</sup></sup>



Sysctl defines a kernel parameter to be set

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of a property to set<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value of a property to set<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.podSecurityContext.windowsOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspecpodsecuritycontext)</sup></sup>



The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>gmsaCredentialSpec</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gmsaCredentialSpecName</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpecName is the name of the GMSA credential spec to use.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostProcess</b></td>
        <td>boolean</td>
        <td>
          HostProcess determines if a container should be run as a 'Host Process' container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUserName</b></td>
        <td>string</td>
        <td>
          The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.ports[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



ServicePort contains information on service's port.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          The port that will be exposed by this service.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>appProtocol</b></td>
        <td>string</td>
        <td>
          The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand. This field follows standard Kubernetes label syntax.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The name of this port within the service. This must be a DNS_LABEL. All ports within a ServiceSpec must have unique names.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodePort</b></td>
        <td>integer</td>
        <td>
          The port on each node on which this service is exposed when type is NodePort or LoadBalancer.  Usually assigned by the system.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          The IP protocol for this port. Supports "TCP", "UDP", and "SCTP". Default is TCP.<br/>
          <br/>
            <i>Default</i>: TCP<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetPort</b></td>
        <td>int or string</td>
        <td>
          Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.resources
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



Resources to set on the OpenTelemetry Collector pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.resources.claims[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.securityContext
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



SecurityContext configures the container security context for the opentelemetry-collector container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>allowPrivilegeEscalation</b></td>
        <td>boolean</td>
        <td>
          AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecsecuritycontextcapabilities">capabilities</a></b></td>
        <td>object</td>
        <td>
          The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>privileged</b></td>
        <td>boolean</td>
        <td>
          Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>procMount</b></td>
        <td>string</td>
        <td>
          procMount denotes the type of proc mount to use for the containers. The default is DefaultProcMount which uses the container runtime defaults for readonly paths and masked paths.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnlyRootFilesystem</b></td>
        <td>boolean</td>
        <td>
          Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsGroup</b></td>
        <td>integer</td>
        <td>
          The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsNonRoot</b></td>
        <td>boolean</td>
        <td>
          Indicates that the container must run as a non-root user.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUser</b></td>
        <td>integer</td>
        <td>
          The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecsecuritycontextselinuxoptions">seLinuxOptions</a></b></td>
        <td>object</td>
        <td>
          The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecsecuritycontextseccompprofile">seccompProfile</a></b></td>
        <td>object</td>
        <td>
          The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecsecuritycontextwindowsoptions">windowsOptions</a></b></td>
        <td>object</td>
        <td>
          The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.securityContext.capabilities
<sup><sup>[↩ Parent](#opentelemetrycollectorspecsecuritycontext)</sup></sup>



The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>add</b></td>
        <td>[]string</td>
        <td>
          Added capabilities<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>drop</b></td>
        <td>[]string</td>
        <td>
          Removed capabilities<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.securityContext.seLinuxOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspecsecuritycontext)</sup></sup>



The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>level</b></td>
        <td>string</td>
        <td>
          Level is SELinux level label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>role</b></td>
        <td>string</td>
        <td>
          Role is a SELinux role label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type is a SELinux type label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          User is a SELinux user label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.securityContext.seccompProfile
<sup><sup>[↩ Parent](#opentelemetrycollectorspecsecuritycontext)</sup></sup>



The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type indicates which kind of seccomp profile will be applied. Valid options are: 
 Localhost - a profile defined in a file on the node should be used.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>localhostProfile</b></td>
        <td>string</td>
        <td>
          localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.securityContext.windowsOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspecsecuritycontext)</sup></sup>



The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>gmsaCredentialSpec</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gmsaCredentialSpecName</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpecName is the name of the GMSA credential spec to use.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostProcess</b></td>
        <td>boolean</td>
        <td>
          HostProcess determines if a container should be run as a 'Host Process' container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUserName</b></td>
        <td>string</td>
        <td>
          The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



TargetAllocator indicates a value which determines whether to spawn a target allocation resource or not.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinity">affinity</a></b></td>
        <td>object</td>
        <td>
          If specified, indicates the pod's scheduling constraints<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>allocationStrategy</b></td>
        <td>enum</td>
        <td>
          AllocationStrategy determines which strategy the target allocator should use for allocation. The current options are least-weighted and consistent-hashing. The default option is least-weighted<br/>
          <br/>
            <i>Enum</i>: least-weighted, consistent-hashing<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enabled indicates whether to use a target allocation mechanism for Prometheus targets or not.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorenvindex">env</a></b></td>
        <td>[]object</td>
        <td>
          ENV vars to set on the OpenTelemetry TargetAllocator's Pods. These can then in certain cases be consumed in the config file for the TargetAllocator.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>filterStrategy</b></td>
        <td>string</td>
        <td>
          FilterStrategy determines how to filter targets before allocating them among the collectors. The only current option is relabel-config (drops targets based on prom relabel_config).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image indicates the container image to use for the OpenTelemetry TargetAllocator.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          NodeSelector to schedule OpenTelemetry TargetAllocator pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorprometheuscr">prometheusCR</a></b></td>
        <td>object</td>
        <td>
          PrometheusCR defines the configuration for the retrieval of PrometheusOperator CRDs ( servicemonitor.monitoring.coreos.com/v1 and podmonitor.monitoring.coreos.com/v1 )  retrieval.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas is the number of pod instances for the underlying TargetAllocator. This should only be set to a value other than 1 if a strategy that allows for high availability is chosen.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources to set on the OpenTelemetryTargetAllocator containers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorsecuritycontext">securityContext</a></b></td>
        <td>object</td>
        <td>
          SecurityContext configures the container security context for the targetallocator.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>serviceAccount</b></td>
        <td>string</td>
        <td>
          ServiceAccount indicates the name of an existing service account to use with this instance. When set, the operator will not automatically create a ServiceAccount for the TargetAllocator.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatortolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Toleration embedded kubernetes pod configuration option, controls how pods can be scheduled with matching taints<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatortopologyspreadconstraintsindex">topologySpreadConstraints</a></b></td>
        <td>[]object</td>
        <td>
          TopologySpreadConstraints embedded kubernetes pod configuration option, controls how pods are spread across your cluster among failure-domains such as regions, zones, nodes, and other user-defined top<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocator)</sup></sup>



If specified, indicates the pod's scheduling constraints

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes node affinity scheduling rules for the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.nodeAffinity
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinity)</sup></sup>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>
          If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitynodeaffinity)</sup></sup>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>
          A node selector term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitynodeaffinity)</sup></sup>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>
          Required. A list of node selector terms. The terms are ORed.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitynodeaffinityrequiredduringschedulingignoredduringexecution)</sup></sup>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinity)</sup></sup>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-locate

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinity)</sup></sup>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-locate

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatoraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.env[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocator)</sup></sup>



EnvVar represents an environment variable present in a Container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the environment variable. Must be a C_IDENTIFIER.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorenvindexvaluefrom">valueFrom</a></b></td>
        <td>object</td>
        <td>
          Source for the environment variable's value. Cannot be used if value is not empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.env[index].valueFrom
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorenvindex)</sup></sup>



Source for the environment variable's value. Cannot be used if value is not empty.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorenvindexvaluefromconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorenvindexvaluefromfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorenvindexvaluefromresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorenvindexvaluefromsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret in the pod's namespace<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.env[index].valueFrom.configMapKeyRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorenvindexvaluefrom)</sup></sup>



Selects a key of a ConfigMap.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.env[index].valueFrom.fieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorenvindexvaluefrom)</sup></sup>



Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.env[index].valueFrom.resourceFieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorenvindexvaluefrom)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.env[index].valueFrom.secretKeyRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorenvindexvaluefrom)</sup></sup>



Selects a key of a secret in the pod's namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key of the secret to select from.  Must be a valid secret key.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.prometheusCR
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocator)</sup></sup>



PrometheusCR defines the configuration for the retrieval of PrometheusOperator CRDs ( servicemonitor.monitoring.coreos.com/v1 and podmonitor.monitoring.coreos.com/v1 )  retrieval.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enabled indicates whether to use a PrometheusOperator custom resources as targets or not.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>podMonitorSelector</b></td>
        <td>map[string]string</td>
        <td>
          PodMonitors to be selected for target discovery. This is a map of {key,value} pairs. Each {key,value} in the map is going to exactly match a label in a PodMonitor's meta labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scrapeInterval</b></td>
        <td>string</td>
        <td>
          Interval between consecutive scrapes. Equivalent to the same setting on the Prometheus CRD. 
 Default: "30s"<br/>
          <br/>
            <i>Format</i>: duration<br/>
            <i>Default</i>: 30s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>serviceMonitorSelector</b></td>
        <td>map[string]string</td>
        <td>
          ServiceMonitors to be selected for target discovery. This is a map of {key,value} pairs. Each {key,value} in the map is going to exactly match a label in a ServiceMonitor's meta labels.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.resources
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocator)</sup></sup>



Resources to set on the OpenTelemetryTargetAllocator containers.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.resources.claims[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.securityContext
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocator)</sup></sup>



SecurityContext configures the container security context for the targetallocator.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fsGroup</b></td>
        <td>integer</td>
        <td>
          A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod: 
 1.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>fsGroupChangePolicy</b></td>
        <td>string</td>
        <td>
          fsGroupChangePolicy defines behavior of changing ownership and permission of the volume before being exposed inside Pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsGroup</b></td>
        <td>integer</td>
        <td>
          The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsNonRoot</b></td>
        <td>boolean</td>
        <td>
          Indicates that the container must run as a non-root user.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUser</b></td>
        <td>integer</td>
        <td>
          The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorsecuritycontextselinuxoptions">seLinuxOptions</a></b></td>
        <td>object</td>
        <td>
          The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorsecuritycontextseccompprofile">seccompProfile</a></b></td>
        <td>object</td>
        <td>
          The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>supplementalGroups</b></td>
        <td>[]integer</td>
        <td>
          A list of groups applied to the first process run in each container, in addition to the container's primary GID, the fsGroup (if specified), and group memberships defined in the container image for th<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorsecuritycontextsysctlsindex">sysctls</a></b></td>
        <td>[]object</td>
        <td>
          Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported sysctls (by the container runtime) might fail to launch. Note that this field cannot be set when spec.os.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatorsecuritycontextwindowsoptions">windowsOptions</a></b></td>
        <td>object</td>
        <td>
          The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.securityContext.seLinuxOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorsecuritycontext)</sup></sup>



The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>level</b></td>
        <td>string</td>
        <td>
          Level is SELinux level label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>role</b></td>
        <td>string</td>
        <td>
          Role is a SELinux role label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type is a SELinux type label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          User is a SELinux user label that applies to the container.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.securityContext.seccompProfile
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorsecuritycontext)</sup></sup>



The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type indicates which kind of seccomp profile will be applied. Valid options are: 
 Localhost - a profile defined in a file on the node should be used.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>localhostProfile</b></td>
        <td>string</td>
        <td>
          localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.securityContext.sysctls[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorsecuritycontext)</sup></sup>



Sysctl defines a kernel parameter to be set

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of a property to set<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value of a property to set<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.securityContext.windowsOptions
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatorsecuritycontext)</sup></sup>



The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>gmsaCredentialSpec</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gmsaCredentialSpecName</b></td>
        <td>string</td>
        <td>
          GMSACredentialSpecName is the name of the GMSA credential spec to use.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hostProcess</b></td>
        <td>boolean</td>
        <td>
          HostProcess determines if a container should be run as a 'Host Process' container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runAsUserName</b></td>
        <td>string</td>
        <td>
          The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.tolerations[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocator)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.topologySpreadConstraints[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocator)</sup></sup>



TopologySpreadConstraint specifies how to spread matching pods among the given topology.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxSkew</b></td>
        <td>integer</td>
        <td>
          MaxSkew describes the degree to which pods may be unevenly distributed.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>whenUnsatisfiable</b></td>
        <td>string</td>
        <td>
          WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatortopologyspreadconstraintsindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select the pods over which spreading will be calculated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minDomains</b></td>
        <td>integer</td>
        <td>
          MinDomains indicates a minimum number of eligible domains.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeAffinityPolicy</b></td>
        <td>string</td>
        <td>
          NodeAffinityPolicy indicates how we will treat Pod's nodeAffinity/nodeSelector when calculating pod topology spread skew.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeTaintsPolicy</b></td>
        <td>string</td>
        <td>
          NodeTaintsPolicy indicates how we will treat node taints when calculating pod topology spread skew.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.topologySpreadConstraints[index].labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatortopologyspreadconstraintsindex)</sup></sup>



LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectargetallocatortopologyspreadconstraintsindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.targetAllocator.topologySpreadConstraints[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectargetallocatortopologyspreadconstraintsindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.tolerations[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.topologySpreadConstraints[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



TopologySpreadConstraint specifies how to spread matching pods among the given topology.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxSkew</b></td>
        <td>integer</td>
        <td>
          MaxSkew describes the degree to which pods may be unevenly distributed.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>whenUnsatisfiable</b></td>
        <td>string</td>
        <td>
          WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspectopologyspreadconstraintsindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select the pods over which spreading will be calculated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minDomains</b></td>
        <td>integer</td>
        <td>
          MinDomains indicates a minimum number of eligible domains.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeAffinityPolicy</b></td>
        <td>string</td>
        <td>
          NodeAffinityPolicy indicates how we will treat Pod's nodeAffinity/nodeSelector when calculating pod topology spread skew.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeTaintsPolicy</b></td>
        <td>string</td>
        <td>
          NodeTaintsPolicy indicates how we will treat node taints when calculating pod topology spread skew.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.topologySpreadConstraints[index].labelSelector
<sup><sup>[↩ Parent](#opentelemetrycollectorspectopologyspreadconstraintsindex)</sup></sup>



LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspectopologyspreadconstraintsindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.topologySpreadConstraints[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspectopologyspreadconstraintsindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.updateStrategy
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



UpdateStrategy represents the strategy the operator will take replacing existing DaemonSet pods with new pods https://kubernetes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecupdatestrategyrollingupdate">rollingUpdate</a></b></td>
        <td>object</td>
        <td>
          Rolling update config params. Present only if type = "RollingUpdate". --- TODO: Update this to follow our convention for oneOf, whatever we decide it to be. Same as Deployment `strategy.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type of daemon set update. Can be "RollingUpdate" or "OnDelete". Default is RollingUpdate.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.updateStrategy.rollingUpdate
<sup><sup>[↩ Parent](#opentelemetrycollectorspecupdatestrategy)</sup></sup>



Rolling update config params. Present only if type = "RollingUpdate". --- TODO: Update this to follow our convention for oneOf, whatever we decide it to be. Same as Deployment `strategy.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxSurge</b></td>
        <td>int or string</td>
        <td>
          The maximum number of nodes with an existing available DaemonSet pod that can have an updated DaemonSet pod during during an update.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxUnavailable</b></td>
        <td>int or string</td>
        <td>
          The maximum number of DaemonSet pods that can be unavailable during the update.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



PersistentVolumeClaim is a user's request for and claim to a persistent volume

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexmetadata">metadata</a></b></td>
        <td>object</td>
        <td>
          Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexspec">spec</a></b></td>
        <td>object</td>
        <td>
          spec defines the desired characteristics of a volume requested by a pod author. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexstatus">status</a></b></td>
        <td>object</td>
        <td>
          status represents the current information/status of a persistent volume claim. Read-only. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].metadata
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindex)</sup></sup>



Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>finalizers</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].spec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindex)</sup></sup>



spec defines the desired characteristics of a volume requested by a pod author. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>
          accessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexspecdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>
          dataSource field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexspecdatasourceref">dataSourceRef</a></b></td>
        <td>object</td>
        <td>
          dataSourceRef specifies the object from which to populate the volume with data, if a non-empty volume is desired.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexspecresources">resources</a></b></td>
        <td>object</td>
        <td>
          resources represents the minimum resources the volume should have.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexspecselector">selector</a></b></td>
        <td>object</td>
        <td>
          selector is a label query over volumes to consider for binding.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>
          storageClassName is the name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>
          volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>
          volumeName is the binding reference to the PersistentVolume backing this claim.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].spec.dataSource
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindexspec)</sup></sup>



dataSource field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind is the type of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>
          APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].spec.dataSourceRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindexspec)</sup></sup>



dataSourceRef specifies the object from which to populate the volume with data, if a non-empty volume is desired.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind is the type of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>
          APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace is the namespace of resource being referenced Note that when a namespace is specified, a gateway.networking.k8s.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].spec.resources
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindexspec)</sup></sup>



resources represents the minimum resources the volume should have.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexspecresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].spec.resources.claims[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindexspecresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].spec.selector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindexspec)</sup></sup>



selector is a label query over volumes to consider for binding.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexspecselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].spec.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindexspecselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].status
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindex)</sup></sup>



status represents the current information/status of a persistent volume claim. Read-only. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>
          accessModes contains the actual access modes the volume backing the PVC has. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>allocatedResourceStatuses</b></td>
        <td>map[string]string</td>
        <td>
          allocatedResourceStatuses stores status of resource being resized for the given PVC. Key names follow standard Kubernetes label syntax.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>allocatedResources</b></td>
        <td>map[string]int or string</td>
        <td>
          allocatedResources tracks the resources allocated to a PVC including its capacity. Key names follow standard Kubernetes label syntax.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>capacity</b></td>
        <td>map[string]int or string</td>
        <td>
          capacity represents the actual resources of the underlying volume.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumeclaimtemplatesindexstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          conditions is the current Condition of persistent volume claim. If underlying persistent volume is being resized then the Condition will be set to 'ResizeStarted'.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>phase</b></td>
        <td>string</td>
        <td>
          phase represents the current phase of PersistentVolumeClaim.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeClaimTemplates[index].status.conditions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumeclaimtemplatesindexstatus)</sup></sup>



PersistentVolumeClaimCondition contains details about state of pvc

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          PersistentVolumeClaimConditionType is a valid value of PersistentVolumeClaimCondition.Type<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>lastProbeTime</b></td>
        <td>string</td>
        <td>
          lastProbeTime is the time we probed the condition.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the time the condition transitioned from one status to another.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is the human-readable message indicating details about last transition.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason is a unique, this should be a short, machine understandable string that gives the reason for condition's last transition.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumeMounts[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



VolumeMount describes a mounting of a Volume within a container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mountPath</b></td>
        <td>string</td>
        <td>
          Path within the container at which the volume should be mounted.  Must not contain ':'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          This must match the Name of a Volume.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mountPropagation</b></td>
        <td>string</td>
        <td>
          mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subPath</b></td>
        <td>string</td>
        <td>
          Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subPathExpr</b></td>
        <td>string</td>
        <td>
          Expanded path within the volume from which the container's volume should be mounted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspec)</sup></sup>



Volume represents a named volume in a pod that may be accessed by any container in the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          name of the volume. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexawselasticblockstore">awsElasticBlockStore</a></b></td>
        <td>object</td>
        <td>
          awsElasticBlockStore represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexazuredisk">azureDisk</a></b></td>
        <td>object</td>
        <td>
          azureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexazurefile">azureFile</a></b></td>
        <td>object</td>
        <td>
          azureFile represents an Azure File Service mount on the host and bind mount to the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexcephfs">cephfs</a></b></td>
        <td>object</td>
        <td>
          cephFS represents a Ceph FS mount on the host that shares a pod's lifetime<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexcinder">cinder</a></b></td>
        <td>object</td>
        <td>
          cinder represents a cinder volume attached and mounted on kubelets host machine. More info: https://examples.k8s.io/mysql-cinder-pd/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>
          configMap represents a configMap that should populate this volume<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexcsi">csi</a></b></td>
        <td>object</td>
        <td>
          csi (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers (Beta feature).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>
          downwardAPI represents downward API about the pod that should populate this volume<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexemptydir">emptyDir</a></b></td>
        <td>object</td>
        <td>
          emptyDir represents a temporary directory that shares a pod's lifetime. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeral">ephemeral</a></b></td>
        <td>object</td>
        <td>
          ephemeral represents a volume that is handled by a cluster storage driver.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexfc">fc</a></b></td>
        <td>object</td>
        <td>
          fc represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexflexvolume">flexVolume</a></b></td>
        <td>object</td>
        <td>
          flexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexflocker">flocker</a></b></td>
        <td>object</td>
        <td>
          flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexgcepersistentdisk">gcePersistentDisk</a></b></td>
        <td>object</td>
        <td>
          gcePersistentDisk represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexgitrepo">gitRepo</a></b></td>
        <td>object</td>
        <td>
          gitRepo represents a git repository at a particular revision. DEPRECATED: GitRepo is deprecated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexglusterfs">glusterfs</a></b></td>
        <td>object</td>
        <td>
          glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/glusterfs/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexhostpath">hostPath</a></b></td>
        <td>object</td>
        <td>
          hostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexiscsi">iscsi</a></b></td>
        <td>object</td>
        <td>
          iscsi represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://examples.k8s.io/volumes/iscsi/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexnfs">nfs</a></b></td>
        <td>object</td>
        <td>
          nfs represents an NFS mount on the host that shares a pod's lifetime More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexpersistentvolumeclaim">persistentVolumeClaim</a></b></td>
        <td>object</td>
        <td>
          persistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. More info: https://kubernetes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexphotonpersistentdisk">photonPersistentDisk</a></b></td>
        <td>object</td>
        <td>
          photonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexportworxvolume">portworxVolume</a></b></td>
        <td>object</td>
        <td>
          portworxVolume represents a portworx volume attached and mounted on kubelets host machine<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojected">projected</a></b></td>
        <td>object</td>
        <td>
          projected items for all in one resources secrets, configmaps, and downward API<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexquobyte">quobyte</a></b></td>
        <td>object</td>
        <td>
          quobyte represents a Quobyte mount on the host that shares a pod's lifetime<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexrbd">rbd</a></b></td>
        <td>object</td>
        <td>
          rbd represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/rbd/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexscaleio">scaleIO</a></b></td>
        <td>object</td>
        <td>
          scaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexsecret">secret</a></b></td>
        <td>object</td>
        <td>
          secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexstorageos">storageos</a></b></td>
        <td>object</td>
        <td>
          storageOS represents a StorageOS volume attached and mounted on Kubernetes nodes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexvspherevolume">vsphereVolume</a></b></td>
        <td>object</td>
        <td>
          vsphereVolume represents a vSphere volume attached and mounted on kubelets host machine<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].awsElasticBlockStore
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



awsElasticBlockStore represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>volumeID</b></td>
        <td>string</td>
        <td>
          volumeID is unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>partition</b></td>
        <td>integer</td>
        <td>
          partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1".<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly value true will force the readOnly setting in VolumeMounts. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].azureDisk
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



azureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>diskName</b></td>
        <td>string</td>
        <td>
          diskName is the Name of the data disk in the blob storage<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>diskURI</b></td>
        <td>string</td>
        <td>
          diskURI is the URI of data disk in the blob storage<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>cachingMode</b></td>
        <td>string</td>
        <td>
          cachingMode is the Host Caching mode: None, Read Only, Read Write.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          kind expected values are Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].azureFile
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



azureFile represents an Azure File Service mount on the host and bind mount to the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>secretName</b></td>
        <td>string</td>
        <td>
          secretName is the  name of secret that contains Azure Storage Account Name and Key<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>shareName</b></td>
        <td>string</td>
        <td>
          shareName is the azure share Name<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].cephfs
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



cephFS represents a Ceph FS mount on the host that shares a pod's lifetime

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>monitors</b></td>
        <td>[]string</td>
        <td>
          monitors is Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is Optional: Used as the mounted root, rather than the full Ceph tree, default is /<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secretFile</b></td>
        <td>string</td>
        <td>
          secretFile is Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexcephfssecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          user is optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].cephfs.secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexcephfs)</sup></sup>



secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].cinder
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



cinder represents a cinder volume attached and mounted on kubelets host machine. More info: https://examples.k8s.io/mysql-cinder-pd/README.md

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>volumeID</b></td>
        <td>string</td>
        <td>
          volumeID used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexcindersecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is optional: points to a secret object containing parameters used to connect to OpenStack.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].cinder.secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexcinder)</sup></sup>



secretRef is optional: points to a secret object containing parameters used to connect to OpenStack.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].configMap
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



configMap represents a configMap that should populate this volume

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultMode</b></td>
        <td>integer</td>
        <td>
          defaultMode is optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          items if unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          optional specify whether the ConfigMap or its keys must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].configMap.items[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexconfigmap)</sup></sup>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the key to project.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].csi
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



csi (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers (Beta feature).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>driver</b></td>
        <td>string</td>
        <td>
          driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType to mount. Ex. "ext4", "xfs", "ntfs". If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexcsinodepublishsecretref">nodePublishSecretRef</a></b></td>
        <td>object</td>
        <td>
          nodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly specifies a read-only configuration for the volume. Defaults to false (read/write).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeAttributes</b></td>
        <td>map[string]string</td>
        <td>
          volumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver's documentation for supported values.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].csi.nodePublishSecretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexcsi)</sup></sup>



nodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].downwardAPI
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



downwardAPI represents downward API about the pod that should populate this volume

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultMode</b></td>
        <td>integer</td>
        <td>
          Optional: mode bits to use on created files by default. Must be a Optional: mode bits used to set permissions on created files by default.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          Items is a list of downward API volume file<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].downwardAPI.items[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexdownwardapi)</sup></sup>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].downwardAPI.items[index].fieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexdownwardapiitemsindex)</sup></sup>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].downwardAPI.items[index].resourceFieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexdownwardapiitemsindex)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].emptyDir
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



emptyDir represents a temporary directory that shares a pod's lifetime. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>medium</b></td>
        <td>string</td>
        <td>
          medium represents what type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sizeLimit</b></td>
        <td>int or string</td>
        <td>
          sizeLimit is the total amount of local storage required for this EmptyDir volume. The size limit is also applicable for memory medium.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



ephemeral represents a volume that is handled by a cluster storage driver.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplate">volumeClaimTemplate</a></b></td>
        <td>object</td>
        <td>
          Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC, i.e.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral.volumeClaimTemplate
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexephemeral)</sup></sup>



Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC, i.e.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespec">spec</a></b></td>
        <td>object</td>
        <td>
          The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatemetadata">metadata</a></b></td>
        <td>object</td>
        <td>
          May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during validation.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral.volumeClaimTemplate.spec
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplate)</sup></sup>



The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>
          accessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespecdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>
          dataSource field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespecdatasourceref">dataSourceRef</a></b></td>
        <td>object</td>
        <td>
          dataSourceRef specifies the object from which to populate the volume with data, if a non-empty volume is desired.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespecresources">resources</a></b></td>
        <td>object</td>
        <td>
          resources represents the minimum resources the volume should have.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespecselector">selector</a></b></td>
        <td>object</td>
        <td>
          selector is a label query over volumes to consider for binding.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>
          storageClassName is the name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>
          volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>
          volumeName is the binding reference to the PersistentVolume backing this claim.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.dataSource
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespec)</sup></sup>



dataSource field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind is the type of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>
          APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.dataSourceRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespec)</sup></sup>



dataSourceRef specifies the object from which to populate the volume with data, if a non-empty volume is desired.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind is the type of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of resource being referenced<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>
          APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace is the namespace of resource being referenced Note that when a namespace is specified, a gateway.networking.k8s.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.resources
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespec)</sup></sup>



resources represents the minimum resources the volume should have.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespecresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.resources.claims[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespecresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.selector
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespec)</sup></sup>



selector is a label query over volumes to consider for binding.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespecselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral.volumeClaimTemplate.spec.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplatespecselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].ephemeral.volumeClaimTemplate.metadata
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexephemeralvolumeclaimtemplate)</sup></sup>



May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during validation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>finalizers</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].fc
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



fc represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lun</b></td>
        <td>integer</td>
        <td>
          lun is Optional: FC target lun number<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetWWNs</b></td>
        <td>[]string</td>
        <td>
          targetWWNs is Optional: FC target worldwide names (WWNs)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>wwids</b></td>
        <td>[]string</td>
        <td>
          wwids Optional: FC volume world wide identifiers (wwids) Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].flexVolume
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



flexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>driver</b></td>
        <td>string</td>
        <td>
          driver is the name of the driver to use for this volume.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>options</b></td>
        <td>map[string]string</td>
        <td>
          options is Optional: this field holds extra command options if any.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly is Optional: defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexflexvolumesecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is Optional: secretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].flexVolume.secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexflexvolume)</sup></sup>



secretRef is Optional: secretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].flocker
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>datasetName</b></td>
        <td>string</td>
        <td>
          datasetName is Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>datasetUUID</b></td>
        <td>string</td>
        <td>
          datasetUUID is the UUID of the dataset. This is unique identifier of a Flocker dataset<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].gcePersistentDisk
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



gcePersistentDisk represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>pdName</b></td>
        <td>string</td>
        <td>
          pdName is unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>partition</b></td>
        <td>integer</td>
        <td>
          partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1".<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].gitRepo
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



gitRepo represents a git repository at a particular revision. DEPRECATED: GitRepo is deprecated.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>repository</b></td>
        <td>string</td>
        <td>
          repository is the URL<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>directory</b></td>
        <td>string</td>
        <td>
          directory is the target directory name. Must not contain or start with '..'.  If '.' is supplied, the volume directory will be the git repository.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>revision</b></td>
        <td>string</td>
        <td>
          revision is the commit hash for the specified revision.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].glusterfs
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/glusterfs/README.md

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>endpoints</b></td>
        <td>string</td>
        <td>
          endpoints is the endpoint name that details Glusterfs topology. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the Glusterfs volume path. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].hostPath
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



hostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path of the directory on the host. If the path is a symlink, it will follow the link to the real path. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type for HostPath Volume Defaults to "" More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].iscsi
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



iscsi represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://examples.k8s.io/volumes/iscsi/README.md

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>iqn</b></td>
        <td>string</td>
        <td>
          iqn is the target iSCSI Qualified Name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>lun</b></td>
        <td>integer</td>
        <td>
          lun represents iSCSI Target Lun number.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>targetPortal</b></td>
        <td>string</td>
        <td>
          targetPortal is iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>chapAuthDiscovery</b></td>
        <td>boolean</td>
        <td>
          chapAuthDiscovery defines whether support iSCSI Discovery CHAP authentication<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>chapAuthSession</b></td>
        <td>boolean</td>
        <td>
          chapAuthSession defines whether support iSCSI Session CHAP authentication<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initiatorName</b></td>
        <td>string</td>
        <td>
          initiatorName is the custom iSCSI Initiator Name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>iscsiInterface</b></td>
        <td>string</td>
        <td>
          iscsiInterface is the interface Name that uses an iSCSI transport. Defaults to 'default' (tcp).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>portals</b></td>
        <td>[]string</td>
        <td>
          portals is the iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexiscsisecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is the CHAP Secret for iSCSI target and initiator authentication<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].iscsi.secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexiscsi)</sup></sup>



secretRef is the CHAP Secret for iSCSI target and initiator authentication

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].nfs
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



nfs represents an NFS mount on the host that shares a pod's lifetime More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path that is exported by the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>server</b></td>
        <td>string</td>
        <td>
          server is the hostname or IP address of the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the NFS export to be mounted with read-only permissions. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].persistentVolumeClaim
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



persistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. More info: https://kubernetes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>claimName</b></td>
        <td>string</td>
        <td>
          claimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly Will force the ReadOnly setting in VolumeMounts. Default false.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].photonPersistentDisk
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



photonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>pdID</b></td>
        <td>string</td>
        <td>
          pdID is the ID that identifies Photon Controller persistent disk<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].portworxVolume
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



portworxVolume represents a portworx volume attached and mounted on kubelets host machine

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>volumeID</b></td>
        <td>string</td>
        <td>
          volumeID uniquely identifies a Portworx volume<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fSType represents the filesystem type to mount Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



projected items for all in one resources secrets, configmaps, and downward API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultMode</b></td>
        <td>integer</td>
        <td>
          defaultMode are the mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindex">sources</a></b></td>
        <td>[]object</td>
        <td>
          sources is the list of volume projections<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojected)</sup></sup>



Projection that may be projected along with other supported volume types

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>
          configMap information about the configMap data to project<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>
          downwardAPI information about the downwardAPI data to project<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindexsecret">secret</a></b></td>
        <td>object</td>
        <td>
          secret information about the secret data to project<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindexserviceaccounttoken">serviceAccountToken</a></b></td>
        <td>object</td>
        <td>
          serviceAccountToken is information about the serviceAccountToken data to project<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index].configMap
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojectedsourcesindex)</sup></sup>



configMap information about the configMap data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          items if unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          optional specify whether the ConfigMap or its keys must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index].configMap.items[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojectedsourcesindexconfigmap)</sup></sup>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the key to project.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index].downwardAPI
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojectedsourcesindex)</sup></sup>



downwardAPI information about the downwardAPI data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          Items is a list of DownwardAPIVolume file<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index].downwardAPI.items[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojectedsourcesindexdownwardapi)</sup></sup>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>
          Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>
          Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index].downwardAPI.items[index].fieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojectedsourcesindexdownwardapiitemsindex)</sup></sup>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>
          Path of the field to select in the specified API version.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          Version of the schema the FieldPath is written in terms of, defaults to "v1".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index].downwardAPI.items[index].resourceFieldRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojectedsourcesindexdownwardapiitemsindex)</sup></sup>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>
          Required: resource to select<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          Container name: required for volumes, optional for env vars<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>
          Specifies the output format of the exposed resources, defaults to "1"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index].secret
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojectedsourcesindex)</sup></sup>



secret information about the secret data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexprojectedsourcesindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          items if unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          optional field specify whether the Secret or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index].secret.items[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojectedsourcesindexsecret)</sup></sup>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the key to project.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].projected.sources[index].serviceAccountToken
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexprojectedsourcesindex)</sup></sup>



serviceAccountToken is information about the serviceAccountToken data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the path relative to the mount point of the file to project the token into.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>audience</b></td>
        <td>string</td>
        <td>
          audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>expirationSeconds</b></td>
        <td>integer</td>
        <td>
          expirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].quobyte
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



quobyte represents a Quobyte mount on the host that shares a pod's lifetime

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>registry</b></td>
        <td>string</td>
        <td>
          registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>volume</b></td>
        <td>string</td>
        <td>
          volume is a string that references an already created Quobyte volume by name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>group</b></td>
        <td>string</td>
        <td>
          group to map volume access to Default is no group<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tenant</b></td>
        <td>string</td>
        <td>
          tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          user to map volume access to Defaults to serivceaccount user<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].rbd
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



rbd represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/rbd/README.md

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          image is the rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>monitors</b></td>
        <td>[]string</td>
        <td>
          monitors is a collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>keyring</b></td>
        <td>string</td>
        <td>
          keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>pool</b></td>
        <td>string</td>
        <td>
          pool is the rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexrbdsecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>user</b></td>
        <td>string</td>
        <td>
          user is the rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].rbd.secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexrbd)</sup></sup>



secretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].scaleIO
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



scaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>gateway</b></td>
        <td>string</td>
        <td>
          gateway is the host address of the ScaleIO API Gateway.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexscaleiosecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>system</b></td>
        <td>string</td>
        <td>
          system is the name of the storage system as configured in ScaleIO.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Default is "xfs".<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protectionDomain</b></td>
        <td>string</td>
        <td>
          protectionDomain is the name of the ScaleIO Protection Domain for the configured storage.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sslEnabled</b></td>
        <td>boolean</td>
        <td>
          sslEnabled Flag enable/disable SSL communication with Gateway, default false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageMode</b></td>
        <td>string</td>
        <td>
          storageMode indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storagePool</b></td>
        <td>string</td>
        <td>
          storagePool is the ScaleIO Storage Pool associated with the protection domain.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>
          volumeName is the name of a volume already created in the ScaleIO system that is associated with this volume source.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].scaleIO.secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexscaleio)</sup></sup>



secretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].secret
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultMode</b></td>
        <td>integer</td>
        <td>
          defaultMode is Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>
          items If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          optional field specify whether the Secret or its keys must be defined<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secretName</b></td>
        <td>string</td>
        <td>
          secretName is the name of the secret in the pod's namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].secret.items[index]
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexsecret)</sup></sup>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the key to project.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>
          mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].storageos
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



storageOS represents a StorageOS volume attached and mounted on Kubernetes nodes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readOnly</b></td>
        <td>boolean</td>
        <td>
          readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorspecvolumesindexstorageossecretref">secretRef</a></b></td>
        <td>object</td>
        <td>
          secretRef specifies the secret to use for obtaining the StorageOS API credentials.  If not specified, default values will be attempted.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>
          volumeName is the human-readable name of the StorageOS volume.  Volume names are only unique within a namespace.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeNamespace</b></td>
        <td>string</td>
        <td>
          volumeNamespace specifies the scope of the volume within StorageOS.  If no namespace is specified then the Pod's namespace will be used.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].storageos.secretRef
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindexstorageos)</sup></sup>



secretRef specifies the secret to use for obtaining the StorageOS API credentials.  If not specified, default values will be attempted.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.spec.volumes[index].vsphereVolume
<sup><sup>[↩ Parent](#opentelemetrycollectorspecvolumesindex)</sup></sup>



vsphereVolume represents a vSphere volume attached and mounted on kubelets host machine

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>volumePath</b></td>
        <td>string</td>
        <td>
          volumePath is the path that identifies vSphere volume vmdk<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>fsType</b></td>
        <td>string</td>
        <td>
          fsType is filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storagePolicyID</b></td>
        <td>string</td>
        <td>
          storagePolicyID is the storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storagePolicyName</b></td>
        <td>string</td>
        <td>
          storagePolicyName is the storage Policy Based Management (SPBM) profile name.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.status
<sup><sup>[↩ Parent](#opentelemetrycollector)</sup></sup>



OpenTelemetryCollectorStatus defines the observed state of OpenTelemetryCollector.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          Image indicates the container image to use for the OpenTelemetry Collector.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>messages</b></td>
        <td>[]string</td>
        <td>
          Messages about actions performed by the operator on this resource. Deprecated: use Kubernetes events instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas is currently not being set and might be removed in the next version. Deprecated: use "OpenTelemetryCollector.Status.Scale.Replicas" instead.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#opentelemetrycollectorstatusscale">scale</a></b></td>
        <td>object</td>
        <td>
          Scale is the OpenTelemetryCollector's scale subresource status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>version</b></td>
        <td>string</td>
        <td>
          Version of the managed OpenTelemetry Collector (operand)<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OpenTelemetryCollector.status.scale
<sup><sup>[↩ Parent](#opentelemetrycollectorstatus)</sup></sup>



Scale is the OpenTelemetryCollector's scale subresource status.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          The total number non-terminated pods targeted by this OpenTelemetryCollector's deployment or statefulSet.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>selector</b></td>
        <td>string</td>
        <td>
          The selector used to match the OpenTelemetryCollector's deployment or statefulSet pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>statusReplicas</b></td>
        <td>string</td>
        <td>
          StatusReplicas is the number of pods targeted by this OpenTelemetryCollector's with a Ready Condition / Total number of non-terminated pods targeted by this OpenTelemetryCollector's (their labels matc<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>