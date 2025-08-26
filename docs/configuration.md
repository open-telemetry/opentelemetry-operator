# CLI Flags Reference

This document lists all available CLI flags for the OpenTelemetry Operator. These configuration flags can also be parsed as JSON using the same keys mentioned below.

## Flag Table by Categories

### Core Configuration

<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Type</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr><td><b>metrics-addr</b></td><td>string</td><td>The address the metric endpoint binds to.</td></tr>
    <tr><td><b>health-probe-addr</b></td><td>string</td><td>The address the probe endpoint binds to.</td></tr>
    <tr><td><b>pprof-addr</b></td><td>string</td><td>The address to expose the pprof server. Default is empty string which disables the pprof server.</td></tr>
    <tr><td><b>enable-leader-election</b></td><td>bool</td><td>Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.</td></tr>
    <tr><td><b>webhook-port</b></td><td>int</td><td>The port the webhook endpoint binds to.</td></tr>
    <tr><td><b>enable-webhooks</b></td><td>bool</td><td>Enable webhooks for the controllers.</td></tr>
  </tbody>
</table>

### Auto-Instrumentation Enablement

<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Type</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr><td><b>enable-multi-instrumentation</b></td><td>bool</td><td>Controls whether the operator supports multi instrumentation.</td></tr>
    <tr><td><b>enable-apache-httpd-instrumentation</b></td><td>bool</td><td>Controls whether the operator supports Apache HTTPD auto-instrumentation.</td></tr>
    <tr><td><b>enable-dotnet-instrumentation</b></td><td>bool</td><td>Controls whether the operator supports dotnet auto-instrumentation.</td></tr>
    <tr><td><b>enable-go-instrumentation</b></td><td>bool</td><td>Controls whether the operator supports Go auto-instrumentation.</td></tr>
    <tr><td><b>enable-python-instrumentation</b></td><td>bool</td><td>Controls whether the operator supports python auto-instrumentation.</td></tr>
    <tr><td><b>enable-nginx-instrumentation</b></td><td>bool</td><td>Controls whether the operator supports nginx auto-instrumentation.</td></tr>
    <tr><td><b>enable-nodejs-instrumentation</b></td><td>bool</td><td>Controls whether the operator supports nodejs auto-instrumentation.</td></tr>
    <tr><td><b>enable-java-instrumentation</b></td><td>bool</td><td>Controls whether the operator supports java auto-instrumentation.</td></tr>
  </tbody>
</table>

### Image Configuration

<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Type</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr><td><b>collector-image</b></td><td>string</td><td>The default OpenTelemetry collector image. This image is used when no image is specified in the CustomResource.</td></tr>
    <tr><td><b>target-allocator-image</b></td><td>string</td><td>The default OpenTelemetry target allocator image. This image is used when no image is specified in the CustomResource.</td></tr>
    <tr><td><b>operator-opamp-bridge-image</b></td><td>string</td><td>The default OpenTelemetry Operator OpAMP Bridge image. This image is used when no image is specified in the CustomResource.</td></tr>
    <tr><td><b>auto-instrumentation-java-image</b></td><td>string</td><td>The default OpenTelemetry Java instrumentation image. This image is used when no image is specified in the CustomResource.</td></tr>
    <tr><td><b>auto-instrumentation-nodejs-image</b></td><td>string</td><td>The default OpenTelemetry NodeJS instrumentation image. This image is used when no image is specified in the CustomResource.</td></tr>
    <tr><td><b>auto-instrumentation-python-image</b></td><td>string</td><td>The default OpenTelemetry Python instrumentation image. This image is used when no image is specified in the CustomResource.</td></tr>
    <tr><td><b>auto-instrumentation-dotnet-image</b></td><td>string</td><td>The default OpenTelemetry DotNet instrumentation image. This image is used when no image is specified in the CustomResource.</td></tr>
    <tr><td><b>auto-instrumentation-go-image</b></td><td>string</td><td>The default OpenTelemetry Go instrumentation image. This image is used when no image is specified in the CustomResource.</td></tr>
    <tr><td><b>auto-instrumentation-apache-httpd-image</b></td><td>string</td><td>The default OpenTelemetry Apache HTTPD instrumentation image. This image is used when no image is specified in the CustomResource.</td></tr>
    <tr><td><b>auto-instrumentation-nginx-image</b></td><td>string</td><td>The default OpenTelemetry Nginx instrumentation image. This image is used when no image is specified in the CustomResource.</td></tr>
  </tbody>
</table>

### Monitoring & Observability

<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Type</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr><td><b>enable-cr-metrics</b></td><td>bool</td><td>Controls whether exposing the CR metrics is enabled.</td></tr>
    <tr><td><b>create-sm-operator-metrics</b></td><td>bool</td><td>Create a ServiceMonitor for the operator metrics.</td></tr>
    <tr><td><b>openshift-create-dashboard</b></td><td>bool</td><td>Create an OpenShift dashboard for monitoring the OpenTelemetryCollector instances.</td></tr>
  </tbody>
</table>

### Security & TLS

<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Type</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr><td><b>tls-min-version</b></td><td>string</td><td>Minimum TLS version supported. Value must match version names from https://golang.org/pkg/crypto/tls/#pkg-constants.</td></tr>
    <tr><td><b>tls-cipher-suites</b></td><td>string[]</td><td>Comma-separated list of cipher suites for the server. Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants). If omitted, the default Go cipher suites will be used.</td></tr>
  </tbody>
</table>

### Logging Configuration

<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Type</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr><td><b>zap-message-key</b></td><td>string</td><td>The message key to be used in the customized Log Encoder.</td></tr>
    <tr><td><b>zap-level-key</b></td><td>string</td><td>The level key to be used in the customized Log Encoder.</td></tr>
    <tr><td><b>zap-time-key</b></td><td>string</td><td>The time key to be used in the customized Log Encoder.</td></tr>
    <tr><td><b>zap-level-format</b></td><td>string</td><td>The level format to be used in the customized Log Encoder.</td></tr>
  </tbody>
</table>

### Filtering & Other

<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Type</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr><td><b>labels-filter</b></td><td>string[]</td><td>Labels to filter away from propagating onto deploys. Patterns are literal strings optionally containing a <code>*</code> wildcard character. Example: <code>--labels-filter=.*filter.out</code> will filter out labels like <code>label.filter.out: true</code>.</td></tr>
    <tr><td><b>annotations-filter</b></td><td>string[]</td><td>Annotations to filter away from propagating onto deploys. Patterns are literal strings optionally containing a <code>*</code> wildcard character. Example: <code>--annotations-filter=.*filter.out</code> will filter out annotations like <code>annotation.filter.out: true</code>.</td></tr>
    <tr><td><b>fips-disabled-components</b></td><td>string</td><td>Disabled collector components when operator runs on FIPS enabled platform. Example: <code>receiver.foo,receiver.bar,exporter.baz</code>.</td></tr>
    <tr><td><b>ignore-missing-collector-crds</b></td><td>bool</td><td>Ignore missing OpenTelemetryCollector CRDs presence in the cluster.</td></tr>
    <tr><td><b>create-rbac-permissions</b></td><td>bool</td><td>Automatically create RBAC permissions needed by the processors (deprecated).</td></tr>
  </tbody>
</table>
