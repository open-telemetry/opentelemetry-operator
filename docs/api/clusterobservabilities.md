# API Reference

Packages:

- [opentelemetry.io/v1alpha1](#opentelemetryiov1alpha1)

# opentelemetry.io/v1alpha1

Resource Types:

- [ClusterObservability](#clusterobservability)




## ClusterObservability
<sup><sup>[↩ Parent](#opentelemetryiov1alpha1 )</sup></sup>






ClusterObservability is the Schema for the clusterobservabilities API.

<table>
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
      <td>ClusterObservability</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#clusterobservabilityspec">spec</a></b></td>
        <td>object</td>
        <td>
          ClusterObservabilitySpec defines the desired state of ClusterObservability.
This follows a simplified design using a single OTLP HTTP exporter for all signals.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterobservabilitystatus">status</a></b></td>
        <td>object</td>
        <td>
          ClusterObservabilityStatus defines the observed state of ClusterObservability.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterObservability.spec
<sup><sup>[↩ Parent](#clusterobservability)</sup></sup>



ClusterObservabilitySpec defines the desired state of ClusterObservability.
This follows a simplified design using a single OTLP HTTP exporter for all signals.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#clusterobservabilityspecexporter">exporter</a></b></td>
        <td>object</td>
        <td>
          Exporter defines the OTLP HTTP exporter configuration for all signals.
The collector will automatically append appropriate paths for each signal type.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>signals</b></td>
        <td>[]enum</td>
        <td>
          Signals defines which observability signals to collect and export.
Must contain at least one signal type from: logs, traces, metrics, profiles<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### ClusterObservability.spec.exporter
<sup><sup>[↩ Parent](#clusterobservabilityspec)</sup></sup>



Exporter defines the OTLP HTTP exporter configuration for all signals.
The collector will automatically append appropriate paths for each signal type.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>compression</b></td>
        <td>enum</td>
        <td>
          Compression defines the compression algorithm to use.
By default gzip compression is enabled. Use "none" to disable.<br/>
          <br/>
            <i>Enum</i>: gzip, none, <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>encoding</b></td>
        <td>enum</td>
        <td>
          Encoding defines the encoding to use for the messages.
Valid options: proto, json. Default is proto.<br/>
          <br/>
            <i>Enum</i>: proto, json<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>endpoint</b></td>
        <td>string</td>
        <td>
          Endpoint is the target base URL to send data to (e.g., https://example.com:4318).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>headers</b></td>
        <td>map[string]string</td>
        <td>
          Headers defines additional headers to be sent with each request.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logs_endpoint</b></td>
        <td>string</td>
        <td>
          LogsEndpoint is the target URL to send log data to (e.g., https://example.com:4318/v1/logs).
If this setting is present the endpoint setting is ignored for logs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>metrics_endpoint</b></td>
        <td>string</td>
        <td>
          MetricsEndpoint is the target URL to send metric data to (e.g., https://example.com:4318/v1/metrics).
If this setting is present the endpoint setting is ignored for metrics.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>profiles_endpoint</b></td>
        <td>string</td>
        <td>
          ProfilesEndpoint is the target URL to send profile data to (e.g., https://example.com:4318/v1/development/profiles).
If this setting is present the endpoint setting is ignored for profiles.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>read_buffer_size</b></td>
        <td>integer</td>
        <td>
          ReadBufferSize for HTTP client. Default is 0.<br/>
          <br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterobservabilityspecexporterretry_on_failure">retry_on_failure</a></b></td>
        <td>object</td>
        <td>
          RetryOnFailure defines retry configuration for failed requests.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterobservabilityspecexportersending_queue">sending_queue</a></b></td>
        <td>object</td>
        <td>
          SendingQueue defines configuration for the sending queue.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeout</b></td>
        <td>string</td>
        <td>
          Timeout is the HTTP request time limit (e.g., "30s", "1m"). Default is 30s.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterobservabilityspecexportertls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS defines TLS configuration for the exporter.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>traces_endpoint</b></td>
        <td>string</td>
        <td>
          TracesEndpoint is the target URL to send trace data to (e.g., https://example.com:4318/v1/traces).
If this setting is present the endpoint setting is ignored for traces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>write_buffer_size</b></td>
        <td>integer</td>
        <td>
          WriteBufferSize for HTTP client. Default is 512 * 1024.<br/>
          <br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterObservability.spec.exporter.retry_on_failure
<sup><sup>[↩ Parent](#clusterobservabilityspecexporter)</sup></sup>



RetryOnFailure defines retry configuration for failed requests.

<table>
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
          Enabled controls whether retry is enabled. Default is true.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>initial_interval</b></td>
        <td>string</td>
        <td>
          InitialInterval is the initial retry interval (e.g., "5s"). Default is 5s.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>max_elapsed_time</b></td>
        <td>string</td>
        <td>
          MaxElapsedTime is the maximum elapsed time for retries (e.g., "5m"). Default is 5m.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>max_interval</b></td>
        <td>string</td>
        <td>
          MaxInterval is the maximum retry interval (e.g., "30s"). Default is 30s.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>multiplier</b></td>
        <td>string</td>
        <td>
          Multiplier is the multiplier for retry intervals (e.g., "1.5"). Default is 1.5.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>randomization_factor</b></td>
        <td>string</td>
        <td>
          RandomizationFactor is the randomization factor for retry intervals (e.g., "0.5"). Default is 0.5.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterObservability.spec.exporter.sending_queue
<sup><sup>[↩ Parent](#clusterobservabilityspecexporter)</sup></sup>



SendingQueue defines configuration for the sending queue.

<table>
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
          Enabled controls whether the queue is enabled. Default is true.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>num_consumers</b></td>
        <td>integer</td>
        <td>
          NumConsumers is the number of consumers that dequeue batches. Default is 10.<br/>
          <br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>queue_size</b></td>
        <td>integer</td>
        <td>
          QueueSize is the maximum number of batches allowed in queue at a given time. Default is 1000.<br/>
          <br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterObservability.spec.exporter.tls
<sup><sup>[↩ Parent](#clusterobservabilityspecexporter)</sup></sup>



TLS defines TLS configuration for the exporter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ca_file</b></td>
        <td>string</td>
        <td>
          CAFile is the path to the CA certificate file for server verification.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>cert_file</b></td>
        <td>string</td>
        <td>
          CertFile is the path to the client certificate file for mutual TLS.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecure</b></td>
        <td>boolean</td>
        <td>
          Insecure controls whether to use insecure transport. Default is false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key_file</b></td>
        <td>string</td>
        <td>
          KeyFile is the path to the client private key file for mutual TLS.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>server_name</b></td>
        <td>string</td>
        <td>
          ServerName for TLS handshake. If empty, uses the hostname from endpoint.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterObservability.status
<sup><sup>[↩ Parent](#clusterobservability)</sup></sup>



ClusterObservabilityStatus defines the observed state of ClusterObservability.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#clusterobservabilitystatuscomponentsstatuskey">componentsStatus</a></b></td>
        <td>map[string]object</td>
        <td>
          ComponentsStatus provides status information about individual observability components.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterobservabilitystatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          Conditions represent the latest available observations of the ClusterObservability state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>configVersions</b></td>
        <td>map[string]string</td>
        <td>
          ConfigVersions tracks the version hashes of the configuration files used.
This enables detection of config changes when operator is upgraded.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          Message provides additional information about the current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          ObservedGeneration is the most recent generation observed for this ClusterObservability.
It corresponds to the ClusterObservability's generation, which is updated on mutation
by the API Server.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>phase</b></td>
        <td>string</td>
        <td>
          Phase represents the current phase of the ClusterObservability.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterObservability.status.componentsStatus[key]
<sup><sup>[↩ Parent](#clusterobservabilitystatus)</sup></sup>



ComponentStatus represents the status of an individual component.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastUpdated</b></td>
        <td>string</td>
        <td>
          LastUpdated is the last time this status was updated.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          Message provides additional information about the component status.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ready</b></td>
        <td>boolean</td>
        <td>
          Ready indicates whether the component is ready.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterObservability.status.conditions[index]
<sup><sup>[↩ Parent](#clusterobservabilitystatus)</sup></sup>



ClusterObservabilityCondition represents a condition of a ClusterObservability.

<table>
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
          Status of the condition.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type of condition.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          Last time the condition transitioned from one status to another.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          A human readable message indicating details about the transition.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          ObservedGeneration represents the .metadata.generation that the condition was set based upon.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          The reason for the condition's last transition.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>