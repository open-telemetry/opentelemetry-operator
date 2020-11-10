package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

func TestLabelsCommonSet(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-ns",
		},
	}

	// test
	labels := Labels(otelcol)
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
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
	labels := Labels(otelcol)

	// verify
	assert.Len(t, labels, 5)
	assert.Equal(t, "mycomponent", labels["myapp"])
}
