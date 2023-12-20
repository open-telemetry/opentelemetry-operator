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

package v1alpha2

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

var collectorCfg = `
receivers:
  jaeger:
    protocols:
     thrift_compact:
     grpc:
       endpoint: 0.0.0.0:14250
  kafka:
    protocol_version: 2.0.0
  
  test_types:
    number_float:
      value: 12.1
    number_int:
      value: 12

processors:
  batch:

exporters:
  debug:

service:
  pipelines:
   traces:
    receivers: [jaeger]
    processors: []
    exporters: [debug]
`

func TestConfigMarshalling(t *testing.T) {
	jsonCfg, err := yaml.YAMLToJSON([]byte(collectorCfg))
	require.NoError(t, err)

	fmt.Println(string(jsonCfg))
	c := &Config{}
	err = json.Unmarshal(jsonCfg, c)
	require.NoError(t, err)

	jsonConfig, err := json.Marshal(c)
	require.NoError(t, err)
	assert.JSONEq(t, string(jsonCfg), string(jsonConfig))

	yamlCfg, err := yaml.JSONToYAML(jsonConfig)
	assert.YAMLEq(t, collectorCfg, string(yamlCfg))
}
