// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adapters

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	// prepare

	// First Test - Exporters
	configStr := `
receivers:
  httpd/mtls:
    protocols:
      http:
        endpoint: mysite.local:55690
  jaeger:
    protocols:
      grpc:
  prometheus:
    protocols:
      grpc:

processors:

exporters:
  debug:

service:
  pipelines:
    metrics:
      receivers: [httpd/mtls, jaeger]
      exporters: [debug]
    metrics/1:
      receivers: [httpd/mtls, jaeger]
      exporters: [debug]
`
	// // prepare
	config, err := ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	check := getEnabledComponents(config, ComponentTypeReceiver)
	require.NotEmpty(t, check)
}

func TestEmptyEnabledReceivers(t *testing.T) {
	// prepare

	// First Test - Exporters
	configStr := `
receivers:
  httpd/mtls:
    protocols:
      http:
        endpoint: mysite.local:55690
  jaeger:
    protocols:
      grpc:
  prometheus:
    protocols:
      grpc:

processors:

exporters:
  debug:

service:
  pipelines:
    metrics:
      receivers: []
      exporters: []
    metrics/1:
      receivers: []
      exporters: []
`
	// // prepare
	config, err := ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	check := getEnabledComponents(config, ComponentTypeReceiver)
	require.Empty(t, check)
}
