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

package upgrade_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func TestRemoveConnectionDelay(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	existing := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `exporters:
  opencensus:
    compression: "on"
    reconnection_delay: 15
    num_workers: 123`,
		},
	}
	existing.Status.Version = "0.8.0"

	// sanity check
	require.Contains(t, existing.Spec.Config, "reconnection_delay")

	// test
	res, err := upgrade.ManagedInstance(context.Background(), upgrade.Params{
		Log: logger,
		Version: version.Get(),
		Client: nil,
	}, existing)
	assert.NoError(t, err)

	// verify
	assert.Contains(t, res.Spec.Config, "opencensus:")
	assert.Contains(t, res.Spec.Config, `compression: "on"`)
	assert.NotContains(t, res.Spec.Config, "reconnection_delay")
	assert.Contains(t, res.Spec.Config, "num_workers: 123")
	assert.Contains(t, res.Status.Messages[0], "upgrade to v0.9.0 removed the property reconnection_delay for exporter")
}
