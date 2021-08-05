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

package reconcilers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDesiredConfigMap(t *testing.T) {
	t.Run("should return expected config map", func(t *testing.T) {
		expectedLables := map[string]string{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/instance":   "default.test",
			"app.kubernetes.io/part-of":    "opentelemetry",
			"app.kubernetes.io/component":  "opentelemetry-collector",
			"app.kubernetes.io/name":       "test-collector",
		}

		expectedData := map[string]string{
			"collector.yaml": `
    receivers:
      jaeger:
        protocols:
          grpc:
    processors:

    exporters:
      logging:

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          processors: []
          exporters: [logging]

`,
		}

		actual := desiredConfigMap(context.Background(), params())

		assert.Equal(t, "test-collector", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})

}
