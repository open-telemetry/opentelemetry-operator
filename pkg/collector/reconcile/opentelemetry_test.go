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

package reconcile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

func TestSelf(t *testing.T) {
	t.Run("should add version to the status", func(t *testing.T) {
		instance := params().Instance
		createObjectIfNotExists(t, "test", &instance)
		err := Self(context.Background(), params())
		assert.NoError(t, err)

		actual := v1alpha1.OpenTelemetryCollector{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test"})
		assert.NoError(t, err)
		assert.True(t, exists)

		assert.Equal(t, actual.Status.Version, "0.0.0")

	})
}

func TestUpdateScaleSubResourceStatus(t *testing.T) {
	t.Run("should update the scale subresource status for the OpenTelemetry CR", func(t *testing.T) {
		params := paramsWithMode(v1alpha1.ModeDaemonSet)
		ctx := context.Background()
		cli := params.Client
		changed := params.Instance
		err := updateScaleSubResourceStatus(ctx, cli, &changed)

		assert.NoError(t, err)
		name := naming.Collector(changed)

		labels := collector.Labels(changed, []string{})
		labels["app.kubernetes.io/name"] = name
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: labels})
		assert.NoError(t, err)
		assert.Equal(t, selector.String(), changed.Status.Scale.Selector)
	})
}
