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

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

func TestLabelsCommonSet(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-ns",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator:0.47.0",
		},
	}

	// test
	labels := Labels(otelcol, []string{})
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "0.47.0", labels["app.kubernetes.io/version"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-collector", labels["app.kubernetes.io/component"])
}

func TestLabelsTagUnset(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-ns",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator",
		},
	}

	// test
	labels := Labels(otelcol, []string{})
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "latest", labels["app.kubernetes.io/version"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-collector", labels["app.kubernetes.io/component"])
}

func TestLabelsPropagateDown(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"myapp": "mycomponent"},
		},
	}

	// test
	labels := Labels(otelcol, []string{})

	// verify
	assert.Len(t, labels, 6)
	assert.Equal(t, "mycomponent", labels["myapp"])
}

func TestLabelsFilter(t *testing.T) {
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"test.bar.io": "foo", "test.foo.io": "bar"},
		},
	}

	// This requires the filter to be in regex match form and not the other simpler wildcard one.
	labels := Labels(otelcol, []string{".*.bar.io"})

	// verify
	assert.Len(t, labels, 6)
	assert.NotContains(t, labels, "test.bar.io")
	assert.Equal(t, "bar", labels["test.foo.io"])
}

func TestSelectorLabels(t *testing.T) {
	// prepare
	expected := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "my-namespace.my-opentelemetry-collector",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
	}
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{Name: "my-opentelemetry-collector", Namespace: "my-namespace"},
	}

	// test
	result := SelectorLabels(otelcol)

	// verify
	assert.Equal(t, expected, result)
}
