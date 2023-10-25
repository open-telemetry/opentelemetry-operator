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

package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	collectorName      = "my-instance"
	collectorNamespace = "my-ns"
)

func TestLabelsCommonSet(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      collectorName,
			Namespace: collectorNamespace,
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator:0.47.0",
		},
	}

	// test
	labels := Labels(otelcol.ObjectMeta, collectorName, otelcol.Spec.Image, "opentelemetry-collector", []string{})
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "0.47.0", labels["app.kubernetes.io/version"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-collector", labels["app.kubernetes.io/component"])
}
func TestLabelsSha256Set(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      collectorName,
			Namespace: collectorNamespace,
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator@sha256:c6671841470b83007e0553cdadbc9d05f6cfe17b3ebe9733728dc4a579a5b532",
		},
	}

	// test
	labels := Labels(otelcol.ObjectMeta, collectorName, otelcol.Spec.Image, "opentelemetry-collector", []string{})
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "c6671841470b83007e0553cdadbc9d05f6cfe17b3ebe9733728dc4a579a5b53", labels["app.kubernetes.io/version"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-collector", labels["app.kubernetes.io/component"])

	// prepare
	otelcolTag := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      collectorName,
			Namespace: collectorNamespace,
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator:0.81.0@sha256:c6671841470b83007e0553cdadbc9d05f6cfe17b3ebe9733728dc4a579a5b532",
		},
	}

	// test
	labelsTag := Labels(otelcolTag.ObjectMeta, collectorName, otelcolTag.Spec.Image, "opentelemetry-collector", []string{})
	assert.Equal(t, "opentelemetry-operator", labelsTag["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labelsTag["app.kubernetes.io/instance"])
	assert.Equal(t, "0.81.0", labelsTag["app.kubernetes.io/version"])
	assert.Equal(t, "opentelemetry", labelsTag["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-collector", labelsTag["app.kubernetes.io/component"])
}
func TestLabelsTagUnset(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      collectorName,
			Namespace: collectorNamespace,
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator",
		},
	}

	// test
	labels := Labels(otelcol.ObjectMeta, collectorName, otelcol.Spec.Image, "opentelemetry-collector", []string{})
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
			Labels: map[string]string{
				"myapp":                  "mycomponent",
				"app.kubernetes.io/name": "test",
			},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator",
		},
	}

	// test
	labels := Labels(otelcol.ObjectMeta, collectorName, otelcol.Spec.Image, "opentelemetry-collector", []string{})

	// verify
	assert.Len(t, labels, 7)
	assert.Equal(t, "mycomponent", labels["myapp"])
	assert.Equal(t, "test", labels["app.kubernetes.io/name"])
}

func TestLabelsFilter(t *testing.T) {
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"test.bar.io": "foo", "test.foo.io": "bar"},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator",
		},
	}

	// This requires the filter to be in regex match form and not the other simpler wildcard one.
	labels := Labels(otelcol.ObjectMeta, collectorName, otelcol.Spec.Image, "opentelemetry-collector", []string{".*.bar.io"})

	// verify
	assert.Len(t, labels, 7)
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
	result := SelectorLabels(otelcol.ObjectMeta, "opentelemetry-collector")

	// verify
	assert.Equal(t, expected, result)
}
