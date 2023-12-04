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

package targetallocator

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/stretchr/testify/assert"
)

func TestDesiredServiceMonitors(t *testing.T) {
	otelcol := collectorInstance()
	cfg := config.New()

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	actual, err := ServiceMonitor(params)
	assert.NoError(t, err)
	assert.Nil(t, actual)

	params.OtelCol.Spec.TargetAllocator.Observability.Metrics.EnableMetrics = true
	actual, err = ServiceMonitor(params)
	assert.NoError(t, err)
	assert.NotNil(t, actual)
	assert.Equal(t, naming.TargetAllocator(params.OtelCol.Name), actual.Name)
	assert.Equal(t, params.OtelCol.Namespace, actual.Namespace)
	assert.Equal(t, "targetallocation", actual.Spec.Endpoints[0].Port)
}
