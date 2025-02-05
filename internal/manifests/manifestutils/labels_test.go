// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	collectorName      = "my-instance"
	collectorNamespace = "my-ns"
	taname             = "my-instance"
	tanamespace        = "my-ns"
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
		"app.kubernetes.io/name":       "my-opentelemetry-collector-targetallocator",
		"app.kubernetes.io/part-of":    "opentelemetry",
	}
	tainstance := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{Name: "my-opentelemetry-collector", Namespace: "my-namespace"},
	}

	// test
	result := TASelectorLabels(tainstance, "opentelemetry-collector")

	// verify
	assert.Equal(t, expected, result)
}

func TestLabelsTACommonSet(t *testing.T) {
	// prepare
	tainstance := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taname,
			Namespace: tanamespace,
		},
	}

	// test
	labels := Labels(tainstance.ObjectMeta, taname, tainstance.Spec.Image, "opentelemetry-targetallocator", nil)
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-targetallocator", labels["app.kubernetes.io/component"])
	assert.Equal(t, "latest", labels["app.kubernetes.io/version"])
	assert.Equal(t, taname, labels["app.kubernetes.io/name"])
}

func TestLabelsTAPropagateDown(t *testing.T) {
	// prepare
	tainstance := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"myapp":                  "mycomponent",
				"app.kubernetes.io/name": "test",
			},
		},
	}

	// test
	labels := Labels(tainstance.ObjectMeta, taname, tainstance.Spec.Image, "opentelemetry-targetallocator", nil)

	selectorLabels := TASelectorLabels(tainstance, "opentelemetry-targetallocator")

	// verify
	assert.Len(t, labels, 7)
	assert.Equal(t, "mycomponent", labels["myapp"])
	assert.Equal(t, "test", labels["app.kubernetes.io/name"])
	assert.Equal(t, "test", selectorLabels["app.kubernetes.io/name"])
}

func TestSelectorTALabels(t *testing.T) {
	// prepare
	tainstance := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taname,
			Namespace: tanamespace,
		},
	}

	// test
	labels := TASelectorLabels(tainstance, "opentelemetry-targetallocator")
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-targetallocator", labels["app.kubernetes.io/component"])
	assert.Equal(t, naming.TargetAllocator(tainstance.Name), labels["app.kubernetes.io/name"])
}
