package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

func TestServiceAccountNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	// test
	sa := ServiceAccountName(otelcol)

	// verify
	assert.Equal(t, "my-instance-collector", sa)
}

func TestServiceAccountOverride(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			ServiceAccount: "my-special-sa",
		},
	}

	// test
	sa := ServiceAccountName(otelcol)

	// verify
	assert.Equal(t, "my-special-sa", sa)
}
