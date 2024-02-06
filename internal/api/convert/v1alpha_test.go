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

package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func Test_V1Alpha1to2(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := `---
receivers:
  otlp:
    protocols:
      grpc:
processors:
  resourcedetection:
    detectors: [kubernetes]
exporters:
  otlp:
    endpoint: "otlp:4317"
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resourcedetection]
      exporters: [otlp]
`
		cfgV1 := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Config: config,
			},
		}

		cfgV2, err := V1Alpha1to2(cfgV1)
		assert.Nil(t, err)
		assert.NotNil(t, cfgV2)

		yamlCfg, err := yaml.Marshal(&cfgV2.Spec.Config)
		assert.Nil(t, err)
		assert.YAMLEq(t, config, string(yamlCfg))
	})
	t.Run("invalid config", func(t *testing.T) {
		config := `!!!`
		cfgV1 := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Config: config,
			},
		}

		_, err := V1Alpha1to2(cfgV1)
		assert.ErrorContains(t, err, "could not convert config json to v1alpha2.Config")
	})
}
