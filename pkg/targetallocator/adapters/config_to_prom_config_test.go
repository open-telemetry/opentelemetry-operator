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
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
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
		"config": map[interface{}]interface{}{
			"scrape_config": map[interface{}]interface{}{
				"job_name":        "otel-collector",
				"scrape_interval": "10s",
			},
		},
	}

	// test
	promConfig, err := ta.ConfigToPromConfig(configStr)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, expectedData, promConfig)
}

func TestExtractPromConfigWithTAConfigFromConfig(t *testing.T) {
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
    target_allocator:
      endpoint: "test:80"
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
`
	expectedData := map[interface{}]interface{}{
		"config": map[interface{}]interface{}{
			"scrape_config": map[interface{}]interface{}{
				"job_name":        "otel-collector",
				"scrape_interval": "10s",
			},
		},
		"target_allocator": map[interface{}]interface{}{
			"endpoint": "test:80",
		},
	}

	// test
	promConfig, err := ta.ConfigToPromConfig(configStr)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, expectedData, promConfig)
}

func TestExtractPromConfigFromNullConfig(t *testing.T) {
	configStr := `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
  examplereceiver/settings:
    endpoint: "0.0.0.0:12346"
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
`

	// test
	promConfig, err := ta.ConfigToPromConfig(configStr)
	assert.Equal(t, err, fmt.Errorf("no prometheus available as part of the configuration"))

	// verify
	assert.True(t, reflect.ValueOf(promConfig).IsNil())
}

func TestValidatePromConfig(t *testing.T) {
	testCases := []struct {
		description                   string
		config                        map[interface{}]interface{}
		targetAllocatorEnabled        bool
		targetAllocatorRewriteEnabled bool
		expectedError                 error
	}{
		{
			description:                   "target_allocator and rewrite enabled",
			config:                        map[interface{}]interface{}{},
			targetAllocatorEnabled:        true,
			targetAllocatorRewriteEnabled: true,
			expectedError:                 nil,
		},
		{
			description: "target_allocator enabled, target_allocator section present",
			config: map[interface{}]interface{}{
				"target_allocator": map[interface{}]interface{}{},
			},
			targetAllocatorEnabled:        true,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 nil,
		},
		{
			description: "target_allocator enabled, config section present",
			config: map[interface{}]interface{}{
				"config": map[interface{}]interface{}{},
			},
			targetAllocatorEnabled:        true,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 nil,
		},
		{
			description:                   "target_allocator enabled, neither section present",
			config:                        map[interface{}]interface{}{},
			targetAllocatorEnabled:        true,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 errors.New("either target allocator or prometheus config needs to be present"),
		},
		{
			description: "target_allocator disabled, config section present",
			config: map[interface{}]interface{}{
				"config": map[interface{}]interface{}{},
			},
			targetAllocatorEnabled:        false,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 nil,
		},
		{
			description:                   "target_allocator disabled, config section not present",
			config:                        map[interface{}]interface{}{},
			targetAllocatorEnabled:        false,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 fmt.Errorf("no %s available as part of the configuration", "prometheusConfig"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.description, func(t *testing.T) {
			err := ta.ValidatePromConfig(testCase.config, testCase.targetAllocatorEnabled, testCase.targetAllocatorRewriteEnabled)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}
