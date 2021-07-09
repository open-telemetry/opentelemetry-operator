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

package adapters_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	lbadapters "github.com/open-telemetry/opentelemetry-operator/pkg/loadbalancer/adapters"
)

func TestExtractPromConfigFromConfig(t *testing.T) {
	configStr := `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
  examplereceiver/settings:
    endpoint: "0.0.0.0:12346"
  prometheus:
    config:
      scrape_config:
        job_name: otel-collector
        scrape_interval: 10s
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
`
	expectedData := map[interface{}]interface{}{
		"scrape_config": map[interface{}]interface{}{
			"job_name":        "otel-collector",
			"scrape_interval": "10s",
		},
	}

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	promConfig, notify := lbadapters.ConfigToPromConfig(config)
	assert.Equal(t, notify, "")

	// verify
	assert.Equal(t, expectedData, promConfig)
}

func TestExtractPromConfigFromNullConfig(t *testing.T) {
	configStr := `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
  examplereceiver/settings:
    endpoint: "0.0.0.0:12346"
  prometheus:
    config:
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	promConfig, notify := lbadapters.ConfigToPromConfig(config)
	assert.Equal(t, notify, lbadapters.ErrorNotAMap("prometheusConfig"))

	// verify
	assert.True(t, reflect.ValueOf(promConfig).IsNil())
}
